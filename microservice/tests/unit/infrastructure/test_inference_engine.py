"""Unit tests for sinapsis_ai.infrastructure.inference_engine.

The MONAI workflow is fully mocked via an injected fake workflow factory.
OutputExtractors are also injected as fakes so no nibabel/numpy needed.
No network, no model weights, no monai/torch import at test time.
"""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from sinapsis_ai.domain.errors import InferenceError
from sinapsis_ai.domain.models import AnalysisType, InferenceOutput, ModelRef
from sinapsis_ai.infrastructure.inference_engine import (
    _ADAPTER_CATALOG,
    _EXTRACTOR_CATALOG,
    MonaiInferenceEngine,
    _monai_workflow_factory,
)
from sinapsis_ai.infrastructure.output_extractors import (
    BreastDensityCSVExtractor,
    DetectionExtractor,
    SegmentationExtractor,
)

_MODEL_REF = ModelRef(
    name="spleen_ct_segmentation", version="0.5.3", local_path="/cache/spleen"
)
_DEFAULT_TYPE = AnalysisType.CT_SPLEEN_SEGMENTATION


# ---------------------------------------------------------------------------
# Test doubles
# ---------------------------------------------------------------------------


class FakeWorkflow:
    """Test double for a MONAI bundle workflow."""

    def __init__(self, output_dir: str, produce_output: bool = True) -> None:
        self._output_dir = output_dir
        self._produce_output = produce_output
        self.lifecycle: list[str] = []

    def initialize(self) -> None:
        self.lifecycle.append("initialize")

    def run(self) -> None:
        self.lifecycle.append("run")
        if self._produce_output:
            prediction = Path(self._output_dir) / "scan" / "scan_pred.nii.gz"
            prediction.parent.mkdir(parents=True, exist_ok=True)
            prediction.write_bytes(b"fake-nifti")

    def finalize(self) -> None:
        self.lifecycle.append("finalize")


class FakeWorkflowFactory:
    def __init__(self, produce_output: bool = True) -> None:
        self.calls: list[tuple[str, str, str, str, dict]] = []
        self.workflows: list[FakeWorkflow] = []
        self._produce_output = produce_output

    def __call__(
        self,
        bundle_path: str,
        image_path: str,
        output_dir: str,
        device: str,
        overrides: dict,
    ) -> FakeWorkflow:
        self.calls.append((bundle_path, image_path, output_dir, device, overrides))
        wf = FakeWorkflow(output_dir, produce_output=self._produce_output)
        self.workflows.append(wf)
        return wf


def _fake_extractor(output: InferenceOutput) -> MagicMock:
    """Return an OutputExtractor mock that always returns *output*."""
    ext = MagicMock()
    ext.extract.return_value = output
    return ext


def _segmentation_output() -> InferenceOutput:
    return InferenceOutput(
        artifact_paths={"segmentation_mask": "/tmp/mask.nii.gz"},
        metrics={"volume_voxels": 42.0},
    )


def _fake_adapter(overrides: dict | None = None) -> MagicMock:
    """Return a BundleConfigAdapter mock."""
    adp = MagicMock()
    adp.build_overrides.return_value = overrides or {"datalist": ["/scan.nii.gz"]}
    return adp


def _make_engine(
    factory: FakeWorkflowFactory | None = None,
    extractor_catalog: dict | None = None,
    adapter_catalog: dict | None = None,
) -> MonaiInferenceEngine:
    factory = factory or FakeWorkflowFactory()
    if extractor_catalog is None:
        extractor_catalog = {_DEFAULT_TYPE: _fake_extractor(_segmentation_output())}
    if adapter_catalog is None:
        adapter_catalog = {_DEFAULT_TYPE: _fake_adapter()}
    return MonaiInferenceEngine(
        device="cpu",
        workflow_factory=factory,
        extractor_catalog=extractor_catalog,
        adapter_catalog=adapter_catalog,
    )


# ---------------------------------------------------------------------------
# Workflow lifecycle
# ---------------------------------------------------------------------------


