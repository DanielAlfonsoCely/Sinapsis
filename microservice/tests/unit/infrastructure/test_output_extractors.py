"""Unit tests for infrastructure.output_extractors.

All extractors are tested with synthetic fixtures in tmp_path.
No real MONAI inference, no network access, no nibabel/torch import required
for the classification and detection extractors (JSON/CSV only).
"""

from __future__ import annotations

import json
import struct
from pathlib import Path

import pytest

from sinapsis_ai.domain.errors import InferenceError
from sinapsis_ai.domain.models import InferenceOutput
from sinapsis_ai.infrastructure.output_extractors import (
    BreastDensityCSVExtractor,
    ClassificationExtractor,
    DetectionExtractor,
    SegmentationExtractor,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _write_minimal_nifti(path: Path) -> None:
    """Write the smallest valid NIfTI-1 file (348-byte header + 1 voxel = 1)."""
    # NIfTI-1 header is exactly 348 bytes.
    # Fields we care about: sizeof_hdr=348, dim[0]=3, dim[1..3]=1, datatype=2 (uint8),
    # bitpix=8, pixdim[1..3]=1.0, vox_offset=352, magic='n+1\0'.
    hdr = bytearray(348)
    # sizeof_hdr (int32 LE at offset 0)
    struct.pack_into("<i", hdr, 0, 348)
    # dim[0]=3, dim[1]=dim[2]=dim[3]=1 (int16 LE at offsets 40,42,44,46)
    struct.pack_into("<hhhh", hdr, 40, 3, 1, 1, 1)
    # datatype=2 (uint8), bitpix=8 (int16 at offsets 70, 72)
    struct.pack_into("<hh", hdr, 70, 2, 8)
    # pixdim[0..3] = 1.0 (float32 at offset 76, 4 bytes each × 4)
    struct.pack_into("<ffff", hdr, 76, 1.0, 1.0, 1.0, 1.0)
    # vox_offset = 352.0 (float32 at offset 108)
    struct.pack_into("<f", hdr, 108, 352.0)
    # magic 'n+1\0' at offset 344
    hdr[344:348] = b"n+1\x00"
    # Pad to vox_offset (352) then write one non-zero voxel byte
    path.write_bytes(bytes(hdr) + b"\x00\x00\x00\x00" + b"\x01")


def _write_minimal_nifti_gz(path: Path) -> None:
    """Write a gzip-compressed minimal NIfTI file."""
    import gzip
    import io

    buf = io.BytesIO()
    hdr = bytearray(348)
    struct.pack_into("<i", hdr, 0, 348)
    struct.pack_into("<hhhh", hdr, 40, 3, 1, 1, 1)
    struct.pack_into("<hh", hdr, 70, 2, 8)
    struct.pack_into("<ffff", hdr, 76, 1.0, 1.0, 1.0, 1.0)
    struct.pack_into("<f", hdr, 108, 352.0)
    hdr[344:348] = b"n+1\x00"
    raw = bytes(hdr) + b"\x00\x00\x00\x00" + b"\x01"
    with gzip.GzipFile(fileobj=buf, mode="wb") as gz:
        gz.write(raw)
    path.write_bytes(buf.getvalue())


# ---------------------------------------------------------------------------
# SegmentationExtractor tests
# ---------------------------------------------------------------------------


def test_segmentation_extractor_finds_nifti_gz_and_computes_volume(
    tmp_path: Path,
) -> None:
    """A .nii.gz file is found and volume_voxels is computed (≥ 0)."""
    mask_path = tmp_path / "scan_pred.nii.gz"
    _write_minimal_nifti_gz(mask_path)

    result = SegmentationExtractor().extract(tmp_path)

    assert isinstance(result, InferenceOutput)
    assert result.artifact_paths == {"segmentation_mask": str(mask_path)}
    assert "volume_voxels" in result.metrics
    assert result.metrics["volume_voxels"] >= 0


def test_segmentation_extractor_nested_dir_found(tmp_path: Path) -> None:
    """Extractor finds .nii.gz even in a subdirectory of output_dir."""
    sub = tmp_path / "study"
    sub.mkdir()
    mask_path = sub / "pred.nii.gz"
    _write_minimal_nifti_gz(mask_path)

    result = SegmentationExtractor().extract(tmp_path)

    assert result.artifact_paths["segmentation_mask"] == str(mask_path)


def test_segmentation_extractor_empty_dir_raises(tmp_path: Path) -> None:
    """Empty output_dir → InferenceError."""
    with pytest.raises(InferenceError, match="no.*nii"):
        SegmentationExtractor().extract(tmp_path)


def test_segmentation_extractor_no_nifti_raises(tmp_path: Path) -> None:
    """Only non-NIfTI files → InferenceError."""
    (tmp_path / "scores.json").write_text("{}")
    with pytest.raises(InferenceError, match="no.*nii"):
        SegmentationExtractor().extract(tmp_path)


# ---------------------------------------------------------------------------
# ClassificationExtractor tests
# ---------------------------------------------------------------------------


def test_classification_extractor_reads_json_scores(tmp_path: Path) -> None:
    """Reads class scores JSON and returns max class + probability."""
    scores = {"A": 0.1, "B": 0.7, "C": 0.15, "D": 0.05}
    (tmp_path / "scores.json").write_text(json.dumps(scores))

    result = ClassificationExtractor().extract(tmp_path)

    assert result.artifact_paths == {}
    assert result.metrics["predicted_class"] == "B"
    assert abs(result.metrics["probability"] - 0.7) < 1e-6


def test_classification_extractor_single_class(tmp_path: Path) -> None:
    """Single class in JSON → that class is predicted with probability 1.0."""
    (tmp_path / "out.json").write_text(json.dumps({"malignant": 1.0}))

    result = ClassificationExtractor().extract(tmp_path)

    assert result.metrics["predicted_class"] == "malignant"
    assert result.metrics["probability"] == pytest.approx(1.0)


def test_classification_extractor_empty_dir_raises(tmp_path: Path) -> None:
    """Empty output_dir → InferenceError."""
    with pytest.raises(InferenceError, match="no.*json"):
        ClassificationExtractor().extract(tmp_path)


def test_classification_extractor_invalid_json_raises(tmp_path: Path) -> None:
    """Malformed JSON → InferenceError."""
    (tmp_path / "scores.json").write_text("{not valid}")
    with pytest.raises(InferenceError):
        ClassificationExtractor().extract(tmp_path)


def test_classification_extractor_empty_json_raises(tmp_path: Path) -> None:
    """Empty dict in JSON → InferenceError (no classes to pick from)."""
    (tmp_path / "scores.json").write_text("{}")
    with pytest.raises(InferenceError, match="empty"):
        ClassificationExtractor().extract(tmp_path)


# ---------------------------------------------------------------------------
# DetectionExtractor tests
# ---------------------------------------------------------------------------

_DETECTIONS = [
    {"box": [10, 20, 30, 40, 50, 60], "score": 0.9, "diameter_mm": 12.4},
    {"box": [5, 5, 5, 15, 15, 15], "score": 0.8, "diameter_mm": 8.1},
    {"box": [0, 0, 0, 10, 10, 10], "score": 0.7},
]


def test_detection_extractor_reads_json_detections(tmp_path: Path) -> None:
    """Reads detections JSON, returns lesion_count and max_diameter_mm."""
    (tmp_path / "detections.json").write_text(json.dumps(_DETECTIONS))

    result = DetectionExtractor().extract(tmp_path)

    assert result.artifact_paths == {}
    assert result.metrics["lesion_count"] == 3
    assert result.metrics["max_diameter_mm"] == pytest.approx(12.4)


def test_detection_extractor_empty_detections_returns_zero(tmp_path: Path) -> None:
    """Empty detections list → lesion_count=0, max_diameter_mm=0.0."""
    (tmp_path / "detections.json").write_text("[]")

    result = DetectionExtractor().extract(tmp_path)

    assert result.metrics["lesion_count"] == 0
    assert result.metrics["max_diameter_mm"] == 0.0


def test_detection_extractor_no_diameter_field(tmp_path: Path) -> None:
    """Detections without diameter_mm → max_diameter_mm=0.0, count correct."""
    detections = [{"box": [0, 0, 0, 5, 5, 5], "score": 0.9}]
    (tmp_path / "out.json").write_text(json.dumps(detections))

    result = DetectionExtractor().extract(tmp_path)

    assert result.metrics["lesion_count"] == 1
    assert result.metrics["max_diameter_mm"] == 0.0


def test_detection_extractor_empty_dir_raises(tmp_path: Path) -> None:
    """Empty output_dir → InferenceError."""
    with pytest.raises(InferenceError, match="no.*json"):
        DetectionExtractor().extract(tmp_path)


def test_detection_extractor_invalid_json_raises(tmp_path: Path) -> None:
    """Malformed JSON → InferenceError."""
    (tmp_path / "detections.json").write_text("not json")
    with pytest.raises(InferenceError):
        DetectionExtractor().extract(tmp_path)


def test_detection_extractor_non_list_json_only_raises(tmp_path: Path) -> None:
    """Only JSON objects (no list) in output_dir → InferenceError."""
    (tmp_path / "datalist.json").write_text('{"validation": []}')
    (tmp_path / "metadata.json").write_text('{"key": "value"}')
    with pytest.raises(InferenceError, match="no JSON list"):
        DetectionExtractor().extract(tmp_path)


def test_detection_extractor_skips_datalist_finds_result(tmp_path: Path) -> None:
    """Datalist JSON (dict) and result JSON (list) coexist: the list is used."""
    datalist = {"validation": [{"image": "/tmp/scan.nii.gz"}]}
    (tmp_path / "luna16_datalist.json").write_text(json.dumps(datalist))

    detections = [{"box": [0, 0, 0, 5, 5, 5], "score": 0.95, "diameter_mm": 6.0}]
    (tmp_path / "result_luna16_fold0.json").write_text(json.dumps(detections))

    result = DetectionExtractor().extract(tmp_path)

    assert result.metrics["lesion_count"] == 1
    assert result.metrics["max_diameter_mm"] == pytest.approx(6.0)


# ---------------------------------------------------------------------------
# BreastDensityCSVExtractor tests
# ---------------------------------------------------------------------------

_BREAST_CSV_HEADER = "/tmp/scan.nii.gz,0.12,0.58,0.20,0.10\n"
_BREAST_CSV_CLASSES = ("A", "B", "C", "D")


def test_breast_density_extractor_reads_csv(tmp_path: Path) -> None:
    """Reads predictions.csv and returns density class B with highest score."""
    (tmp_path / "predictions.csv").write_text(_BREAST_CSV_HEADER)

    result = BreastDensityCSVExtractor().extract(tmp_path)

    assert result.artifact_paths == {}
    assert result.metrics["predicted_class"] == "B"
    assert abs(result.metrics["probability"] - 0.58) < 1e-4


def test_breast_density_extractor_uniform_scores_picks_first(tmp_path: Path) -> None:
    """Tied scores: max() picks the first class alphabetically."""
    (tmp_path / "predictions.csv").write_text("/img.nii.gz,0.25,0.25,0.25,0.25\n")

    result = BreastDensityCSVExtractor().extract(tmp_path)

    assert result.metrics["predicted_class"] in _BREAST_CSV_CLASSES
    assert abs(result.metrics["probability"] - 0.25) < 1e-4


def test_breast_density_extractor_empty_dir_raises(tmp_path: Path) -> None:
    """Empty output_dir → InferenceError."""
    with pytest.raises(InferenceError, match="no.*csv"):
        BreastDensityCSVExtractor().extract(tmp_path)


def test_breast_density_extractor_non_numeric_score_raises(tmp_path: Path) -> None:
    """Non-numeric score in CSV → InferenceError."""
    (tmp_path / "predictions.csv").write_text("/img.nii.gz,A,B,C,D\n")
    with pytest.raises(InferenceError, match="non-numeric"):
        BreastDensityCSVExtractor().extract(tmp_path)


def test_breast_density_extractor_too_few_columns_raises(tmp_path: Path) -> None:
    """Row with fewer than 5 columns (path + 4 scores) → InferenceError."""
    (tmp_path / "predictions.csv").write_text("/img.nii.gz,0.5\n")
    with pytest.raises(InferenceError, match="no data rows"):
        BreastDensityCSVExtractor().extract(tmp_path)
