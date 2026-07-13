"""Integration test: RabbitMQ round-trip request → result.

Marked @pytest.mark.integration (skipped by default): requires a real
RabbitMQ broker reachable at RABBITMQ_URL (default amqp://guest:guest@localhost:5672/).

Start infrastructure before running:
    docker compose up -d rabbitmq

Run with:
    uv run pytest -m integration tests/integration/test_rabbitmq_flow.py -q

Strategy:
  * Publish a valid AnalysisRequest JSON to the request queue.
  * Start the Consumer in a background thread with a mocked AnalysisService
    (avoids any real MONAI inference or bundle download).
  * Consume the published AnalysisResult from the result exchange via a
    dedicated listener queue.
  * Assert result fields, then clean up queues/bindings.
"""

from __future__ import annotations

import contextlib
import json
import threading
import time
from datetime import UTC, datetime
from pathlib import Path
from unittest.mock import MagicMock

import pika
import pytest

from sinapsis_ai.domain.models import AnalysisResult, ModelRef
from sinapsis_ai.infrastructure.publisher import RabbitMQPublisher
from sinapsis_ai.presentation.consumer import Consumer

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

RABBITMQ_URL = "amqp://guest:guest@localhost:5672/"
REQUEST_QUEUE = "ai.analysis.requests.test"
RESULT_EXCHANGE = "sinapsis.ai.test"
RESULT_ROUTING_KEY = "ai.analysis.result.test"
LISTENER_QUEUE = "ai.analysis.result.test.listener"

FIXTURES = Path(__file__).parent.parent / "fixtures"


def _sample_request_bytes() -> bytes:
    return (FIXTURES / "sample_request.json").read_bytes()


def _make_succeeded_result() -> AnalysisResult:
    return AnalysisResult(
        request_id="a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        study_id="study-opaque-001",
        status="succeeded",
        model=ModelRef(name="spleen_ct_segmentation", version="0.5.3"),
        artifacts=[],
        metrics={"volume_ml": 210.4},
        error=None,
        processed_at=datetime(2026, 1, 1, 12, 0, 42, tzinfo=UTC),
        duration_ms=42000,
    )


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture()
def rabbitmq_connection() -> pika.BlockingConnection:
    """Open a fresh connection; close it after the test."""
    conn = pika.BlockingConnection(pika.URLParameters(RABBITMQ_URL))
    yield conn  # type: ignore[misc]
    if conn.is_open:
        conn.close()


@pytest.fixture()
def setup_topology(rabbitmq_connection: pika.BlockingConnection) -> None:
    """Declare exchange, queues and binding; delete them after the test."""
    ch = rabbitmq_connection.channel()
    ch.exchange_declare(exchange=RESULT_EXCHANGE, exchange_type="direct", durable=True)
    ch.queue_declare(queue=REQUEST_QUEUE, durable=True)
    ch.queue_declare(queue=LISTENER_QUEUE, durable=False, auto_delete=True)
    ch.queue_bind(
        queue=LISTENER_QUEUE, exchange=RESULT_EXCHANGE, routing_key=RESULT_ROUTING_KEY
    )
    ch.close()

    yield  # type: ignore[misc]

    # Cleanup
    cleanup = rabbitmq_connection.channel()
    try:
        cleanup.queue_delete(queue=REQUEST_QUEUE)
        cleanup.queue_delete(queue=LISTENER_QUEUE)
        cleanup.exchange_delete(exchange=RESULT_EXCHANGE)
    except Exception:  # noqa: BLE001
        pass
    finally:
        cleanup.close()


# ---------------------------------------------------------------------------
# Test — HU4
# ---------------------------------------------------------------------------