def test_engine_runs_workflow_lifecycle_in_order() -> None:
    """initialize → run → finalize are called in the correct order."""
    factory = FakeWorkflowFactory()
    engine = _make_engine(factory=factory)

    engine.run(_MODEL_REF, "/input/scan.nii.gz", _DEFAULT_TYPE)

    assert factory.workflows[0].lifecycle == ["initialize", "run", "finalize"]


def test_engine_passes_correct_args_to_factory() -> None:
    """Workflow factory receives all 5 args: paths, device and overrides dict."""
    factory = FakeWorkflowFactory()
    engine = _make_engine(factory=factory)

    engine.run(_MODEL_REF, "/work/scan.nii.gz", _DEFAULT_TYPE)

    bundle_path, image_path, output_dir, device, overrides = factory.calls[0]
    assert bundle_path == "/cache/spleen"
    assert image_path == "/work/scan.nii.gz"
    assert Path(output_dir).is_dir()
    assert device == "cpu"
    assert isinstance(overrides, dict)


# ---------------------------------------------------------------------------
# Extractor dispatch
# ---------------------------------------------------------------------------


def test_engine_dispatches_to_correct_extractor() -> None:
    """The extractor for the given AnalysisType is called, not the other."""
    seg_ext = _fake_extractor(_segmentation_output())
    det_ext = _fake_extractor(
        InferenceOutput(artifact_paths={}, metrics={"lesion_count": 2.0})
    )
    catalog = {
        AnalysisType.CT_SPLEEN_SEGMENTATION: seg_ext,
        AnalysisType.CT_LUNG_NODULE_DETECTION: det_ext,
    }
    adp_catalog = {
        AnalysisType.CT_SPLEEN_SEGMENTATION: _fake_adapter(),
        AnalysisType.CT_LUNG_NODULE_DETECTION: _fake_adapter(),
    }
    factory = FakeWorkflowFactory()
    engine = MonaiInferenceEngine(
        device="cpu",
        workflow_factory=factory,
        extractor_catalog=catalog,
        adapter_catalog=adp_catalog,
    )

    result = engine.run(
        _MODEL_REF, "/scan.nii.gz", AnalysisType.CT_LUNG_NODULE_DETECTION
    )

    det_ext.extract.assert_called_once()
    seg_ext.extract.assert_not_called()
    assert result.metrics["lesion_count"] == 2.0


def test_engine_unknown_analysis_type_raises_inference_error() -> None:
    """An AnalysisType with no extractor → InferenceError before running workflow."""
    engine = MonaiInferenceEngine(
        device="cpu",
        workflow_factory=FakeWorkflowFactory(),
        extractor_catalog={},  # empty catalog
        adapter_catalog={_DEFAULT_TYPE: _fake_adapter()},
    )
    with pytest.raises(InferenceError, match="No extractor"):
        engine.run(_MODEL_REF, "/scan.nii.gz", AnalysisType.CT_SPLEEN_SEGMENTATION)


def test_engine_missing_adapter_raises_inference_error() -> None:
    """An AnalysisType with no adapter → InferenceError before running workflow."""
    engine = MonaiInferenceEngine(
        device="cpu",
        workflow_factory=FakeWorkflowFactory(),
        extractor_catalog={_DEFAULT_TYPE: _fake_extractor(_segmentation_output())},
        adapter_catalog={},  # empty adapter catalog
    )
    with pytest.raises(InferenceError, match="No adapter"):
        engine.run(_MODEL_REF, "/scan.nii.gz", _DEFAULT_TYPE)


def test_engine_returns_extractor_output() -> None:
    """The InferenceOutput from the extractor is returned unchanged."""
    expected = InferenceOutput(
        artifact_paths={"segmentation_mask": "/tmp/m.nii.gz"},
        metrics={"volume_voxels": 99.0},
    )
    factory = FakeWorkflowFactory()
    engine = _make_engine(
        factory=factory,
        extractor_catalog={_DEFAULT_TYPE: _fake_extractor(expected)},
    )

    result = engine.run(_MODEL_REF, "/scan.nii.gz", _DEFAULT_TYPE)

    assert result is expected


# ---------------------------------------------------------------------------
# Error handling
# ---------------------------------------------------------------------------


