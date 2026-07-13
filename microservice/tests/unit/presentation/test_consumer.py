"""Unit tests for presentation.consumer (Consumer).

The pika channel and AnalysisService are fully mocked — no real RabbitMQ
connection is opened and no real inference is executed.
"""

from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from sinapsis_ai.domain.errors import BundleResolutionError, ImageAccessError
from sinapsis_ai.domain.models import AnalysisResult, ModelRef
from sinapsis_ai.presentation.consumer import Consumer
from sinapsis_ai.presentation.retry_policy import RetryPolicy

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

FIXTURES = Path(__file__).parent.parent.parent / "fixtures"


def _load_sample_request() -> bytes:
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


def _make_failed_result() -> AnalysisResult:
    return AnalysisResult(
        request_id="a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        study_id="study-opaque-001",
        status="failed",
        model=ModelRef(name="spleen_ct_segmentation", version="0.5.3"),
        artifacts=[],
        metrics={},
        error={"code": "INFERENCE_ERROR", "message": "boom"},
        processed_at=datetime(2026, 1, 1, 12, 0, 42, tzinfo=UTC),
        duration_ms=1000,
    )


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture()
def mock_channel() -> MagicMock:
    ch = MagicMock()
    ch.basic_ack = MagicMock()
    ch.basic_nack = MagicMock()
    return ch


@pytest.fixture()
def mock_service() -> MagicMock:
    return MagicMock()


def _no_retry_policy() -> RetryPolicy:
    """RetryPolicy with zero backoff and max_retries=1 (effectively no retry)."""
    return RetryPolicy(max_retries=1, backoff_s=0.0)


def _retry_policy(max_retries: int = 3) -> RetryPolicy:
    return RetryPolicy(max_retries=max_retries, backoff_s=0.0)


@pytest.fixture()
def consumer(mock_channel: MagicMock, mock_service: MagicMock) -> Consumer:
    return Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=_no_retry_policy(),
    )


def _delivery_tag() -> MagicMock:
    """Fake delivery tag (any object works; pika uses it opaquely)."""
    tag = MagicMock()
    tag.__int__ = lambda self: 1  # type: ignore[method-assign]
    return tag


def _method(delivery_tag: MagicMock | None = None) -> MagicMock:
    m = MagicMock()
    m.delivery_tag = delivery_tag or _delivery_tag()
    return m


# ---------------------------------------------------------------------------
# Tests — HU1
# ---------------------------------------------------------------------------


def test_consumer_valid_message_acks_and_publishes_result(
    consumer: Consumer,
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """Valid message → service called → basic_ack emitted."""
    mock_service.run_analysis.return_value = _make_succeeded_result()
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), _load_sample_request())

    mock_service.run_analysis.assert_called_once()
    mock_channel.basic_ack.assert_called_once_with(delivery_tag=method.delivery_tag)
    mock_channel.basic_nack.assert_not_called()


def test_consumer_invalid_message_deadletters(
    consumer: Consumer,
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """Payload missing image_uri → nack(requeue=False), no service call."""
    payload = json.dumps(
        {
            "request_id": "req-bad",
            "study_id": "study-x",
            "patient_ref": "p-ref",
            "analysis_type": "ct_spleen_segmentation",
            "image_uri": "",  # empty — triggers InvalidMessageError
            "requested_by": "doc-1",
            "correlation_id": "corr-1",
            "issued_at": "2026-01-01T12:00:00Z",
        }
    ).encode()
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), payload)

    mock_service.run_analysis.assert_not_called()
    mock_channel.basic_nack.assert_called_once_with(
        delivery_tag=method.delivery_tag, requeue=False
    )
    mock_channel.basic_ack.assert_not_called()


def test_consumer_unknown_analysis_type_deadletters(
    consumer: Consumer,
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """Unknown analysis_type → UnknownAnalysisTypeError → nack(requeue=False)."""
    payload = json.dumps(
        {
            "request_id": "req-unk",
            "study_id": "study-x",
            "patient_ref": "p-ref",
            "analysis_type": "non_existent_type",
            "image_uri": "s3://bucket/scan.nii.gz",
            "requested_by": "doc-1",
            "correlation_id": "corr-1",
            "issued_at": "2026-01-01T12:00:00Z",
        }
    ).encode()
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), payload)

    mock_service.run_analysis.assert_not_called()
    mock_channel.basic_nack.assert_called_once_with(
        delivery_tag=method.delivery_tag, requeue=False
    )
    mock_channel.basic_ack.assert_not_called()


