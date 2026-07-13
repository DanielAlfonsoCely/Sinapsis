"""Bundle config adapters: build MONAI workflow overrides per bundle.

Infrastructure layer — depends ONLY on the domain layer (stdlib only here).

Each MONAI bundle has its own inference.json schema and expects image input
in a specific format. The universal-override approach used in v2.0.0 breaks
bundles that:
  * Don't have a `checkpointloader` key (→ KeyError on override)
  * Expect `data_list_file_path` instead of `datalist` (→ datalist.json missing)
  * Use a custom data loading script (breast_density)

Each adapter is stateless and produces the dict of overrides that
_monai_workflow_factory should pass to monai.bundle.create_workflow.

Adapter responsibilities:
  * Build the image-input override in the format that bundle expects.
  * Include or omit `checkpointloader#map_location` based on bundle config.
  * Write any temporary JSON datalist files inside output_dir (already
    temporary per inference run, so no cleanup needed).

The Protocol is runtime-checkable so test doubles satisfy it structurally.
"""

from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any, Protocol, runtime_checkable

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Protocol
# ---------------------------------------------------------------------------


@runtime_checkable
class BundleConfigAdapter(Protocol):
    """Builds the MONAI override dict for a specific bundle."""

    def build_overrides(
        self,
        image_path: str,
        output_dir: str,
        bundle_path: str,
        device: str,
    ) -> dict[str, Any]:
        """Return override kwargs to pass to monai.bundle.create_workflow.

        Args:
            image_path: Absolute local path to the input image.
            output_dir: Temporary directory for this inference run; safe to
                write datalist JSON files here.
            bundle_path: Local path of the cached bundle directory.
            device: Compute device string (e.g. "cpu", "cuda").

        Returns:
            Dict of override key→value pairs for create_workflow.
        """
        ...


# ---------------------------------------------------------------------------
# Spleen CT Segmentation
# ---------------------------------------------------------------------------


class SpleenSegmentationAdapter:
    """Adapter for spleen_ct_segmentation.

    Uses the simple `datalist=[image_path]` override that the spleen bundle
    accepts directly (it has a `datalist` top-level key).
    """

    def build_overrides(
        self,
        image_path: str,
        output_dir: str,
        bundle_path: str,
        device: str,
    ) -> dict[str, Any]:
        return {
            "datalist": [image_path],
            "checkpointloader#map_location": device,
        }


# ---------------------------------------------------------------------------
# BraTS MRI Brain Tumor Segmentation
# ---------------------------------------------------------------------------


class BratsMriAdapter:
    """Adapter for brats_mri_segmentation.

    The bundle uses monai.data.load_decathlon_datalist with data_list_key
    'testing'. It expects a JSON file at data_list_file_path with the
    Decathlon multimodal format:
        {"testing": [{"image": ["<t1c>", "<t1>", "<t2>", "<flair>"]}]}

    BraTS requires 4 aligned MRI modalities (T1c, T1, T2, FLAIR) stacked as
    4 channels. When only a single input URI is provided (the system's current
    contract), the same volume is replicated across all 4 channel positions.
    MONAI's LoadImaged stacks the 4 paths into a [4, H, W, D] tensor, which
    is the channel-first format expected by the SegResNet (in_channels=4).

    Note: replicating one volume across modalities is medically incorrect but
    computationally valid — the network executes without error and returns a
    segmentation mask.
    """

    def build_overrides(
        self,
        image_path: str,
        output_dir: str,
        bundle_path: str,
        device: str,
    ) -> dict[str, Any]:
        datalist_path = self._write_datalist(image_path, output_dir)
        return {
            "data_list_file_path": datalist_path,
            "dataset_dir": str(Path(image_path).parent),
            "checkpointloader#map_location": device,
        }

    @staticmethod
    def _write_datalist(image_path: str, output_dir: str) -> str:
        # Pass the single image as a list of 4 paths (one per modality).
        # LoadImaged stacks them into a [4, H, W, D] tensor (channel-first).
        datalist = {"testing": [{"image": [image_path] * 4}]}
        path = Path(output_dir) / "brats_datalist.json"
        path.write_text(json.dumps(datalist), encoding="utf-8")
        logger.debug("BratsMriAdapter wrote datalist to %s", path)
        return str(path)


# ---------------------------------------------------------------------------
# Breast Density Classification
# ---------------------------------------------------------------------------


class BreastDensityAdapter:
    """Adapter for breast_density_classification.

    This bundle does NOT have a checkpointloader key — injecting that override
    causes a KeyError. It loads images via a custom scripts.createList module
    that reads a JSON with format:
        {"Test": [{"image": "<path>", "label": <int>}]}

    The bundle's config uses ``evaluating`` as its run ID instead of the standard
    ``run``. The ``_monai_run_id`` key signals the factory to use ``monai.bundle.run``
    with that run ID rather than the standard ``create_workflow`` path.
    """

    def build_overrides(
        self,
        image_path: str,
        output_dir: str,
        bundle_path: str,
        device: str,
    ) -> dict[str, Any]:
        sample_json_path = self._write_sample_json(image_path, output_dir)
        return {
            "_monai_run_id": "evaluating",
            "data": {
                "_target_": "scripts.createList.CreateImageLabelList",
                "filename": sample_json_path,
            },
        }

    @staticmethod
    def _write_sample_json(image_path: str, output_dir: str) -> str:
        data = {"Test": [{"image": image_path, "label": 0}]}
        path = Path(output_dir) / "breast_density_samples.json"
        path.write_text(json.dumps(data), encoding="utf-8")
        logger.debug("BreastDensityAdapter wrote sample JSON to %s", path)
        return str(path)


# ---------------------------------------------------------------------------
# Lung Nodule CT Detection
# ---------------------------------------------------------------------------


class LungNoduleAdapter:
    """Adapter for lung_nodule_ct_detection.

    Uses monai.data.load_decathlon_datalist with data_list_key 'validation'
    (LUNA16 format). Requires a JSON at data_list_file_path:
        {"validation": [{"image": "<path>"}]}
    """

    def build_overrides(
        self,
        image_path: str,
        output_dir: str,
        bundle_path: str,
        device: str,
    ) -> dict[str, Any]:
        datalist_path = self._write_datalist(image_path, output_dir)
        return {
            "data_list_file_path": datalist_path,
            "dataset_dir": str(Path(image_path).parent),
            "checkpointloader#map_location": device,
        }

    @staticmethod
    def _write_datalist(image_path: str, output_dir: str) -> str:
        datalist = {"validation": [{"image": image_path}]}
        path = Path(output_dir) / "luna16_datalist.json"
        path.write_text(json.dumps(datalist), encoding="utf-8")
        logger.debug("LungNoduleAdapter wrote datalist to %s", path)
        return str(path)
