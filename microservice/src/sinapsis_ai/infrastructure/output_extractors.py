"""Output extractors: translate bundle output files into InferenceOutput.

Infrastructure layer -- depends ONLY on the domain layer.

Each extractor handles the specific output format of a class of MONAI bundles:
  * SegmentationExtractor     -- finds .nii/.nii.gz, computes volume_voxels.
  * ClassificationExtractor   -- reads a JSON dict {class: score}, returns the
                                 winning class and its probability.
  * DetectionExtractor        -- reads a JSON list of detection dicts, returns
                                 lesion_count + max_diameter_mm.
  * BreastDensityCSVExtractor -- reads the CSV from ClassificationSaver with
                                 softmax scores per class (A, B, C, D density).

Design notes:
  - nibabel and numpy are imported LAZILY inside SegmentationExtractor.extract() so
    that workers doing only classification or detection don't pay the import cost.
  - ClassificationExtractor and DetectionExtractor use only stdlib json/csv.
  - DetectionExtractor skips JSON files whose top-level value is not a list so that
    adapter-written datalist files in the same output_dir are not misidentified as
    detection results.
  - All extractors raise InferenceError (domain exception) on any unexpected condition
    so callers only need to handle one exception type.
  - The Protocol is runtime-checkable so test doubles satisfy it structurally.
"""

from __future__ import annotations

import csv
import json
import logging
from pathlib import Path
from typing import Any, Protocol, runtime_checkable

from sinapsis_ai.domain.errors import InferenceError
from sinapsis_ai.domain.models import InferenceOutput

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Protocol
# ---------------------------------------------------------------------------


@runtime_checkable
class OutputExtractor(Protocol):
    """Translates the raw output directory of a MONAI bundle into InferenceOutput."""

    def extract(self, output_dir: Path) -> InferenceOutput:
        """Extract model outputs from *output_dir*.

        Args:
            output_dir: Directory where the MONAI bundle wrote its predictions.

        Returns:
            InferenceOutput with artifact_paths and metrics populated.

        Raises:
            InferenceError: The expected output files are missing or malformed.
        """
        ...


# ---------------------------------------------------------------------------
# Segmentation
# ---------------------------------------------------------------------------


class SegmentationExtractor:
    """Extract a segmentation mask (.nii.gz) and compute voxel volume.

    Searches *output_dir* recursively for the first NIfTI file (.nii or .nii.gz).
    Computes volume_voxels as the count of non-zero voxels in the mask array.
    """

    def extract(self, output_dir: Path) -> InferenceOutput:
        nifti_files = sorted(
            p
            for p in output_dir.rglob("*")
            if p.is_file()
            and (p.suffix == ".gz" and p.stem.endswith(".nii") or p.suffix == ".nii")
        )
        if not nifti_files:
            raise InferenceError(
                f"SegmentationExtractor: no .nii/.nii.gz files found in {output_dir}"
            )

        mask_path = nifti_files[0]
        volume_voxels = self._count_nonzero(mask_path)

        logger.info(
            "Segmentation extracted: file=%s volume_voxels=%d",
            mask_path.name,
            volume_voxels,
        )
        return InferenceOutput(
            artifact_paths={"segmentation_mask": str(mask_path)},
            metrics={"volume_voxels": float(volume_voxels)},
        )

    @staticmethod
    def _count_nonzero(mask_path: Path) -> int:
        """Count non-zero voxels in a NIfTI file (lazy nibabel + numpy import)."""
        try:
            import nibabel as nib
            import numpy as np

            img: Any = nib.load(str(mask_path))
            data = np.asarray(img.get_fdata())
            return int(np.count_nonzero(data))
        except Exception as exc:
            logger.warning(
                "Could not compute volume_voxels for %s: %s — defaulting to 0",
                mask_path.name,
                exc,
            )
            return 0


# ---------------------------------------------------------------------------
# Classification
# ---------------------------------------------------------------------------


class ClassificationExtractor:
    """Extract class probabilities from a JSON scores file.

    Expects the bundle to write a JSON file in output_dir with the format::

        {"class_a": 0.1, "class_b": 0.7, "class_c": 0.2}

    Returns the class with the highest score as ``predicted_class`` and its
    score as ``probability``.
    """

    def extract(self, output_dir: Path) -> InferenceOutput:
        json_files = sorted(p for p in output_dir.rglob("*") if p.suffix == ".json")
        if not json_files:
            raise InferenceError(
                f"ClassificationExtractor: no .json files found in {output_dir}"
            )

        scores = self._load_scores(json_files[0])
        if not scores:
            raise InferenceError(
                f"ClassificationExtractor: empty scores dict in {json_files[0]}"
            )

        predicted_class = max(scores, key=lambda k: scores[k])
        probability = float(scores[predicted_class])

        logger.info(
            "Classification extracted: predicted_class=%s probability=%.4f",
            predicted_class,
            probability,
        )
        return InferenceOutput(
            artifact_paths={},
            metrics={
                "predicted_class": predicted_class,
                "probability": probability,
            },
        )

    @staticmethod
    def _load_scores(path: Path) -> dict[str, float]:
        try:
            raw: Any = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            raise InferenceError(
                f"ClassificationExtractor: failed to parse {path}: {exc}"
            ) from exc
        if not isinstance(raw, dict):
            raise InferenceError(
                f"ClassificationExtractor: expected JSON dict, got {type(raw).__name__}"
            )
        return {str(k): float(v) for k, v in raw.items()}


