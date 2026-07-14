"""Unit tests for infrastructure.publisher (RabbitMQPublisher).

The pika channel is fully mocked — no real RabbitMQ connection is opened.
"""

from __future__ import annotations

import json
from datetime import UTC, datetime
from unittest.mock import MagicMock

import pika
import pytest

from sinapsis_ai.domain.models import AnalysisResult, Artifact, ModelRef
from sinapsis_ai.infrastructure.publisher import RabbitMQPublisher

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

EXCHANGE = "sinapsis.ai"
ROUTING_KEY = "ai.analysis.result"


@pytest.fixture()
def mock_channel() -> MagicMock:
    """A mock pika BlockingChannel."""
    return MagicMock()


@pytest.fixture()
def publisher(mock_channel: MagicMock) -> RabbitMQPublisher:
    return RabbitMQPublisher(
        channel=mock_channel,
        result_exchange=EXCHANGE,
        result_routing_key=ROUTING_KEY,
    )


@pytest.fixture()
def succeeded_result() -> AnalysisResult:
    return AnalysisResult(
        request_id="req-001",
        study_id="study-abc",
        status="succeeded",
        model=ModelRef(name="spleen_ct_segmentation", version="0.5.3"),
        artifacts=[Artifact(type="segmentation_mask", uri="s3://bucket/mask.nii.gz")],
        metrics={"volume_ml": 210.4},
        error=None,
        processed_at=datetime(2026, 1, 1, 12, 0, 42, tzinfo=UTC),
        duration_ms=42000,
    )


@pytest.fixture()
def failed_result() -> AnalysisResult:
    return AnalysisResult(
        request_id="req-002",
        study_id="study-xyz",
        status="failed",
        model=ModelRef(name="spleen_ct_segmentation", version="0.5.3"),
        artifacts=[],
        metrics={},
        error={"code": "INFERENCE_ERROR", "message": "CUDA out of memory"},
        processed_at=datetime(2026, 1, 1, 12, 1, 0, tzinfo=UTC),
        duration_ms=5000,
    )


# ---------------------------------------------------------------------------
# Tests — HU2
# ---------------------------------------------------------------------------


def test_publisher_publishes_persistent_to_result_exchange(
    publisher: RabbitMQPublisher,
    mock_channel: MagicMock,
    succeeded_result: AnalysisResult,
) -> None:
    """basic_publish is called with delivery_mode=2, correct exchange and routing key."""  # noqa: E501
    publisher.publish(succeeded_result)

    mock_channel.basic_publish.assert_called_once()
    _, kwargs = mock_channel.basic_publish.call_args

    assert kwargs["exchange"] == EXCHANGE
    assert kwargs["routing_key"] == ROUTING_KEY

    props: pika.BasicProperties = kwargs["properties"]
    assert props.delivery_mode == 2  # persistent

    body: dict = json.loads(kwargs["body"])
    assert body["request_id"] == "req-001"
    assert body["study_id"] == "study-abc"
    assert body["status"] == "succeeded"
    assert body["model"] == {"name": "spleen_ct_segmentation", "version": "0.5.3"}
    assert body["artifacts"] == [
        {"type": "segmentation_mask", "uri": "s3://bucket/mask.nii.gz"}
    ]
    assert body["metrics"] == {"volume_ml": 210.4}
    assert body["error"] is None
    assert body["duration_ms"] == 42000


def test_publisher_publishes_failed_result(
    publisher: RabbitMQPublisher,
    mock_channel: MagicMock,
    failed_result: AnalysisResult,
) -> None:
    """Failed result includes the error block in the published JSON."""
    publisher.publish(failed_result)

    mock_channel.basic_publish.assert_called_once()
    _, kwargs = mock_channel.basic_publish.call_args
    body: dict = json.loads(kwargs["body"])

    assert body["status"] == "failed"
    assert body["error"] == {"code": "INFERENCE_ERROR", "message": "CUDA out of memory"}
    assert body["artifacts"] == []
    assert body["metrics"] == {}


def test_publisher_body_is_utf8_bytes(
    publisher: RabbitMQPublisher,
    mock_channel: MagicMock,
    succeeded_result: AnalysisResult,
) -> None:
    """The body passed to basic_publish is bytes, not str."""
    publisher.publish(succeeded_result)
    _, kwargs = mock_channel.basic_publish.call_args
    assert isinstance(kwargs["body"], bytes)


def test_publisher_uses_configured_exchange_and_routing_key(
    mock_channel: MagicMock,
    succeeded_result: AnalysisResult,
) -> None:
    """Publisher respects the exchange and routing key passed at construction."""
    custom_publisher = RabbitMQPublisher(
        channel=mock_channel,
        result_exchange="custom.exchange",
        result_routing_key="custom.routing.key",
    )
    custom_publisher.publish(succeeded_result)
    _, kwargs = mock_channel.basic_publish.call_args
    assert kwargs["exchange"] == "custom.exchange"
    assert kwargs["routing_key"] == "custom.routing.key"
