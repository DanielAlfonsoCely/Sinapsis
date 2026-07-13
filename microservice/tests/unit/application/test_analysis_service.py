"""Unit tests for sinapsis_ai.application.analysis_service.

The AnalysisService is exercised with injected test doubles for all four
ports (bundle resolver, image store, inference engine, publisher) — no MONAI,
no torch, no RabbitMQ (CHANGELOG v0.4.0).
"""

from __future__ import annotations

from datetime import UTC, datetime

import pytest

from sinapsis_ai.application.analysis_service import AnalysisService
from sinapsis_ai.domain.errors import (
    BundleResolutionError,
    ImageAccessError,
    InferenceError,
)
from sinapsis_ai.domain.models import (
    AnalysisRequest,
    AnalysisResult,
    AnalysisType,
    InferenceOutput,
    ModelRef,
)

# ---------------------------------------------------------------------------
# Test doubles for the application ports
# ---------------------------------------------------------------------------

_MODEL_REF = ModelRef(
    name="spleen_ct_segmentation", version="0.5.3", local_path="/cache/spleen"
)

_REQUEST = AnalysisRequest(
    request_id="req-001",
    study_id="study-001",
    patient_ref="patient-opaque-001",
    analysis_type=AnalysisType.CT_SPLEEN_SEGMENTATION,
    image_uri="file:///data/scan.nii.gz",
    requested_by="doctor-opaque-001",
    correlation_id="corr-001",
    issued_at=datetime(2026, 1, 1, 12, 0, 0, tzinfo=UTC),
)


class StubBundleResolver:
    def __init__(self, error: Exception | None = None) -> None:
        self.calls: list[AnalysisType] = []
        self._error = error

    def resolve(self, analysis_type: AnalysisType) -> ModelRef:
        self.calls.append(analysis_type)
        if self._error is not None:
            raise self._error
        return _MODEL_REF


class StubImageStore:
    def __init__(
        self,
        fetch_error: Exception | None = None,
        save_error: Exception | None = None,
    ) -> None:
        self.fetch_calls: list[str] = []
        self.save_calls: list[tuple[str, str, str]] = []
        self._fetch_error = fetch_error
        self._save_error = save_error

    def fetch(self, image_uri: str) -> str:
        self.fetch_calls.append(image_uri)
        if self._fetch_error is not None:
            raise self._fetch_error
        return "/work/input/scan.nii.gz"

    def save_artifact(self, study_id: str, artifact_type: str, local_path: str) -> str:
        self.save_calls.append((study_id, artifact_type, local_path))
        if self._save_error is not None:
            raise self._save_error
        return f"file:///artifacts/{study_id}/{artifact_type}.nii.gz"


class StubInferenceEngine:
    def __init__(
        self,
        output: InferenceOutput | None = None,
        error: Exception | None = None,
    ) -> None:
        self.calls: list[tuple[ModelRef, str]] = []
        self._output = output if output is not None else InferenceOutput()
        self._error = error

    def run(
        self,
        model: ModelRef,
        image_path: str,
        analysis_type: object = None,
    ) -> InferenceOutput:
        self.calls.append((model, image_path))
        if self._error is not None:
            raise self._error
        return self._output


class SpyPublisher:
    def __init__(self) -> None:
        self.published: list[AnalysisResult] = []

    def publish(self, result: AnalysisResult) -> None:
        self.published.append(result)


def _make_service(
    resolver: StubBundleResolver | None = None,
    store: StubImageStore | None = None,
    engine: StubInferenceEngine | None = None,
) -> tuple[
    AnalysisService,
    StubBundleResolver,
    StubImageStore,
    StubInferenceEngine,
    SpyPublisher,
]:
    resolver = resolver if resolver is not None else StubBundleResolver()
    store = store if store is not None else StubImageStore()
    engine = engine if engine is not None else StubInferenceEngine()
    publisher = SpyPublisher()
    service = AnalysisService(
        bundle_resolver=resolver,
        image_store=store,
        inference_engine=engine,
        publisher=publisher,
    )
    return service, resolver, store, engine, publisher


# ---------------------------------------------------------------------------
# HU1 — happy path orchestration
# ---------------------------------------------------------------------------


def test_run_analysis_publishes_result() -> None:
    """The service orchestrates the flow and publishes the succeeded result."""
    engine = StubInferenceEngine(
        output=InferenceOutput(
            artifact_paths={"segmentation_mask": "/work/out/mask.nii.gz"},
            metrics={"volume_ml": 210.4},
        )
    )
    service, resolver, store, engine, publisher = _make_service(engine=engine)

    result = service.run_analysis(_REQUEST)

    # Orchestration sequence hit every collaborator with the right inputs.
    assert resolver.calls == [AnalysisType.CT_SPLEEN_SEGMENTATION]
    assert store.fetch_calls == ["file:///data/scan.nii.gz"]
    assert engine.calls == [(_MODEL_REF, "/work/input/scan.nii.gz")]
    # The published result is the returned result.
    assert publisher.published == [result]
    assert result.status == "succeeded"
    assert result.request_id == "req-001"
    assert result.study_id == "study-001"
    assert result.model == _MODEL_REF
    assert result.error is None
    assert result.duration_ms > 0
    assert result.processed_at.tzinfo is not None