# ---------------------------------------------------------------------------
# Detection
# ---------------------------------------------------------------------------


class DetectionExtractor:
    """Extract detection metrics from a JSON detections file.

    Expects the bundle to write a JSON file in output_dir with the format::

        [
            {"box": [...], "score": 0.9, "diameter_mm": 12.4},
            {"box": [...], "score": 0.8},
            ...
        ]

    Returns:
        * ``lesion_count``: total number of detected lesions.
        * ``max_diameter_mm``: maximum diameter across detections (0.0 if unknown).

    Implementation note: the adapter may write input datalist JSON files into the
    same output_dir. This extractor skips any JSON whose top-level value is not a
    list so those datalist files are not misidentified as detection results.
    """

    def extract(self, output_dir: Path) -> InferenceOutput:
        json_files = sorted(p for p in output_dir.rglob("*") if p.suffix == ".json")
        if not json_files:
            raise InferenceError(
                f"DetectionExtractor: no .json files found in {output_dir}"
            )

        detections = self._find_detections(json_files)

        lesion_count = len(detections)
        diameters = [float(d["diameter_mm"]) for d in detections if "diameter_mm" in d]
        max_diameter_mm = max(diameters) if diameters else 0.0

        logger.info(
            "Detection extracted: lesion_count=%d max_diameter_mm=%.2f",
            lesion_count,
            max_diameter_mm,
        )
        return InferenceOutput(
            artifact_paths={},
            metrics={
                "lesion_count": float(lesion_count),
                "max_diameter_mm": max_diameter_mm,
            },
        )

    @staticmethod
    def _find_detections(json_files: list[Path]) -> list[dict[str, Any]]:
        """Return detections from the first JSON file that contains a list.

        Skips files whose top-level JSON value is not a list (e.g. input datalists
        written by the adapter into the same output directory).
        """
        last_exc: Exception | None = None
        for path in json_files:
            try:
                raw: Any = json.loads(path.read_text(encoding="utf-8"))
            except (OSError, json.JSONDecodeError) as exc:
                last_exc = exc
                continue
            if isinstance(raw, list):
                return list(raw)
        # All files were non-list or unreadable.
        if last_exc is not None:
            raise InferenceError(
                f"DetectionExtractor: failed to parse any detection JSON: {last_exc}"
            ) from last_exc
        raise InferenceError(
            f"DetectionExtractor: no JSON list found in {json_files[0].parent} "
            f"(checked {len(json_files)} file(s); all contained non-list values)"
        )


# ---------------------------------------------------------------------------
# Breast Density
# ---------------------------------------------------------------------------

_BREAST_DENSITY_CLASSES = ("A", "B", "C", "D")


class BreastDensityCSVExtractor:
    """Extract breast density classification from the CSV produced by the bundle.

    The ``breast_density_classification`` bundle writes a ``predictions.csv``
    via ``monai.handlers.ClassificationSaver``. Each row has the format::

        <image_path>,<score_A>,<score_B>,<score_C>,<score_D>

    Returns the density class with the highest softmax score as
    ``predicted_class`` (one of "A", "B", "C", "D") and its score as
    ``probability``.
    """

    def extract(self, output_dir: Path) -> InferenceOutput:
        csv_files = sorted(p for p in output_dir.rglob("*") if p.suffix == ".csv")
        if not csv_files:
            raise InferenceError(
                f"BreastDensityCSVExtractor: no .csv files found in {output_dir}"
            )

        predicted_class, probability = self._load_scores(csv_files[0])

        logger.info(
            "Breast density extracted: predicted_class=%s probability=%.4f",
            predicted_class,
            probability,
        )
        return InferenceOutput(
            artifact_paths={},
            metrics={
                "predicted_class": predicted_class,
                "probability": probability,
            },
        )

    @staticmethod
    def _load_scores(path: Path) -> tuple[str, float]:
        """Parse the first data row of *path* and return (class, score)."""
        try:
            text = path.read_text(encoding="utf-8")
        except OSError as exc:
            raise InferenceError(
                f"BreastDensityCSVExtractor: cannot read {path}: {exc}"
            ) from exc

        rows = list(csv.reader(text.splitlines()))
        # Filter out empty rows and potential header lines (need path + 4 scores).
        n_classes = len(_BREAST_DENSITY_CLASSES)
        data_rows = [r for r in rows if len(r) >= n_classes + 1]
        if not data_rows:
            raise InferenceError(f"BreastDensityCSVExtractor: no data rows in {path}")

        # First column is the image path; remaining are scores per class.
        score_fields = data_rows[0][1 : n_classes + 1]
        try:
            scores = {
                cls: float(s)
                for cls, s in zip(_BREAST_DENSITY_CLASSES, score_fields, strict=True)
            }
        except ValueError as exc:
            raise InferenceError(
                f"BreastDensityCSVExtractor: non-numeric score in {path}: {exc}"
            ) from exc

        predicted_class = max(scores, key=lambda k: scores[k])
        return predicted_class, float(scores[predicted_class])