def test_consumer_inference_error_publishes_failed_result_and_acks(
    consumer: Consumer,
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """InferenceError handled inside service → service returns failed result → ack.

    The AnalysisService (v0.4.0) catches InferenceError, builds a failed
    AnalysisResult, publishes it and returns it. From the consumer's perspective
    run_analysis() returns normally — so the consumer must ack.
    """
    mock_service.run_analysis.return_value = _make_failed_result()
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), _load_sample_request())

    mock_service.run_analysis.assert_called_once()
    mock_channel.basic_ack.assert_called_once_with(delivery_tag=method.delivery_tag)
    mock_channel.basic_nack.assert_not_called()


def test_consumer_malformed_json_deadletters(
    consumer: Consumer,
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """Non-JSON body → InvalidMessageError → nack(requeue=False)."""
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), b"not-json{{{")

    mock_service.run_analysis.assert_not_called()
    mock_channel.basic_nack.assert_called_once_with(
        delivery_tag=method.delivery_tag, requeue=False
    )


def test_consumer_unexpected_error_deadletters(
    consumer: Consumer,
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """Unexpected exception from service → nack(requeue=False), no crash."""
    mock_service.run_analysis.side_effect = RuntimeError("unexpected")
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), _load_sample_request())

    mock_channel.basic_nack.assert_called_once_with(
        delivery_tag=method.delivery_tag, requeue=False
    )
    mock_channel.basic_ack.assert_not_called()


def test_consumer_start_consuming_sets_prefetch_and_registers_callback(
    consumer: Consumer,
    mock_channel: MagicMock,
) -> None:
    """start_consuming sets qos prefetch and calls basic_consume."""
    consumer.start_consuming(queue="ai.analysis.requests", prefetch=1)

    mock_channel.basic_qos.assert_called_once_with(prefetch_count=1)
    mock_channel.basic_consume.assert_called_once()
    call_kwargs = mock_channel.basic_consume.call_args
    assert (
        call_kwargs.kwargs.get("queue") == "ai.analysis.requests"
        or call_kwargs.args[0] == "ai.analysis.requests"
    )


# ---------------------------------------------------------------------------
# Tests — HU1 (retry policy integration in consumer)
# ---------------------------------------------------------------------------


def test_consumer_transient_error_retries_and_succeeds(
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """BundleResolutionError twice then success → ack, service invoked 3 times."""
    mock_service.run_analysis.side_effect = [
        BundleResolutionError("net"),
        BundleResolutionError("net"),
        _make_succeeded_result(),
    ]
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=_retry_policy(max_retries=3),
    )
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), _load_sample_request())

    assert mock_service.run_analysis.call_count == 3
    mock_channel.basic_ack.assert_called_once_with(delivery_tag=method.delivery_tag)
    mock_channel.basic_nack.assert_not_called()


def test_consumer_transient_error_exhausts_retries_deadletters(
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """ImageAccessError always → nack after max_retries attempts."""
    mock_service.run_analysis.side_effect = ImageAccessError("s3 down")
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=_retry_policy(max_retries=3),
    )
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), _load_sample_request())

    assert mock_service.run_analysis.call_count == 3
    mock_channel.basic_nack.assert_called_once_with(
        delivery_tag=method.delivery_tag, requeue=False
    )
    mock_channel.basic_ack.assert_not_called()


def test_consumer_non_transient_error_deadletters_immediately(
    mock_channel: MagicMock,
    mock_service: MagicMock,
) -> None:
    """UnknownAnalysisTypeError → nack at validation stage, no service invocation."""
    payload = json.dumps(
        {
            "request_id": "req-x",
            "study_id": "s",
            "patient_ref": "p",
            "analysis_type": "bad_type",
            "image_uri": "s3://bucket/scan.nii.gz",
            "requested_by": "doc",
            "correlation_id": "c",
            "issued_at": "2026-01-01T12:00:00Z",
        }
    ).encode()
    consumer = Consumer(
        service=mock_service,
        channel=mock_channel,
        retry_policy=_retry_policy(max_retries=3),
    )
    method = _method()

    consumer._on_message(mock_channel, method, MagicMock(), payload)

    mock_service.run_analysis.assert_not_called()
    mock_channel.basic_nack.assert_called_once_with(
        delivery_tag=method.delivery_tag, requeue=False
    )
    mock_channel.basic_ack.assert_not_called()
