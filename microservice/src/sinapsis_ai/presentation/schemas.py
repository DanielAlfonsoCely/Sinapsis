"""Message DTOs and domain translation for the SINAPSIS AI Presentation layer.

Responsibilities:
  - Validate incoming RabbitMQ JSON payloads against the AnalysisRequest contract
    (PROJECT_ARCHITECTURE §9) and translate them to domain entities.
  - Serialize domain AnalysisResult entities to the result JSON contract (§9).

Rules:
  - Pydantic ValidationError MUST NOT escape this module. All validation failures
    are wrapped in domain exceptions (InvalidMessageError, UnknownAnalysisTypeError).
  - This module imports domain entities but NEVER imports pika, monai or
    pydantic-settings.
"""

from __future__ import annotations

import json
from datetime import UTC, datetime
from typing import Any

from pydantic import BaseModel, field_validator, model_validator

from sinapsis_ai.domain.errors import InvalidMessageError, UnknownAnalysisTypeError
from sinapsis_ai.domain.models import (
    AnalysisRequest,
    AnalysisResult,
    AnalysisType,
    Artifact,
    ModelRef,
)

# ---------------------------------------------------------------------------
# Request DTO
# ---------------------------------------------------------------------------


class AnalysisRequestDTO(BaseModel):
    """Pydantic model that validates an incoming analysis request JSON message.

    Corresponds to the AnalysisRequest contract in PROJECT_ARCHITECTURE §9.
    Extra fields from the broker are silently ignored for forward compatibility.
    """

    model_config = {"extra": "ignore"}

    request_id: str
    study_id: str
    patient_ref: str
    analysis_type: str
    image_uri: str
    requested_by: str
    correlation_id: str
    issued_at: datetime

    @field_validator("analysis_type")
    @classmethod
    def validate_analysis_type(cls, v: str) -> str:
        """Reject unknown analysis types before building the domain entity."""
        valid_values = {t.value for t in AnalysisType}
        if v not in valid_values:
            raise UnknownAnalysisTypeError(
                f"Unknown analysis_type {v!r}. Allowed: {sorted(valid_values)}"
            )
        return v

    @model_validator(mode="after")
    def validate_image_uri(self) -> AnalysisRequestDTO:
        """Reject empty image_uri."""
        if not self.image_uri:
            raise InvalidMessageError("image_uri must not be empty")
        return self

    def to_domain(self) -> AnalysisRequest:
        """Translate this DTO into a pure domain AnalysisRequest entity."""
        return AnalysisRequest(
            request_id=self.request_id,
            study_id=self.study_id,
            patient_ref=self.patient_ref,
            analysis_type=AnalysisType(self.analysis_type),
            image_uri=self.image_uri,
            requested_by=self.requested_by,
            correlation_id=self.correlation_id,
            issued_at=self.issued_at,
        )


def parse_request(raw: str | bytes) -> AnalysisRequest:
    """Parse a raw JSON string/bytes into a domain AnalysisRequest.

    Wraps all pydantic ValidationError instances into domain exceptions so that
    callers never need to handle pydantic internals.

    Raises:
        InvalidMessageError: If the JSON is malformed or a required field is
            missing or invalid.
        UnknownAnalysisTypeError: If analysis_type is not in the allowed set.
    """
    from pydantic import ValidationError

    try:
        data = json.loads(raw)
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise InvalidMessageError(f"Message is not valid JSON: {exc}") from exc

    try:
        dto = AnalysisRequestDTO.model_validate(data)
    except UnknownAnalysisTypeError:
        # Already the correct domain exception — re-raise as-is.
        raise
    except ValidationError as exc:
        # Wrap pydantic validation failures in the domain exception.
        raise InvalidMessageError(f"Message failed schema validation: {exc}") from exc

    return dto.to_domain()


# ---------------------------------------------------------------------------
# Result serialisation
# ---------------------------------------------------------------------------


def serialise_result(result: AnalysisResult) -> dict[str, Any]:
    """Serialise a domain AnalysisResult to a JSON-compatible dict.

    Follows the AnalysisResult contract in PROJECT_ARCHITECTURE §9:
      request_id, study_id, status, model, artifacts, metrics, error,
      processed_at, duration_ms.
    """
    processed_at = result.processed_at
    if processed_at.tzinfo is None:
        processed_at = processed_at.replace(tzinfo=UTC)

    return {
        "request_id": result.request_id,
        "study_id": result.study_id,
        "status": result.status,
        "model": {
            "name": result.model.name,
            "version": result.model.version,
        },
        "artifacts": [{"type": a.type, "uri": a.uri} for a in result.artifacts],
        "metrics": dict(result.metrics),
        "error": result.error,
        "processed_at": processed_at.isoformat(),
        "duration_ms": result.duration_ms,
    }


# ---------------------------------------------------------------------------
# Re-export helpers for consumers
# ---------------------------------------------------------------------------

__all__ = [
    "AnalysisRequestDTO",
    "parse_request",
    "serialise_result",
    # Domain models re-exported for convenience
    "ModelRef",
    "Artifact",
]
