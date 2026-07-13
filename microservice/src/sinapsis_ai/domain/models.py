"""Domain entities for the SINAPSIS AI microservice.

All entities are pure Python dataclasses with zero external dependencies
(no pika, no monai, no pydantic-settings). Pydantic is intentionally NOT used
here to keep the domain layer fully decoupled from infrastructure concerns.

Validation that requires external knowledge (e.g. allowed analysis types,
JSON schemas) is the responsibility of the Presentation layer (schemas.py).
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any


class AnalysisType(StrEnum):
    """Catalogue of supported analysis types.

    Each member's value is the snake_case bundle name used both as the
    JSON field value in the message contract and as the MONAI bundle identifier.
    """

    CT_SPLEEN_SEGMENTATION = "ct_spleen_segmentation"
    CT_LUNG_NODULE_DETECTION = "ct_lung_nodule_detection"
    MRI_BRAIN_TUMOR_SEGMENTATION = "mri_brain_tumor_segmentation"
    XR_BREAST_DENSITY_CLASSIFICATION = "xr_breast_density_classification"


@dataclass(frozen=True)
class AnalysisRequest:
    """A validated analysis request received from RabbitMQ.

    All identifiers are opaque strings (no PII). The image is referenced
    by URI only — no pixel data is stored in this entity.

    Attributes:
        request_id: Unique identifier for this request (UUID string).
        study_id: Opaque identifier for the study being analysed.
        patient_ref: Opaque patient reference (no PII).
        analysis_type: The type of MONAI analysis to run.
        image_uri: URI pointing to the input image (e.g. s3://bucket/scan.nii.gz).
        requested_by: Opaque identifier of the requesting doctor (already authorised).
        correlation_id: End-to-end tracing identifier propagated from the backend.
        issued_at: UTC datetime when the request was issued.
    """

    request_id: str
    study_id: str
    patient_ref: str
    analysis_type: AnalysisType
    image_uri: str
    requested_by: str
    correlation_id: str
    issued_at: datetime

    def __post_init__(self) -> None:
        if not self.image_uri:
            raise ValueError("image_uri must not be empty")
        if not self.request_id:
            raise ValueError("request_id must not be empty")
        if not self.correlation_id:
            raise ValueError("correlation_id must not be empty")


@dataclass(frozen=True)
class ModelRef:
    """Reference to the resolved MONAI bundle that executed the inference.

    Attributes:
        name: Bundle name (e.g. "spleen_ct_segmentation").
        version: Bundle version string (e.g. "0.5.3"); may be empty if unknown.
        local_path: Local filesystem path of the cached bundle directory.
            Internal to the service (Application/Infrastructure layers);
            never serialised into the message contract. May be empty when
            the bundle has not been resolved to a local copy.
    """

    name: str
    version: str
    local_path: str = ""


@dataclass(frozen=True)
class Artifact:
    """A persisted output artefact produced by the inference pipeline.

    Attributes:
        type: Semantic type of the artefact (e.g. "segmentation_mask").
        uri: URI where the artefact was stored (e.g. s3://bucket/mask.nii.gz).
    """

    type: str
    uri: str


@dataclass(frozen=True)
class InferenceOutput:
    """The output of an inference engine run, consumed by the analysis use case.

    Carries only references to locally produced prediction files (no pixel data)
    and computed metrics. The Application layer persists each prediction file as
    an Artifact via the image store.

    Attributes:
        artifact_paths: Mapping of semantic artifact type (e.g.
            "segmentation_mask") to the local filesystem path of the produced
            prediction file.
        metrics: Computed metrics for the analysis (e.g. {"volume_ml": 210.4}).
    """

    artifact_paths: dict[str, str] = field(default_factory=dict)
    metrics: dict[str, Any] = field(default_factory=dict)


@dataclass
class AnalysisResult:
    """The result of an analysis, published back to RabbitMQ.

    Corresponds to the AnalysisResult contract in PROJECT_ARCHITECTURE §9.

    Attributes:
        request_id: Mirrors the originating AnalysisRequest.request_id.
        study_id: Mirrors the originating AnalysisRequest.study_id.
        status: "succeeded" if inference completed, "failed" otherwise.
        model: The bundle that was used (even on failure, if resolved).
        artifacts: List of output artefacts (empty on failure before inference).
        metrics: Dictionary of computed metrics (e.g. {"volume_ml": 210.4}).
        error: Error block {"code": str, "message": str} if status=="failed"; else None.
        processed_at: UTC datetime when processing completed.
        duration_ms: Wall-clock processing time in milliseconds.
    """

    request_id: str
    study_id: str
    status: str  # "succeeded" | "failed"
    model: ModelRef
    artifacts: list[Artifact] = field(default_factory=list)
    metrics: dict[str, Any] = field(default_factory=dict)
    error: dict[str, str] | None = None
    processed_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    duration_ms: int = 0

    def __post_init__(self) -> None:
        if self.status not in ("succeeded", "failed"):
            raise ValueError(
                f"status must be 'succeeded' or 'failed', got {self.status!r}"
            )
        if self.status == "succeeded" and self.error is not None:
            raise ValueError("error must be None when status is 'succeeded'")
