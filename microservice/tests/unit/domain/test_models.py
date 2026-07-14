"""Unit tests for sinapsis_ai.domain.models.

Verifies entity construction, field access, validation and the AnalysisType enum
as specified in CHANGELOG v0.2.0.
No mocks, no infrastructure — pure domain tests.
"""

from datetime import UTC, datetime

import pytest

from sinapsis_ai.domain.models import (
    AnalysisRequest,
    AnalysisResult,
    AnalysisType,
    Artifact,
    InferenceOutput,
    ModelRef,
)

# ---------------------------------------------------------------------------
# Helpers / fixtures
# ---------------------------------------------------------------------------

_NOW = datetime(2026, 1, 1, 12, 0, 0, tzinfo=UTC)

_VALID_REQUEST_KWARGS = {
    "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "study_id": "study-001",
    "patient_ref": "patient-opaque-001",
    "analysis_type": AnalysisType.CT_SPLEEN_SEGMENTATION,
    "image_uri": "s3://bucket/scan.nii.gz",
    "requested_by": "doctor-opaque-001",
    "correlation_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "issued_at": _NOW,
}

_MODEL_REF = ModelRef(name="spleen_ct_segmentation", version="0.5.3")
_ARTIFACT = Artifact(type="segmentation_mask", uri="s3://bucket/mask.nii.gz")


# ---------------------------------------------------------------------------
# AnalysisType
# ---------------------------------------------------------------------------


def test_analysis_type_values() -> None:
    """CT_SPLEEN_SEGMENTATION value is 'ct_spleen_segmentation' (snake_case)."""
    assert AnalysisType.CT_SPLEEN_SEGMENTATION == "ct_spleen_segmentation"
    assert AnalysisType.CT_SPLEEN_SEGMENTATION.value == "ct_spleen_segmentation"


def test_analysis_type_is_str() -> None:
    """AnalysisType inherits from str for direct use in string contexts."""
    assert isinstance(AnalysisType.CT_SPLEEN_SEGMENTATION, str)


# ---------------------------------------------------------------------------
# AnalysisRequest
# ---------------------------------------------------------------------------


def test_analysis_request_construction() -> None:
    """Valid kwargs produce a correct AnalysisRequest with accessible fields."""
    req = AnalysisRequest(**_VALID_REQUEST_KWARGS)

    assert req.request_id == "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    assert req.study_id == "study-001"
    assert req.patient_ref == "patient-opaque-001"
    assert req.analysis_type == AnalysisType.CT_SPLEEN_SEGMENTATION
    assert req.image_uri == "s3://bucket/scan.nii.gz"
    assert req.requested_by == "doctor-opaque-001"
    assert req.issued_at == _NOW


def test_analysis_request_is_frozen() -> None:
    """AnalysisRequest is immutable (frozen dataclass)."""
    req = AnalysisRequest(**_VALID_REQUEST_KWARGS)
    with pytest.raises((AttributeError, TypeError)):
        req.study_id = "modified"  # type: ignore[misc]


def test_analysis_request_rejects_empty_image_uri() -> None:
    """image_uri='' raises ValueError."""
    kwargs = {**_VALID_REQUEST_KWARGS, "image_uri": ""}
    with pytest.raises(ValueError, match="image_uri"):
        AnalysisRequest(**kwargs)


def test_analysis_request_rejects_empty_request_id() -> None:
    """request_id='' raises ValueError."""
    kwargs = {**_VALID_REQUEST_KWARGS, "request_id": ""}
    with pytest.raises(ValueError, match="request_id"):
        AnalysisRequest(**kwargs)


# ---------------------------------------------------------------------------
# ModelRef and Artifact
# ---------------------------------------------------------------------------


def test_model_ref_and_artifact_construction() -> None:
    """ModelRef and Artifact are constructable and fields are accessible."""
    ref = ModelRef(name="spleen_ct_segmentation", version="0.5.3")
    artifact = Artifact(type="segmentation_mask", uri="s3://bucket/mask.nii.gz")

    assert ref.name == "spleen_ct_segmentation"
    assert ref.version == "0.5.3"
    assert artifact.type == "segmentation_mask"
    assert artifact.uri == "s3://bucket/mask.nii.gz"


