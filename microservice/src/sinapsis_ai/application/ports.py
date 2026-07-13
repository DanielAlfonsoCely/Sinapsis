"""Ports (abstract contracts) required by the application layer.

The application layer depends ONLY on these protocols and on the domain —
never on pika, monai, torch or any infrastructure module (DIP, DESIGN.md §3).
Concrete implementations live in infrastructure/ and are injected by the
composition root (main.py):

  * BundleResolver   → infrastructure.bundle_registry.BundleRegistry (v0.3.0)
  * ImageStore       → infrastructure.image_store.LocalImageStore (v0.4.0)
  * InferenceEngine  → infrastructure.inference_engine.MonaiInferenceEngine (v0.4.0)
  * ResultPublisher  → infrastructure.publisher (v0.5.0)

All protocols are runtime-checkable structural types: any object with the
matching methods satisfies them (tests inject lightweight doubles).
"""

from __future__ import annotations

from typing import Protocol, runtime_checkable

from sinapsis_ai.domain.models import (
    AnalysisResult,
    AnalysisType,
    InferenceOutput,
    ModelRef,
)


@runtime_checkable
class BundleResolver(Protocol):
    """Resolves an analysis type to a locally available MONAI bundle."""

    def resolve(self, analysis_type: AnalysisType) -> ModelRef:
        """Return a ModelRef for the analysis type.

        Raises:
            UnknownAnalysisTypeError: No bundle is registered for the type.
            BundleResolutionError: The bundle could not be downloaded/loaded.
        """
        ...


@runtime_checkable
class ImageStore(Protocol):
    """Fetches input images by URI and persists output artifacts."""

    def fetch(self, image_uri: str) -> str:
        """Materialise the referenced image locally and return its local path.

        Raises:
            ImageAccessError: The image cannot be retrieved.
        """
        ...

    def save_artifact(self, study_id: str, artifact_type: str, local_path: str) -> str:
        """Persist a produced artifact file and return its URI.

        Raises:
            ImageAccessError: The artifact cannot be persisted.
        """
        ...


@runtime_checkable
class InferenceEngine(Protocol):
    """Executes the inference pipeline of a resolved bundle on a local image."""

    def run(
        self,
        model: ModelRef,
        image_path: str,
        analysis_type: AnalysisType,
    ) -> InferenceOutput:
        """Run pre-processing → network → inferer → post-processing.

        Args:
            model: Resolved bundle reference.
            image_path: Absolute local path of the input image.
            analysis_type: Used to select the correct output extractor.

        Raises:
            InferenceError: The pipeline failed during execution.
        """
        ...


@runtime_checkable
class ResultPublisher(Protocol):
    """Publishes an AnalysisResult back to the message broker."""

    def publish(self, result: AnalysisResult) -> None:
        """Publish the result (implementation arrives in v0.5.0)."""
        ...
