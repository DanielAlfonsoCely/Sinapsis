"""Unit tests for sinapsis_ai.infrastructure.image_store.

Exercises the local filesystem backend against tmp_path: no network, no
external storage services.
"""

from __future__ import annotations

from pathlib import Path
from urllib.parse import urlparse
from urllib.request import url2pathname

import pytest

from sinapsis_ai.domain.errors import ImageAccessError
from sinapsis_ai.infrastructure.image_store import LocalImageStore


@pytest.fixture()
def store(tmp_path: Path) -> LocalImageStore:
    return LocalImageStore(root_dir=str(tmp_path / "artifacts"))


@pytest.fixture()
def input_image(tmp_path: Path) -> Path:
    image = tmp_path / "input" / "scan.nii.gz"
    image.parent.mkdir(parents=True)
    image.write_bytes(b"fake-nifti-input")
    return image


# ---------------------------------------------------------------------------
# HU3 — fetch
# ---------------------------------------------------------------------------


def test_image_store_fetch_existing_returns_local_path(
    store: LocalImageStore, input_image: Path
) -> None:
    """Plain local paths and file:// URIs resolve to a readable local path."""
    from_plain = store.fetch(str(input_image))
    from_uri = store.fetch(input_image.resolve().as_uri())

    assert Path(from_plain).read_bytes() == b"fake-nifti-input"
    assert Path(from_uri).read_bytes() == b"fake-nifti-input"


def test_image_store_fetch_missing_raises(
    store: LocalImageStore, tmp_path: Path
) -> None:
    """A URI pointing to a non-existent image raises ImageAccessError."""
    with pytest.raises(ImageAccessError, match="not found"):
        store.fetch(str(tmp_path / "does-not-exist.nii.gz"))


def test_image_store_unsupported_scheme_raises(store: LocalImageStore) -> None:
    """A non-local scheme (s3://) raises ImageAccessError (local backend only)."""
    with pytest.raises(ImageAccessError, match="s3"):
        store.fetch("s3://bucket/study/scan.nii.gz")


# ---------------------------------------------------------------------------
# HU3 — save_artifact
# ---------------------------------------------------------------------------


def test_image_store_save_artifact_persists_and_returns_uri(
    store: LocalImageStore, tmp_path: Path
) -> None:
    """A produced artifact is copied under root/<study_id>/ with a file:// URI."""
    produced = tmp_path / "work" / "mask.nii.gz"
    produced.parent.mkdir(parents=True)
    produced.write_bytes(b"fake-nifti-mask")

    uri = store.save_artifact("study-001", "segmentation_mask", str(produced))

    parsed = urlparse(uri)
    assert parsed.scheme == "file"
    persisted = Path(url2pathname(parsed.path))
    assert persisted.is_file()
    assert persisted.read_bytes() == b"fake-nifti-mask"
    assert persisted.parent.name == "study-001"


def test_image_store_save_artifact_failure_raises(
    store: LocalImageStore, tmp_path: Path
) -> None:
    """A missing source file raises ImageAccessError (with cause chained)."""
    with pytest.raises(ImageAccessError, match="segmentation_mask") as exc_info:
        store.save_artifact(
            "study-001", "segmentation_mask", str(tmp_path / "missing.nii.gz")
        )
    assert isinstance(exc_info.value.__cause__, OSError)


def test_image_store_roundtrip_saved_artifact_is_fetchable(
    store: LocalImageStore, tmp_path: Path
) -> None:
    """The URI returned by save_artifact can be fetched back (consistency)."""
    produced = tmp_path / "mask.nii.gz"
    produced.write_bytes(b"roundtrip")

    uri = store.save_artifact("study-002", "segmentation_mask", str(produced))
    fetched = store.fetch(uri)

    assert Path(fetched).read_bytes() == b"roundtrip"
