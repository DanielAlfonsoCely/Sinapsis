"""Unit tests for sinapsis_ai.presentation.schemas.

Verifies DTO validation, domain translation and result serialisation as specified
in CHANGELOG v0.2.0. Uses the sample_request.json fixture.
No RabbitMQ, no MONAI — pure unit tests.
"""

from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path

import pytest

from sinapsis_ai.domain.errors import InvalidMessageError, UnknownAnalysisTypeError
from sinapsis_ai.domain.models import (
    AnalysisResult,
    AnalysisType,
    Artifact,
    ModelRef,
)
from sinapsis_ai.presentation.schemas import parse_request, serialise_result

# ---------------------------------------------------------------------------
# Fixtures / helpers
# ---------------------------------------------------------------------------

_FIXTURES_DIR = Path(__file__).parent.parent.parent / "fixtures"
_SAMPLE_REQUEST_PATH = _FIXTURES_DIR / "sample_request.json"

_MODEL_REF = ModelRef(name="spleen_ct_segmentation", version="0.5.3")
_NOW = datetime(2026, 1, 1, 12, 0, 42, tzinfo=UTC)


def _load_sample() -> dict:  # type: ignore[type-arg]
    return json.loads(_SAMPLE_REQUEST_PATH.read_text())


# ---------------------------------------------------------------------------
# HU3 tests — request parsing
# ---------------------------------------------------------------------------


def test_request_schema_valid_payload() -> None:
    """sample_request.json is parsed into a valid AnalysisRequest domain entity."""
    raw = _SAMPLE_REQUEST_PATH.read_bytes()
    request = parse_request(raw)

    assert request.request_id == "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    assert request.study_id == "study-opaque-001"
    assert request.analysis_type == AnalysisType.CT_SPLEEN_SEGMENTATION
    assert request.image_uri == "s3://sinapsis-images/studies/001/scan.nii.gz"
    assert isinstance(request.issued_at, datetime)


def test_request_schema_rejects_missing_image_uri() -> None:
    """A request without image_uri raises InvalidMessageError."""
    data = _load_sample()
    del data["image_uri"]

    with pytest.raises(InvalidMessageError):
        parse_request(json.dumps(data))


def test_request_schema_rejects_empty_image_uri() -> None:
    """A request with image_uri='' raises InvalidMessageError."""
    data = _load_sample()
    data["image_uri"] = ""

    with pytest.raises(InvalidMessageError):
        parse_request(json.dumps(data))


def test_request_schema_rejects_unknown_analysis_type() -> None:
    """An unknown analysis_type raises UnknownAnalysisTypeError (not
    InvalidMessageError)."""
    data = _load_sample()
    data["analysis_type"] = "unknown_type_xyz"

    with pytest.raises(UnknownAnalysisTypeError):
        parse_request(json.dumps(data))


def test_request_schema_rejects_invalid_json() -> None:
    """Malformed JSON raises InvalidMessageError."""
    with pytest.raises(InvalidMessageError):
        parse_request(b"{not valid json")


def test_request_schema_ignores_extra_fields() -> None:
    """Extra unknown fields in the JSON are silently ignored (forward compatibility)."""
    data = _load_sample()
    data["future_field"] = "some_value"

    request = parse_request(json.dumps(data))
    assert request.study_id == "study-opaque-001"


# ---------------------------------------------------------------------------
# HU3 tests — result serialisation
# ---------------------------------------------------------------------------


def test_result_schema_serializes_success() -> None:
    """AnalysisResult(status='succeeded') serialises to the contract JSON
    (error=null)."""
    result = AnalysisResult(
        request_id="req-001",
        study_id="study-001",
        status="succeeded",
        model=_MODEL_REF,
        artifacts=[Artifact(type="segmentation_mask", uri="s3://bucket/mask.nii.gz")],
        metrics={"volume_ml": 210.4},
        processed_at=_NOW,
        duration_ms=42000,
    )

    payload = serialise_result(result)

    assert payload["request_id"] == "req-001"
    assert payload["study_id"] == "study-001"
    assert payload["status"] == "succeeded"
    assert payload["model"] == {"name": "spleen_ct_segmentation", "version": "0.5.3"}
    assert payload["artifacts"] == [
        {"type": "segmentation_mask", "uri": "s3://bucket/mask.nii.gz"}
    ]
    assert payload["metrics"] == {"volume_ml": 210.4}
    assert payload["error"] is None
    assert payload["duration_ms"] == 42000
    assert "processed_at" in payload
    # Verify it is JSON-serialisable
    json.dumps(payload)


def test_result_schema_serializes_failure() -> None:
    """AnalysisResult(status='failed') serialises with the error block present."""
    error_block = {"code": "INFERENCE_ERROR", "message": "OOM during forward pass"}
    result = AnalysisResult(
        request_id="req-002",
        study_id="study-002",
        status="failed",
        model=_MODEL_REF,
        artifacts=[],
        metrics={},
        error=error_block,
        processed_at=_NOW,
        duration_ms=500,
    )

    payload = serialise_result(result)

    assert payload["status"] == "failed"
    assert payload["error"] == error_block
    assert payload["artifacts"] == []
    assert payload["metrics"] == {}
    # Verify it is JSON-serialisable
    json.dumps(payload)


def test_result_schema_model_excludes_local_path() -> None:
    """The serialised model block contains only name and version — local_path is
    internal and must never leak into the message contract (RF-004, v0.3.0)."""
    result = AnalysisResult(
        request_id="req-001",
        study_id="study-001",
        status="succeeded",
        model=ModelRef(
            name="spleen_ct_segmentation",
            version="0.5.3",
            local_path="/cache/spleen_ct_segmentation",
        ),
        processed_at=_NOW,
        duration_ms=0,
    )

    payload = serialise_result(result)

    assert payload["model"] == {"name": "spleen_ct_segmentation", "version": "0.5.3"}
    assert "local_path" not in payload["model"]


def test_result_schema_all_contract_fields_present() -> None:
    """The serialised dict contains all fields required by PROJECT_ARCHITECTURE §9."""
    result = AnalysisResult(
        request_id="req-001",
        study_id="study-001",
        status="succeeded",
        model=_MODEL_REF,
        processed_at=_NOW,
        duration_ms=0,
    )
    payload = serialise_result(result)

    required_keys = {
        "request_id",
        "study_id",
        "status",
        "model",
        "artifacts",
        "metrics",
        "error",
        "processed_at",
        "duration_ms",
    }
    assert required_keys.issubset(payload.keys())
