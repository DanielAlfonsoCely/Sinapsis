"""MONAI inference engine: executes a resolved bundle's inference pipeline.

Infrastructure layer — depends ONLY on the domain layer. The `monai` import is
lazy and confined to the default workflow factory, so unit tests inject a fake
factory and run without network access and without loading monai/torch.

The engine runs the bundle's full pipeline (pre-processing → network →
inferer → post-processing) on the configured device, then delegates output
extraction to the OutputExtractor registered for the given AnalysisType.

Bundle-specific overrides (image input format, checkpointloader presence, etc.)
are encapsulated in BundleConfigAdapters (v2.1.0) so each bundle receives
exactly the overrides its inference.json schema expects.

Any pipeline failure is translated to InferenceError.
"""

from __future__ import annotations

import logging
import sys
import tempfile
from collections.abc import Callable
from pathlib import Path
from typing import Any

from sinapsis_ai.domain.errors import InferenceError
from sinapsis_ai.domain.models import AnalysisType, InferenceOutput, ModelRef
from sinapsis_ai.infrastructure.bundle_adapters import (
    BratsMriAdapter,
    BreastDensityAdapter,
    BundleConfigAdapter,
    LungNoduleAdapter,
    SpleenSegmentationAdapter,
)
from sinapsis_ai.infrastructure.output_extractors import (
    BreastDensityCSVExtractor,
    DetectionExtractor,
    OutputExtractor,
    SegmentationExtractor,
)

logger = logging.getLogger(__name__)

# Signature of a workflow factory:
#   (bundle_path, image_path, output_dir, device, overrides) -> workflow
WorkflowFactory = Callable[[str, str, str, str, dict[str, Any]], Any]

# Which extractor to use per analysis type (v2.0.0).
_EXTRACTOR_CATALOG: dict[AnalysisType, OutputExtractor] = {
    AnalysisType.CT_SPLEEN_SEGMENTATION: SegmentationExtractor(),
    AnalysisType.CT_LUNG_NODULE_DETECTION: DetectionExtractor(),
    AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION: SegmentationExtractor(),
    AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION: BreastDensityCSVExtractor(),
}

# Which adapter builds the overrides for each bundle (v2.1.0).
# Adding a new model = add one entry here + _EXTRACTOR_CATALOG + _BUNDLE_CATALOG.
_ADAPTER_CATALOG: dict[AnalysisType, BundleConfigAdapter] = {
    AnalysisType.CT_SPLEEN_SEGMENTATION: SpleenSegmentationAdapter(),
    AnalysisType.CT_LUNG_NODULE_DETECTION: LungNoduleAdapter(),
    AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION: BratsMriAdapter(),
    AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION: BreastDensityAdapter(),
}


class _BundleRunWorkflow:
    """Wrap ``monai.bundle.run()`` in the initialize/run/finalize protocol.

    Some MONAI bundles (e.g. breast_density_classification) do not expose a
    standard ``initialize → run → finalize`` workflow because their config uses a
    custom ``evaluating`` run ID instead of ``run``. For those bundles the adapter
    sets ``_monai_run_id`` in the overrides dict to signal that ``monai.bundle.run``
    should be used directly, and this wrapper object is returned so the engine can
    call the standard lifecycle methods without modification.
    """

    def __init__(
        self,
        run_id: str,
        config_file: str,
        bundle_path: str,
        bundle_kwargs: dict[str, Any],
    ) -> None:
        self._run_id = run_id
        self._config_file = config_file
        self._bundle_path = bundle_path
        self._bundle_kwargs = bundle_kwargs

    def initialize(self) -> None:
        pass  # nothing to do — monai.bundle.run handles setup internally

    def run(self) -> None:
        from monai.bundle.scripts import run as bundle_run

        bundle_run(
            run_id=self._run_id,
            config_file=self._config_file,
            bundle_root=self._bundle_path,
            **self._bundle_kwargs,
        )

    def finalize(self) -> None:
        pass  # nothing to do


#: Override key that adapters can place in their overrides dict to select a
#: non-standard run ID (e.g. ``"evaluating"`` for breast_density_classification).
_RUN_ID_OVERRIDE_KEY = "_monai_run_id"


def _evict_bundle_scripts(bundle_path: str) -> None:
    """Remove stale bundle ``scripts`` entries from ``sys.modules``.

    MONAI bundles that ship a ``scripts/`` package all expose their classes
    under the top-level ``scripts`` namespace.  When two bundles are executed
    in the same process (e.g. brats then breast_density), Python caches the
    first ``scripts`` package it resolved in ``sys.modules``.  Subsequent
    bundles whose ``scripts/`` live under a *different* directory therefore
    find the wrong (stale) package and raise ``ModuleNotFoundError``.

    This function removes every ``sys.modules`` entry whose ``__file__``
    lives under a bundle path *other* than the current one, so that the next
    import resolves freshly from the correct ``scripts/`` directory.
    """
    abs_bundle = str(Path(bundle_path).resolve())
    to_delete = [
        name
        for name, mod in sys.modules.items()
        if name == "scripts" or name.startswith("scripts.")
        if getattr(mod, "__file__", None)
        and not str(Path(getattr(mod, "__file__", "")).resolve()).startswith(abs_bundle)
    ]
    for name in to_delete:
        del sys.modules[name]