def test_inference_engine_workflow_failure_raises() -> None:
    """A workflow run() failure is translated to InferenceError."""
    boom = RuntimeError("CUDA OOM")

    class ExplodingWorkflow(FakeWorkflow):
        def run(self) -> None:
            raise boom

    engine = MonaiInferenceEngine(
        device="cpu",
        workflow_factory=lambda bp, ip, od, dv, ov: ExplodingWorkflow(od),
        extractor_catalog={_DEFAULT_TYPE: _fake_extractor(_segmentation_output())},
        adapter_catalog={_DEFAULT_TYPE: _fake_adapter()},
    )

    with pytest.raises(InferenceError) as exc_info:
        engine.run(_MODEL_REF, "/scan.nii.gz", _DEFAULT_TYPE)

    assert exc_info.value.__cause__ is boom


def test_engine_factory_failure_raises_inference_error() -> None:
    """A factory failure → InferenceError."""

    def failing_factory(bp: str, ip: str, od: str, dv: str, ov: dict) -> FakeWorkflow:
        raise FileNotFoundError("inference.json not found")

    engine = MonaiInferenceEngine(
        device="cpu",
        workflow_factory=failing_factory,
        extractor_catalog={_DEFAULT_TYPE: _fake_extractor(_segmentation_output())},
        adapter_catalog={_DEFAULT_TYPE: _fake_adapter()},
    )

    with pytest.raises(InferenceError):
        engine.run(_MODEL_REF, "/scan.nii.gz", _DEFAULT_TYPE)


def test_engine_no_outputs_raises_inference_error() -> None:
    """Pipeline completes without producing files → InferenceError via extractor."""
    # Extractor that raises InferenceError (simulates empty output dir)
    ext = MagicMock()
    ext.extract.side_effect = InferenceError("no output files")

    engine = MonaiInferenceEngine(
        device="cpu",
        workflow_factory=FakeWorkflowFactory(produce_output=False),
        extractor_catalog={_DEFAULT_TYPE: ext},
        adapter_catalog={_DEFAULT_TYPE: _fake_adapter()},
    )

    with pytest.raises(InferenceError, match="no output"):
        engine.run(_MODEL_REF, "/scan.nii.gz", _DEFAULT_TYPE)


# ---------------------------------------------------------------------------
# Default catalog coverage
# ---------------------------------------------------------------------------


def test_default_catalog_covers_all_analysis_types() -> None:
    """Every AnalysisType has an extractor in _EXTRACTOR_CATALOG."""
    for at in AnalysisType:
        assert at in _EXTRACTOR_CATALOG, f"No extractor for {at}"


def test_default_catalog_segmentation_types_use_segmentation_extractor() -> None:
    assert isinstance(
        _EXTRACTOR_CATALOG[AnalysisType.CT_SPLEEN_SEGMENTATION], SegmentationExtractor
    )
    assert isinstance(
        _EXTRACTOR_CATALOG[AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION],
        SegmentationExtractor,
    )


def test_default_catalog_detection_type_uses_detection_extractor() -> None:
    assert isinstance(
        _EXTRACTOR_CATALOG[AnalysisType.CT_LUNG_NODULE_DETECTION], DetectionExtractor
    )


def test_default_catalog_breast_density_uses_breast_density_csv_extractor() -> None:
    assert isinstance(
        _EXTRACTOR_CATALOG[AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION],
        BreastDensityCSVExtractor,
    )


def test_adapter_catalog_covers_all_analysis_types() -> None:
    """Every AnalysisType has an entry in _ADAPTER_CATALOG."""
    for at in AnalysisType:
        assert at in _ADAPTER_CATALOG, f"No adapter for {at!r}"


# ---------------------------------------------------------------------------
# _monai_workflow_factory: sys.path injection
# ---------------------------------------------------------------------------