def test_model_ref_empty_version_allowed() -> None:
    """ModelRef with empty version is valid (some bundles lack semantic versions)."""
    ref = ModelRef(name="some_bundle", version="")
    assert ref.version == ""


def test_model_ref_local_path_defaults_to_empty() -> None:
    """ModelRef without local_path is valid (backwards compatible construction)."""
    ref = ModelRef(name="spleen_ct_segmentation", version="0.5.3")
    assert ref.local_path == ""


def test_model_ref_with_local_path() -> None:
    """ModelRef carries the local cache path of the resolved bundle."""
    ref = ModelRef(
        name="spleen_ct_segmentation",
        version="0.5.3",
        local_path="/cache/spleen_ct_segmentation",
    )
    assert ref.local_path == "/cache/spleen_ct_segmentation"


def test_model_ref_is_frozen() -> None:
    """ModelRef is immutable (frozen dataclass)."""
    ref = ModelRef(name="some_bundle", version="1.0.0")
    with pytest.raises((AttributeError, TypeError)):
        ref.local_path = "/tmp/x"  # type: ignore[misc]


# ---------------------------------------------------------------------------
# InferenceOutput
# ---------------------------------------------------------------------------


def test_inference_output_construction() -> None:
    """InferenceOutput carries artifact paths by type and computed metrics."""
    output = InferenceOutput(
        artifact_paths={"segmentation_mask": "/tmp/work/mask.nii.gz"},
        metrics={"volume_ml": 210.4},
    )
    assert output.artifact_paths == {"segmentation_mask": "/tmp/work/mask.nii.gz"}
    assert output.metrics == {"volume_ml": 210.4}


def test_inference_output_defaults_to_empty() -> None:
    """InferenceOutput defaults to no artifacts and no metrics."""
    output = InferenceOutput()
    assert output.artifact_paths == {}
    assert output.metrics == {}


def test_inference_output_is_frozen() -> None:
    """InferenceOutput is immutable (frozen dataclass)."""
    output = InferenceOutput()
    with pytest.raises((AttributeError, TypeError)):
        output.metrics = {"x": 1.0}  # type: ignore[misc]


# ---------------------------------------------------------------------------
# AnalysisResult
# ---------------------------------------------------------------------------


def test_analysis_result_succeeded() -> None:
    """AnalysisResult with status='succeeded' has error=None."""
    result = AnalysisResult(
        request_id="req-001",
        study_id="study-001",
        status="succeeded",
        model=_MODEL_REF,
        artifacts=[_ARTIFACT],
        metrics={"volume_ml": 210.4},
        processed_at=_NOW,
        duration_ms=42000,
    )

    assert result.status == "succeeded"
    assert result.error is None
    assert result.artifacts == [_ARTIFACT]
    assert result.metrics == {"volume_ml": 210.4}
    assert result.duration_ms == 42000


def test_analysis_result_failed() -> None:
    """AnalysisResult with status='failed' carries an error block."""
    error_block = {"code": "INFERENCE_ERROR", "message": "model exploded"}
    result = AnalysisResult(
        request_id="req-001",
        study_id="study-001",
        status="failed",
        model=_MODEL_REF,
        artifacts=[],
        metrics={},
        error=error_block,
        processed_at=_NOW,
        duration_ms=100,
    )

    assert result.status == "failed"
    assert result.error == error_block
    assert result.artifacts == []


def test_analysis_result_rejects_invalid_status() -> None:
    """status other than 'succeeded'/'failed' raises ValueError."""
    with pytest.raises(ValueError, match="status"):
        AnalysisResult(
            request_id="req-001",
            study_id="study-001",
            status="pending",  # invalid
            model=_MODEL_REF,
        )


def test_analysis_result_rejects_error_on_succeeded() -> None:
    """error must be None when status='succeeded'."""
    with pytest.raises(ValueError, match="error"):
        AnalysisResult(
            request_id="req-001",
            study_id="study-001",
            status="succeeded",
            model=_MODEL_REF,
            error={"code": "X", "message": "Y"},
        )


def test_analysis_result_empty_artifacts_and_metrics_defaults() -> None:
    """artifacts and metrics default to empty collections."""
    result = AnalysisResult(
        request_id="req-001",
        study_id="study-001",
        status="succeeded",
        model=_MODEL_REF,
    )
    assert result.artifacts == []
    assert result.metrics == {}
