"""Unit tests for infrastructure.bundle_adapters.

All adapters are tested in isolation: no real MONAI bundles, no downloads,
no filesystem side-effects beyond tmp_path.
"""

from __future__ import annotations

import json
from pathlib import Path

from sinapsis_ai.domain.models import AnalysisType
from sinapsis_ai.infrastructure.bundle_adapters import (
    BratsMriAdapter,
    BreastDensityAdapter,
    LungNoduleAdapter,
    SpleenSegmentationAdapter,
)
from sinapsis_ai.infrastructure.inference_engine import _ADAPTER_CATALOG

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_IMAGE = "/tmp/scan.nii.gz"
_OUTPUT = "/tmp/out"
_BUNDLE = "/cache/bundle"
_DEVICE = "cpu"


def _overrides(adapter: object) -> dict:
    return adapter.build_overrides(  # type: ignore[union-attr]
        image_path=_IMAGE,
        output_dir=_OUTPUT,
        bundle_path=_BUNDLE,
        device=_DEVICE,
    )


# ---------------------------------------------------------------------------
# Fundacional — catálogo y protocolo (HU1)
# ---------------------------------------------------------------------------


def test_adapter_catalog_covers_all_analysis_types() -> None:
    """Every AnalysisType has an entry in _ADAPTER_CATALOG."""
    for at in AnalysisType:
        assert at in _ADAPTER_CATALOG, f"No adapter for {at!r}"


def test_adapters_are_callable_with_expected_signature(tmp_path: Path) -> None:
    """All adapters respond to build_overrides and return a dict."""
    for at, adapter in _ADAPTER_CATALOG.items():
        result = adapter.build_overrides(
            image_path=str(tmp_path / "img.nii.gz"),
            output_dir=str(tmp_path),
            bundle_path=str(tmp_path),
            device="cpu",
        )
        assert isinstance(result, dict), f"Adapter for {at} must return dict"


# ---------------------------------------------------------------------------
# SpleenSegmentationAdapter (HU2)
# ---------------------------------------------------------------------------


def test_spleen_adapter_includes_datalist() -> None:
    """overrides contain datalist=[image_path]."""
    ov = _overrides(SpleenSegmentationAdapter())
    assert ov["datalist"] == [_IMAGE]


def test_spleen_adapter_includes_checkpoint_map_location() -> None:
    """overrides contain checkpointloader#map_location = device."""
    ov = _overrides(SpleenSegmentationAdapter())
    assert ov["checkpointloader#map_location"] == _DEVICE


def test_spleen_adapter_registered_in_catalog() -> None:
    assert isinstance(
        _ADAPTER_CATALOG[AnalysisType.CT_SPLEEN_SEGMENTATION], SpleenSegmentationAdapter
    )


# ---------------------------------------------------------------------------
# BratsMriAdapter (HU3)
# ---------------------------------------------------------------------------


def test_brats_adapter_creates_decathlon_datalist_json(tmp_path: Path) -> None:
    """overrides include data_list_file_path pointing to a valid Decathlon JSON.

    The image field must be a list of 4 paths (one per MRI modality) so that
    MONAI's LoadImaged stacks them into a [4, H, W, D] channel-first tensor,
    which is the format required by the BraTS SegResNet (in_channels=4).
    """
    adapter = BratsMriAdapter()
    ov = adapter.build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert "data_list_file_path" in ov
    json_path = Path(ov["data_list_file_path"])
    assert json_path.exists()
    data = json.loads(json_path.read_text())
    assert "testing" in data
    image_field = data["testing"][0]["image"]
    assert isinstance(image_field, list), "image must be a list of 4 modality paths"
    assert len(image_field) == 4, (
        "image list must have exactly 4 entries (T1c, T1, T2, FLAIR)"
    )
    assert all(p == _IMAGE for p in image_field), (
        "all modality paths must equal image_path"
    )


def test_brats_adapter_sets_dataset_dir_to_image_parent(tmp_path: Path) -> None:
    """dataset_dir is the parent directory of the image."""
    image = str(tmp_path / "scan.nii.gz")
    ov = BratsMriAdapter().build_overrides(
        image_path=image, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert ov["dataset_dir"] == str(tmp_path)


def test_brats_adapter_includes_checkpoint_map_location(tmp_path: Path) -> None:
    ov = BratsMriAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert ov["checkpointloader#map_location"] == _DEVICE


def test_brats_adapter_registered_in_catalog() -> None:
    assert isinstance(
        _ADAPTER_CATALOG[AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION], BratsMriAdapter
    )


# ---------------------------------------------------------------------------
# BreastDensityAdapter (HU4)
# ---------------------------------------------------------------------------


def test_breast_density_adapter_no_checkpointloader(tmp_path: Path) -> None:
    """overrides must NOT contain checkpointloader#map_location."""
    ov = BreastDensityAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert "checkpointloader#map_location" not in ov


def test_breast_density_adapter_sets_monai_run_id(tmp_path: Path) -> None:
    """overrides must contain _monai_run_id='evaluating' for the factory."""
    ov = BreastDensityAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert ov.get("_monai_run_id") == "evaluating"


def test_breast_density_adapter_creates_sample_json(tmp_path: Path) -> None:
    """overrides contain data.filename pointing to JSON with Test list."""
    ov = BreastDensityAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert "data" in ov
    json_path = Path(ov["data"]["filename"])
    assert json_path.exists()
    data = json.loads(json_path.read_text())
    assert "Test" in data
    assert data["Test"][0]["image"] == _IMAGE
    assert "label" in data["Test"][0]


def test_breast_density_adapter_data_target_preserved(tmp_path: Path) -> None:
    """The _target_ key of data points to the bundle's CreateImageLabelList."""
    ov = BreastDensityAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert "scripts.createList.CreateImageLabelList" in ov["data"]["_target_"]


def test_breast_density_adapter_registered_in_catalog() -> None:
    assert isinstance(
        _ADAPTER_CATALOG[AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION],
        BreastDensityAdapter,
    )


# ---------------------------------------------------------------------------
# LungNoduleAdapter (HU5)
# ---------------------------------------------------------------------------


def test_lung_nodule_adapter_creates_luna16_datalist(tmp_path: Path) -> None:
    """overrides include data_list_file_path with validation key in LUNA16 format."""
    ov = LungNoduleAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert "data_list_file_path" in ov
    json_path = Path(ov["data_list_file_path"])
    assert json_path.exists()
    data = json.loads(json_path.read_text())
    assert "validation" in data
    assert data["validation"][0]["image"] == _IMAGE


def test_lung_nodule_adapter_sets_dataset_dir(tmp_path: Path) -> None:
    image = str(tmp_path / "scan.nii.gz")
    ov = LungNoduleAdapter().build_overrides(
        image_path=image, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert ov["dataset_dir"] == str(tmp_path)


def test_lung_nodule_adapter_includes_checkpoint_map_location(tmp_path: Path) -> None:
    ov = LungNoduleAdapter().build_overrides(
        image_path=_IMAGE, output_dir=str(tmp_path), bundle_path=_BUNDLE, device=_DEVICE
    )
    assert ov["checkpointloader#map_location"] == _DEVICE


def test_lung_nodule_adapter_registered_in_catalog() -> None:
    assert isinstance(
        _ADAPTER_CATALOG[AnalysisType.CT_LUNG_NODULE_DETECTION], LungNoduleAdapter
    )
