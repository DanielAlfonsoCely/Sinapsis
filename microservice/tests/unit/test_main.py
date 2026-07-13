"""Unit tests for graceful shutdown behaviour in main.py.

Tests the _shutdown logic indirectly by exercising the Consumer.inflight
property and the shutdown wait loop that main._shutdown implements.
No real pika connection or RabbitMQ broker is used.
"""

from __future__ import annotations

import threading
import time
from unittest.mock import MagicMock

from sinapsis_ai.main import _declare_topology
from sinapsis_ai.presentation.consumer import Consumer
from sinapsis_ai.presentation.retry_policy import RetryPolicy

# ---------------------------------------------------------------------------
# Tests for Consumer.inflight (HU3 - property used by shutdown logic)
# ---------------------------------------------------------------------------


def test_consumer_inflight_false_when_idle() -> None:
    """Consumer.inflight is False before any message is processed."""
    mock_channel = MagicMock()
    mock_service = MagicMock()
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )
    assert consumer.inflight is False


def test_consumer_inflight_true_during_processing() -> None:
    """Consumer.inflight is True while _on_message is executing."""
    mock_channel = MagicMock()
    mock_service = MagicMock()

    inflight_during: list[bool] = []

    def _slow_run_analysis(request: object) -> object:  # noqa: ARG001
        # Capture inflight state mid-processing
        inflight_during.append(consumer.inflight)
        return MagicMock(request_id="r", status="succeeded")

    mock_service.run_analysis.side_effect = _slow_run_analysis

    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )

    from pathlib import Path

    body = (
        Path(__file__).parent.parent / "fixtures" / "sample_request.json"
    ).read_bytes()

    method = MagicMock()
    method.delivery_tag = 1
    consumer._on_message(mock_channel, method, MagicMock(), body)

    assert inflight_during == [True], "inflight should be True during processing"
    assert consumer.inflight is False, "inflight should be False after processing"


def test_consumer_inflight_false_after_processing_completes() -> None:
    """Consumer.inflight resets to False after successful processing."""
    mock_channel = MagicMock()
    mock_service = MagicMock()
    mock_service.run_analysis.return_value = MagicMock(
        request_id="r", status="succeeded"
    )
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )

    body = (
        __import__("pathlib").Path(__file__).parent.parent
        / "fixtures"
        / "sample_request.json"
    ).read_bytes()
    method = MagicMock()
    method.delivery_tag = 1
    consumer._on_message(mock_channel, method, MagicMock(), body)

    assert consumer.inflight is False


# ---------------------------------------------------------------------------
# Tests for shutdown wait loop logic (HU3)
# ---------------------------------------------------------------------------


def test_graceful_shutdown_waits_for_inflight_message() -> None:
    """Shutdown loop waits until consumer.inflight becomes False."""
    mock_channel = MagicMock()
    mock_service = MagicMock()
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )

    # Simulate a message in flight
    consumer._inflight = True

    # After 0.1s, clear the inflight flag in a background thread
    def _clear_inflight() -> None:
        time.sleep(0.1)
        consumer._inflight = False

    t = threading.Thread(target=_clear_inflight, daemon=True)
    t.start()

    # Replicate the shutdown wait loop from main._shutdown
    timeout_s = 2
    deadline = time.monotonic() + timeout_s
    while consumer.inflight and time.monotonic() < deadline:
        time.sleep(0.01)

    t.join(timeout=1)
    assert not consumer.inflight, "inflight should have been cleared"


def test_graceful_shutdown_immediate_when_idle() -> None:
    """Shutdown loop exits immediately when consumer is not in flight."""
    mock_channel = MagicMock()
    mock_service = MagicMock()
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )

    assert consumer.inflight is False

    start = time.monotonic()
    timeout_s = 5
    deadline = time.monotonic() + timeout_s
    while consumer.inflight and time.monotonic() < deadline:
        time.sleep(0.01)
    elapsed = time.monotonic() - start

    # Should exit the loop almost instantly (well under 0.5 s)
    assert elapsed < 0.5


def test_graceful_shutdown_respects_timeout() -> None:
    """Shutdown loop exits after timeout even if inflight stays True."""
    mock_channel = MagicMock()
    mock_service = MagicMock()
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=RetryPolicy(max_retries=1, backoff_s=0.0),
    )

    consumer._inflight = True  # Never cleared
    timeout_s = 0.2

    start = time.monotonic()
    deadline = time.monotonic() + timeout_s
    while consumer.inflight and time.monotonic() < deadline:
        time.sleep(0.01)
    elapsed = time.monotonic() - start

    # Loop must have exited due to timeout, not because inflight cleared
    assert consumer.inflight is True, "inflight was never cleared in this test"
    assert elapsed >= timeout_s * 0.8  # exited around the timeout boundary


# ---------------------------------------------------------------------------
# Tests for _declare_topology
# ---------------------------------------------------------------------------


def test_declare_topology_calls_exchange_and_queue_declare() -> None:
    """_declare_topology declares DLX, result exchange and request queue."""
    mock_channel = MagicMock()

    _declare_topology(
        channel=mock_channel,
        request_queue="ai.analysis.requests",
        dlx_name="sinapsis.ai.dlx",
        result_exchange="sinapsis.ai",
    )

    assert mock_channel.exchange_declare.call_count == 2
    calls = {
        c.kwargs["exchange"]: c.kwargs
        for c in mock_channel.exchange_declare.call_args_list
    }
    assert calls["sinapsis.ai.dlx"]["exchange_type"] == "fanout"
    assert calls["sinapsis.ai"]["exchange_type"] == "direct"
    mock_channel.queue_declare.assert_called_once_with(
        queue="ai.analysis.requests",
        durable=True,
        arguments={"x-dead-letter-exchange": "sinapsis.ai.dlx"},
    )