def test_analysis_service_builds_metrics_and_artifacts() -> None:
    """Metrics come from the InferenceOutput; artifacts carry store URIs."""
    engine = StubInferenceEngine(
        output=InferenceOutput(
            artifact_paths={"segmentation_mask": "/work/out/mask.nii.gz"},
            metrics={"volume_ml": 210.4, "dice_hint": 0.93},
        )
    )
    service, _, store, _, publisher = _make_service(engine=engine)

    result = service.run_analysis(_REQUEST)

    assert store.save_calls == [
        ("study-001", "segmentation_mask", "/work/out/mask.nii.gz")
    ]
    assert [(a.type, a.uri) for a in result.artifacts] == [
        ("segmentation_mask", "file:///artifacts/study-001/segmentation_mask.nii.gz")
    ]
    assert result.metrics == {"volume_ml": 210.4, "dice_hint": 0.93}
    assert publisher.published == [result]


# ---------------------------------------------------------------------------
# HU1 — inference failure produces a published failed result
# ---------------------------------------------------------------------------


def test_inference_error_publishes_failed_result() -> None:
    """InferenceError → failed result published, normal return (no raise)."""
    engine = StubInferenceEngine(error=InferenceError("model exploded"))
    service, _, store, _, publisher = _make_service(engine=engine)

    result = service.run_analysis(_REQUEST)  # must NOT raise

    assert result.status == "failed"
    assert result.error == {"code": "INFERENCE_ERROR", "message": "model exploded"}
    assert result.model == _MODEL_REF  # traceability: resolved model preserved
    assert result.artifacts == []
    assert result.metrics == {}
    assert result.duration_ms > 0
    assert publisher.published == [result]
    assert store.save_calls == []  # nothing persisted on failure


# ---------------------------------------------------------------------------
# HU1 — pre-inference errors propagate without publishing
# ---------------------------------------------------------------------------


def test_bundle_resolution_error_propagates_without_publish() -> None:
    """BundleResolutionError propagates; nothing is published (RF-005)."""
    resolver = StubBundleResolver(error=BundleResolutionError("download failed"))
    service, _, store, engine, publisher = _make_service(resolver=resolver)

    with pytest.raises(BundleResolutionError):
        service.run_analysis(_REQUEST)

    assert publisher.published == []
    assert store.fetch_calls == []
    assert engine.calls == []


def test_image_access_error_propagates_without_publish() -> None:
    """ImageAccessError on fetch propagates; nothing is published (RF-005)."""
    store = StubImageStore(fetch_error=ImageAccessError("uri not found"))
    service, _, store, engine, publisher = _make_service(store=store)

    with pytest.raises(ImageAccessError):
        service.run_analysis(_REQUEST)

    assert publisher.published == []
    assert engine.calls == []


def test_artifact_save_error_propagates_without_publish() -> None:
    """ImageAccessError while persisting an artifact propagates (edge case)."""
    engine = StubInferenceEngine(
        output=InferenceOutput(artifact_paths={"segmentation_mask": "/out/m.nii.gz"})
    )
    store = StubImageStore(save_error=ImageAccessError("storage unavailable"))
    service, _, _, _, publisher = _make_service(store=store, engine=engine)

    with pytest.raises(ImageAccessError):
        service.run_analysis(_REQUEST)

    assert publisher.published == []  # no partial result published


# ---------------------------------------------------------------------------
# HU4 — structured metrics in logs
# ---------------------------------------------------------------------------


def test_run_analysis_logs_structured_metrics_on_success(
    caplog: pytest.LogCaptureFixture,
) -> None:
    """Successful analysis logs duration_ms, status, artifacts_count, model_name."""
    import logging

    engine = StubInferenceEngine(
        output=InferenceOutput(
            artifact_paths={"segmentation_mask": "/out/mask.nii.gz"},
            metrics={"volume_ml": 210.4},
        )
    )
    service, _, store, _, publisher = _make_service(engine=engine)

    with caplog.at_level(
        logging.INFO, logger="sinapsis_ai.application.analysis_service"
    ):
        service.run_analysis(_REQUEST)

    success_records = [r for r in caplog.records if "succeeded" in r.message]
    assert success_records, "Expected a log record with 'succeeded'"
    msg = success_records[0].message
    assert "duration_ms" in msg
    assert "artifacts_count" in msg
    assert "model_name" in msg or "spleen" in msg


def test_run_analysis_logs_structured_metrics_on_failure(
    caplog: pytest.LogCaptureFixture,
) -> None:
    """Failed analysis (InferenceError) logs status=failed and error_code."""
    import logging

    engine = StubInferenceEngine(error=InferenceError("cuda oom"))
    service, _, _, _, _ = _make_service(engine=engine)

    with caplog.at_level(
        logging.INFO, logger="sinapsis_ai.application.analysis_service"
    ):
        service.run_analysis(_REQUEST)

    failure_records = [r for r in caplog.records if "failed" in r.message]
    assert failure_records, "Expected a log record with 'failed'"
    msg = failure_records[0].message
    assert "error_code" in msg or "INFERENCE_ERROR" in msg