def _monai_workflow_factory(
    bundle_path: str,
    image_path: str,
    output_dir: str,
    device: str,
    overrides: dict[str, Any],
) -> Any:
    """Create the bundle's inference workflow via monai.bundle (lazy import).

    Adds *bundle_path* (resolved to an absolute path) to ``sys.path`` so that
    bundles that ship custom ``scripts/`` modules (e.g. lung_nodule_ct_detection,
    breast_density_classification) can be imported by MONAI's config parser
    without a ``ModuleNotFoundError``.

    Uses absolute paths for the ``sys.path`` guard so that the check is
    consistent with the absolute path that MONAI's ``BundleWorkflow`` also
    inserts, preventing duplicates.

    Evicts stale ``scripts.*`` entries from ``sys.modules`` before each run
    so that consecutive bundles that each ship a ``scripts/`` package do not
    accidentally resolve the previous bundle's module (cross-bundle namespace
    pollution).

    If the adapter placed ``_monai_run_id`` in *overrides*, that key is
    extracted and a ``_BundleRunWorkflow`` wrapper is returned instead of the
    standard ``create_workflow`` object.  This handles bundles whose config
    uses a run ID other than ``"run"`` (e.g. ``"evaluating"``).
    """
    # Resolve to absolute path so our guard matches what MONAI inserts (abs).
    abs_bundle_path = str(Path(bundle_path).resolve())

    # Evict any stale 'scripts.*' modules from a previous bundle run.
    _evict_bundle_scripts(abs_bundle_path)

    # Ensure bundle-local scripts/ packages are importable (absolute path).
    if abs_bundle_path not in sys.path:
        sys.path.insert(0, abs_bundle_path)

    config_file = str(Path(abs_bundle_path) / "configs" / "inference.json")
    # Base overrides that are always safe to apply first; adapter overrides
    # take precedence via dict merge (adapter may override output_dir/device).
    base: dict[str, Any] = {
        "output_dir": output_dir,
        "device": device,
    }
    base.update(overrides)

    # Extract the private signalling key before passing kwargs to MONAI.
    run_id = base.pop(_RUN_ID_OVERRIDE_KEY, None)
    if run_id is not None:
        return _BundleRunWorkflow(
            run_id=str(run_id),
            config_file=config_file,
            bundle_path=abs_bundle_path,
            bundle_kwargs=base,
        )

    from monai.bundle.scripts import create_workflow

    return create_workflow(
        config_file=config_file,
        workflow_type="infer",
        bundle_root=abs_bundle_path,
        **base,
    )


class MonaiInferenceEngine:
    """Executes MONAI bundle inference pipelines on a configured device.

    Args:
        device: Inference device ("cpu" / "cuda").
        workflow_factory: Optional workflow factory override for testing.
            Signature: (bundle_path, image_path, output_dir, device, overrides) -> wf.
        extractor_catalog: Optional extractor map override for testing.
        adapter_catalog: Optional adapter map override for testing.
    """

    def __init__(
        self,
        device: str,
        workflow_factory: WorkflowFactory | None = None,
        extractor_catalog: dict[AnalysisType, OutputExtractor] | None = None,
        adapter_catalog: dict[AnalysisType, BundleConfigAdapter] | None = None,
    ) -> None:
        self._device = device
        self._workflow_factory: WorkflowFactory = (
            workflow_factory
            if workflow_factory is not None
            else _monai_workflow_factory
        )
        self._extractor_catalog: dict[AnalysisType, OutputExtractor] = (
            extractor_catalog if extractor_catalog is not None else _EXTRACTOR_CATALOG
        )
        self._adapter_catalog: dict[AnalysisType, BundleConfigAdapter] = (
            adapter_catalog if adapter_catalog is not None else _ADAPTER_CATALOG
        )

    def run(
        self,
        model: ModelRef,
        image_path: str,
        analysis_type: AnalysisType,
    ) -> InferenceOutput:
        """Run the bundle pipeline and extract the typed output.

        Args:
            model: Resolved bundle reference (name, version, local_path).
            image_path: Absolute local path of the input image.
            analysis_type: Used to select the extractor and adapter.

        Returns:
            InferenceOutput with artifact_paths and metrics populated.

        Raises:
            InferenceError: Missing extractor/adapter, pipeline failure, or
                extractor could not process output files.
        """
        extractor = self._extractor_catalog.get(analysis_type)
        if extractor is None:
            raise InferenceError(
                f"No extractor registered for analysis_type {analysis_type!r}. "
                "Add an entry to _EXTRACTOR_CATALOG in inference_engine.py."
            )

        adapter = self._adapter_catalog.get(analysis_type)
        if adapter is None:
            raise InferenceError(
                f"No adapter registered for analysis_type {analysis_type!r}. "
                "Add an entry to _ADAPTER_CATALOG in inference_engine.py."
            )

        output_dir = tempfile.mkdtemp(prefix="sinapsis_inference_")
        logger.info(
            "Inference started: bundle=%s version=%s analysis_type=%s device=%s",
            model.name,
            model.version or "<unknown>",
            analysis_type.value,
            self._device,
        )

        overrides = adapter.build_overrides(
            image_path=image_path,
            output_dir=output_dir,
            bundle_path=model.local_path,
            device=self._device,
        )
        logger.debug("Bundle overrides for %s: %s", model.name, list(overrides.keys()))

        try:
            workflow = self._workflow_factory(
                model.local_path, image_path, output_dir, self._device, overrides
            )
            workflow.initialize()
            workflow.run()
            workflow.finalize()
        except Exception as exc:
            raise InferenceError(
                f"Inference pipeline failed for bundle {model.name!r}"
            ) from exc

        output = extractor.extract(Path(output_dir))
        logger.info(
            "Inference finished: bundle=%s analysis_type=%s artifacts=%d metrics=%s",
            model.name,
            analysis_type.value,
            len(output.artifact_paths),
            list(output.metrics.keys()),
        )
        return output