def test_monai_workflow_factory_adds_bundle_path_to_sys_path(
    tmp_path: Path,
) -> None:
    """_monai_workflow_factory inserts the resolved absolute bundle path into
    sys.path so that bundle-local scripts/ packages are importable by MONAI's
    config parser without a ModuleNotFoundError.
    """
    bundle_path = str(tmp_path / "fake_bundle")
    Path(bundle_path).mkdir()
    # The factory always resolves to an absolute path before inserting.
    abs_bundle_path = str(Path(bundle_path).resolve())

    captured_path: list[list[str]] = []

    def _fake_create_workflow(**kwargs: object) -> MagicMock:
        captured_path.append(list(sys.path))
        return MagicMock()

    with (
        patch(
            "sinapsis_ai.infrastructure.inference_engine.sys.path",
            new=list(sys.path),
        ) as patched_path,
        patch(
            "monai.bundle.scripts.create_workflow",
            side_effect=_fake_create_workflow,
        ),
    ):
        # Ensure abs path is NOT already in the patched path
        if abs_bundle_path in patched_path:
            patched_path.remove(abs_bundle_path)

        _monai_workflow_factory(
            bundle_path=bundle_path,
            image_path="/tmp/scan.nii.gz",
            output_dir=str(tmp_path / "out"),
            device="cpu",
            overrides={},
        )

    assert captured_path, "create_workflow was not called"
    assert abs_bundle_path in captured_path[0], (
        f"abs bundle_path not found in sys.path during create_workflow call; "
        f"sys.path was: {captured_path[0][:5]}..."
    )


def test_monai_workflow_factory_does_not_duplicate_bundle_path(
    tmp_path: Path,
) -> None:
    """If the resolved absolute bundle_path is already in sys.path it is not
    inserted again — deduplication works on absolute paths so it is consistent
    with the absolute path that MONAI's BundleWorkflow also inserts.
    """
    bundle_path = str(tmp_path / "existing_bundle")
    Path(bundle_path).mkdir()
    abs_bundle_path = str(Path(bundle_path).resolve())

    with patch(
        "sinapsis_ai.infrastructure.inference_engine.sys.path",
        new=[abs_bundle_path, "/some/other"],
    ) as patched_path:
        with patch("monai.bundle.scripts.create_workflow", return_value=MagicMock()):
            _monai_workflow_factory(
                bundle_path=bundle_path,
                image_path="/tmp/scan.nii.gz",
                output_dir=str(tmp_path / "out"),
                device="cpu",
                overrides={},
            )

        assert patched_path.count(abs_bundle_path) == 1, (
            "bundle_path was duplicated in sys.path"
        )


# ---------------------------------------------------------------------------
# _BundleRunWorkflow: non-standard run_id path (breast density)
# ---------------------------------------------------------------------------


def test_monai_workflow_factory_returns_bundle_run_workflow_when_run_id_set(
    tmp_path: Path,
) -> None:
    """When overrides contain _monai_run_id, factory returns a _BundleRunWorkflow."""
    from sinapsis_ai.infrastructure.inference_engine import _BundleRunWorkflow

    bundle_path = str(tmp_path / "bundle")
    Path(bundle_path).mkdir()

    with patch(
        "sinapsis_ai.infrastructure.inference_engine.sys.path", new=list(sys.path)
    ):
        wf = _monai_workflow_factory(
            bundle_path=bundle_path,
            image_path="/tmp/scan.nii.gz",
            output_dir=str(tmp_path / "out"),
            device="cpu",
            overrides={"_monai_run_id": "evaluating", "extra_key": "value"},
        )

    assert isinstance(wf, _BundleRunWorkflow)


