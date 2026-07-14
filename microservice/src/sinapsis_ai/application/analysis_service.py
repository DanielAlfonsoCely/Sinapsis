"""Central use case: orchestrates the full analysis flow.

Application layer — depends only on the domain and on the ports defined in
application/ports.py. Never imports pika, monai, torch or infrastructure
modules; collaborators are injected by the composition root (main.py).

Flow (PROJECT_ARCHITECTURE.md §10):
    resolve bundle → fetch image → run inference → persist artifacts
    → build AnalysisResult → publish

Error policy (DESIGN.md §7):
  * InferenceError → build and publish a failed AnalysisResult, return normally
    (the consumer acks; the backend receives the failure — no lost traceability).
  * UnknownAnalysisTypeError / BundleResolutionError / ImageAccessError →
    propagate WITHOUT publishing (retry/dead-letter policy belongs to the
    consumer, v0.5.0).
"""

from __future__ import annotations

import logging
import time
from datetime import UTC, datetime

from sinapsis_ai.application.ports import (
    BundleResolver,
    ImageStore,
    InferenceEngine,
    ResultPublisher,
)
from sinapsis_ai.domain.errors import InferenceError
from sinapsis_ai.domain.models import (
    AnalysisRequest,
    AnalysisResult,
    Artifact,
    ModelRef,
)

logger = logging.getLogger(__name__)

_INFERENCE_ERROR_CODE = "INFERENCE_ERROR"


class AnalysisService:
    """Orchestrates a single analysis request end to end.

    Args:
        bundle_resolver: Resolves the analysis type to a local MONAI bundle.
        image_store: Fetches the input image and persists output artifacts.
        inference_engine: Executes the bundle's inference pipeline.
        publisher: Publishes the resulting AnalysisResult.
    """

    def __init__(
        self,
        bundle_resolver: BundleResolver,
        image_store: ImageStore,
        inference_engine: InferenceEngine,
        publisher: ResultPublisher,
    ) -> None:
        self._bundle_resolver = bundle_resolver
        self._image_store = image_store
        self._inference_engine = inference_engine
        self._publisher = publisher

    def run_analysis(self, request: AnalysisRequest) -> AnalysisResult:
        """Run the full analysis flow for a validated request.

        Returns:
            The published AnalysisResult (succeeded, or failed on inference
            errors).

        Raises:
            UnknownAnalysisTypeError: No bundle registered for the type.
            BundleResolutionError: The bundle could not be resolved.
            ImageAccessError: The input image could not be fetched or an
                artifact could not be persisted.
        """
        logger.info(
            "Analysis started: request_id=%s correlation_id=%s analysis_type=%s",
            request.request_id,
            request.correlation_id,
            request.analysis_type.value,
        )
        started = time.monotonic()

        # Pre-inference failures propagate to the caller (RF-005).
        model = self._bundle_resolver.resolve(request.analysis_type)
        image_path = self._image_store.fetch(request.image_uri)

        try:
            output = self._inference_engine.run(
                model, image_path, request.analysis_type
            )
        except InferenceError as exc:
            result = self._build_failed_result(request, model, started, exc)
            self._publisher.publish(result)
            logger.info(
                "Analysis failed: request_id=%s correlation_id=%s "
                "analysis_type=%s model_name=%s model_version=%s "
                "duration_ms=%d error_code=%s",
                request.request_id,
                request.correlation_id,
                request.analysis_type.value,
                model.name,
                model.version,
                result.duration_ms,
                _INFERENCE_ERROR_CODE,
            )
            return result

        # Artifact persistence failures propagate (transient storage issues).
        artifacts = [
            Artifact(
                type=artifact_type,
                uri=self._image_store.save_artifact(
                    request.study_id, artifact_type, local_path
                ),
            )
            for artifact_type, local_path in output.artifact_paths.items()
        ]

        result = AnalysisResult(
            request_id=request.request_id,
            study_id=request.study_id,
            status="succeeded",
            model=model,
            artifacts=artifacts,
            metrics=dict(output.metrics),
            error=None,
            processed_at=datetime.now(UTC),
            duration_ms=self._elapsed_ms(started),
        )
        self._publisher.publish(result)
        logger.info(
            "Analysis succeeded: request_id=%s correlation_id=%s "
            "analysis_type=%s model_name=%s model_version=%s "
            "artifacts_count=%d duration_ms=%d",
            request.request_id,
            request.correlation_id,
            request.analysis_type.value,
            model.name,
            model.version,
            len(artifacts),
            result.duration_ms,
        )
        return result

    def _build_failed_result(
        self,
        request: AnalysisRequest,
        model: ModelRef,
        started: float,
        exc: InferenceError,
    ) -> AnalysisResult:
        """Build a failed AnalysisResult preserving traceability (no internals)."""
        return AnalysisResult(
            request_id=request.request_id,
            study_id=request.study_id,
            status="failed",
            model=model,
            artifacts=[],
            metrics={},
            error={"code": _INFERENCE_ERROR_CODE, "message": str(exc)},
            processed_at=datetime.now(UTC),
            duration_ms=self._elapsed_ms(started),
        )

    @staticmethod
    def _elapsed_ms(started: float) -> int:
        """Milliseconds elapsed since `started`, always at least 1."""
        return max(1, int((time.monotonic() - started) * 1000))