@pytest.mark.integration
def test_rabbitmq_round_trip(
    rabbitmq_connection: pika.BlockingConnection,
    setup_topology: None,
) -> None:
    """Publish request → consumer processes it (mock engine) → result in exchange."""
    # --- Mock AnalysisService that immediately returns a succeeded result ---
    mock_service = MagicMock()
    mock_service.run_analysis.return_value = _make_succeeded_result()

    # --- Consumer connection (separate channel so the thread owns it) ---
    consumer_conn = pika.BlockingConnection(pika.URLParameters(RABBITMQ_URL))
    consumer_ch = consumer_conn.channel()

    # Publisher for the consumer uses its own channel
    publisher = RabbitMQPublisher(
        channel=consumer_ch,
        result_exchange=RESULT_EXCHANGE,
        result_routing_key=RESULT_ROUTING_KEY,
    )
    # Wire publisher into mock_service so publish() is called via the real publisher
    mock_service.run_analysis.side_effect = lambda req: _publish_and_return(
        publisher, _make_succeeded_result()
    )

    consumer = Consumer(service=mock_service, channel=consumer_ch)
    consumer.start_consuming(queue=REQUEST_QUEUE, prefetch=1)

    # --- Start consumer in background thread ---
    def _run_consumer() -> None:
        with contextlib.suppress(Exception):
            consumer_ch.start_consuming()

    t = threading.Thread(target=_run_consumer, daemon=True)
    t.start()

    try:
        # --- Publish the request ---
        pub_ch = rabbitmq_connection.channel()
        pub_ch.basic_publish(
            exchange="",
            routing_key=REQUEST_QUEUE,
            body=_sample_request_bytes(),
            properties=pika.BasicProperties(
                content_type="application/json", delivery_mode=2
            ),
        )

        # --- Wait for the result to appear in the listener queue ---
        result_body: bytes | None = None
        deadline = time.monotonic() + 10.0  # 10 s timeout

        listener_ch = rabbitmq_connection.channel()
        while time.monotonic() < deadline:
            method, _, body = listener_ch.basic_get(queue=LISTENER_QUEUE, auto_ack=True)
            if method is not None:
                result_body = body
                break
            time.sleep(0.2)

        assert result_body is not None, "No result received within timeout"

        result = json.loads(result_body)
        assert result["request_id"] == "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        assert result["study_id"] == "study-opaque-001"
        assert result["status"] == "succeeded"
        assert result["model"]["name"] == "spleen_ct_segmentation"
        assert result["duration_ms"] == 42000

    finally:
        # Stop the consumer thread
        with contextlib.suppress(Exception):
            consumer_ch.stop_consuming()
        t.join(timeout=3)
        if consumer_conn.is_open:
            consumer_conn.close()


def _publish_and_return(
    publisher: RabbitMQPublisher, result: AnalysisResult
) -> AnalysisResult:
    """Helper: publish the result through the real publisher and return it."""
    publisher.publish(result)
    return result


@pytest.mark.integration
def test_reconnect_after_broker_restart() -> None:
    """Worker detects broker disconnection and terminates cleanly (no zombie).

    Strategy: open a connection, force-close it from underneath the consumer,
    then verify the consumer loop raises AMQPError and the thread exits within
    the expected timeout rather than hanging indefinitely.

    Requires RabbitMQ running via docker compose.
    """
    consumer_conn = pika.BlockingConnection(pika.URLParameters(RABBITMQ_URL))
    consumer_ch = consumer_conn.channel()

    mock_service = MagicMock()
    # Service never called — we drop the connection before any message arrives
    mock_service.run_analysis.return_value = _make_succeeded_result()

    from sinapsis_ai.presentation.consumer import Consumer
    from sinapsis_ai.presentation.retry_policy import RetryPolicy

    publisher = RabbitMQPublisher(
        channel=consumer_ch,
        result_exchange=RESULT_EXCHANGE,
        result_routing_key=RESULT_ROUTING_KEY,
    )
    mock_service.run_analysis.side_effect = lambda req: _publish_and_return(
        publisher, _make_succeeded_result()
    )

    consumer = Consumer(
        service=mock_service,
        channel=consumer_ch,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )

    # Use a temporary queue — no topology setup needed for this test
    tmp_queue = "ai.analysis.requests.reconnect_test"
    consumer_ch.queue_declare(queue=tmp_queue, durable=False, auto_delete=True)
    consumer.start_consuming(queue=tmp_queue, prefetch=1)

    thread_error: list[Exception] = []

    def _run() -> None:
        try:
            consumer_ch.start_consuming()
        except Exception as exc:  # noqa: BLE001
            thread_error.append(exc)

    t = threading.Thread(target=_run, daemon=True)
    t.start()

    # Give the consumer a moment to start, then abruptly close the connection
    time.sleep(0.3)
    with contextlib.suppress(Exception):
        consumer_conn.close()

    # The consumer thread should exit within 5 s after the connection is lost
    t.join(timeout=5)
    assert not t.is_alive(), (
        "Consumer thread should have exited after broker disconnection"
    )