def test_bundle_run_workflow_run_id_not_passed_to_monai(
    tmp_path: Path,
) -> None:
    """_monai_run_id is stripped from the kwargs passed to monai.bundle.run."""
    from sinapsis_ai.infrastructure.inference_engine import _BundleRunWorkflow

    bundle_path = str(tmp_path / "bundle")
    Path(bundle_path).mkdir()

    captured_kwargs: list[dict] = []

    def _fake_bundle_run(**kwargs: object) -> None:
        captured_kwargs.append(dict(kwargs))

    with patch(
        "sinapsis_ai.infrastructure.inference_engine.sys.path", new=list(sys.path)
    ):
        wf = _monai_workflow_factory(
            bundle_path=bundle_path,
            image_path="/tmp/scan.nii.gz",
            output_dir=str(tmp_path / "out"),
            device="cpu",
            overrides={"_monai_run_id": "evaluating", "extra_key": "val"},
        )

    assert isinstance(wf, _BundleRunWorkflow)

    with patch(
        "sinapsis_ai.infrastructure.inference_engine._BundleRunWorkflow.run",
        side_effect=lambda: _fake_bundle_run(**captured_kwargs[0]) if False else None,
    ):
        # Verify initialize and finalize are no-ops (must not raise)
        wf.initialize()
        wf.finalize()

    # _monai_run_id must not appear in bundle kwargs
    assert "_monai_run_id" not in wf._bundle_kwargs
    # extra_key should be preserved
    assert wf._bundle_kwargs.get("extra_key") == "val"
    # run_id captured on the object
    assert wf._run_id == "evaluating"


def test_bundle_run_workflow_initialize_finalize_are_noop(tmp_path: Path) -> None:
    """_BundleRunWorkflow.initialize() and finalize() don't raise."""
    from sinapsis_ai.infrastructure.inference_engine import _BundleRunWorkflow

    wf = _BundleRunWorkflow(
        run_id="evaluating",
        config_file="/fake/config.json",
        bundle_path="/fake/bundle",
        bundle_kwargs={},
    )
    wf.initialize()  # must not raise
    wf.finalize()  # must not raise


# ---------------------------------------------------------------------------
# _evict_bundle_scripts: cross-bundle scripts namespace pollution
# ---------------------------------------------------------------------------


def test_evict_bundle_scripts_removes_stale_foreign_module(
    tmp_path: Path,
) -> None:
    """Stale 'scripts' modules from a different bundle are removed from
    sys.modules so that the next bundle resolves its own scripts package.
    """
    from types import ModuleType

    from sinapsis_ai.infrastructure.inference_engine import _evict_bundle_scripts

    # Create two fake bundle directories
    bundle_a = tmp_path / "bundle_a"
    bundle_b = tmp_path / "bundle_b"
    bundle_a.mkdir()
    bundle_b.mkdir()

    # Plant a fake 'scripts' module that belongs to bundle_a
    fake_scripts = ModuleType("scripts")
    fake_scripts.__file__ = str(bundle_a / "scripts" / "__init__.py")
    fake_createlist = ModuleType("scripts.createList")
    fake_createlist.__file__ = str(bundle_a / "scripts" / "createList.py")

    with patch.dict(
        sys.modules,
        {"scripts": fake_scripts, "scripts.createList": fake_createlist},
    ):
        # Evicting for bundle_b should remove the bundle_a entries
        _evict_bundle_scripts(str(bundle_b.resolve()))
        assert "scripts" not in sys.modules
        assert "scripts.createList" not in sys.modules


def test_evict_bundle_scripts_keeps_current_bundle_module(
    tmp_path: Path,
) -> None:
    """Modules already belonging to the current bundle are NOT evicted."""
    from types import ModuleType

    from sinapsis_ai.infrastructure.inference_engine import _evict_bundle_scripts

    bundle = tmp_path / "my_bundle"
    bundle.mkdir()

    fake_scripts = ModuleType("scripts")
    fake_scripts.__file__ = str(bundle / "scripts" / "__init__.py")

    with patch.dict(sys.modules, {"scripts": fake_scripts}):
        _evict_bundle_scripts(str(bundle.resolve()))
        # Same bundle — module must remain
        assert "scripts" in sys.modules


def test_evict_bundle_scripts_ignores_modules_without_file(
    tmp_path: Path,
) -> None:
    """Modules with no __file__ (e.g. built-ins) are left untouched."""
    from types import ModuleType

    from sinapsis_ai.infrastructure.inference_engine import _evict_bundle_scripts

    bundle = tmp_path / "bundle"
    bundle.mkdir()

    nameless = ModuleType("scripts")
    # No __file__ attribute set

    with patch.dict(sys.modules, {"scripts": nameless}):
        _evict_bundle_scripts(str(bundle.resolve()))
        assert "scripts" in sys.modules
