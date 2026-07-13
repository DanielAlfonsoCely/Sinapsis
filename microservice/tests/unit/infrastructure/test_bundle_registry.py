"""Unit tests for sinapsis_ai.infrastructure.bundle_registry.

The MONAI download is fully mocked via an injected fake download function:
no network access, no model weights, no monai/torch import at test time
(CHANGELOG v0.3.0, CE-003).
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from sinapsis_ai.domain.errors import BundleResolutionError, UnknownAnalysisTypeError
from sinapsis_ai.domain.models import AnalysisType, ModelRef
from sinapsis_ai.infrastructure import bundle_registry
from sinapsis_ai.infrastructure.bundle_registry import BundleRegistry

# ---------------------------------------------------------------------------
# Helpers / fixtures
# ---------------------------------------------------------------------------

_BUNDLE_NAME = "spleen_ct_segmentation"
_BUNDLE_VERSION = "0.5.3"


def _materialise_bundle(
    cache_dir: Path,
    bundle_name: str = _BUNDLE_NAME,
    metadata: dict[str, object] | None = None,
    write_metadata: bool = True,
) -> Path:
    """Create a fake bundle directory with the standard MONAI layout."""
    configs_dir = cache_dir / bundle_name / "configs"
    configs_dir.mkdir(parents=True, exist_ok=True)
    if write_metadata:
        if metadata is None:
            metadata = {"version": _BUNDLE_VERSION}
        (configs_dir / "metadata.json").write_text(json.dumps(metadata))
    return cache_dir / bundle_name


class FakeDownloader:
    """Test double for the download function: records calls, materialises a bundle."""

    def __init__(
        self,
        metadata: dict[str, object] | None = None,
        write_metadata: bool = True,
        create_dir: bool = True,
    ) -> None:
        self.calls: list[tuple[str, Path, str]] = []
        self._metadata = metadata
        self._write_metadata = write_metadata
        self._create_dir = create_dir

    def __call__(self, bundle_name: str, cache_dir: Path, source: str) -> None:
        self.calls.append((bundle_name, cache_dir, source))
        if self._create_dir:
            _materialise_bundle(
                cache_dir,
                bundle_name,
                metadata=self._metadata,
                write_metadata=self._write_metadata,
            )


@pytest.fixture()
def cache_dir(tmp_path: Path) -> Path:
    """A cache directory path that does NOT exist yet (registry must create it)."""
    return tmp_path / "bundle_cache"


def _make_registry(
    cache_dir: Path, downloader: FakeDownloader | None = None
) -> tuple[BundleRegistry, FakeDownloader]:
    downloader = downloader if downloader is not None else FakeDownloader()
    registry = BundleRegistry(
        cache_dir=str(cache_dir), source="monaihosting", download_fn=downloader
    )
    return registry, downloader


# ---------------------------------------------------------------------------
# HU1 — resolve a known type to a ready-to-use bundle
# ---------------------------------------------------------------------------


def test_resolve_known_type_returns_modelref(cache_dir: Path) -> None:
    """A known analysis type downloads the bundle and returns a full ModelRef."""
    registry, downloader = _make_registry(cache_dir)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert isinstance(ref, ModelRef)
    assert ref.name == _BUNDLE_NAME
    assert ref.version == _BUNDLE_VERSION
    assert ref.local_path == str(cache_dir / _BUNDLE_NAME)
    assert downloader.calls == [(_BUNDLE_NAME, cache_dir, "monaihosting")]


def test_resolve_creates_cache_dir_if_missing(cache_dir: Path) -> None:
    """BUNDLE_CACHE_DIR is created on demand (RF-008)."""
    assert not cache_dir.exists()
    registry, _ = _make_registry(cache_dir)

    registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert cache_dir.is_dir()


def test_download_failure_raises_bundle_resolution_error(cache_dir: Path) -> None:
    """A failing download is translated to BundleResolutionError with its cause."""

    boom = ConnectionError("network unreachable")

    def failing_download(bundle_name: str, cache_dir: Path, source: str) -> None:
        raise boom

    registry = BundleRegistry(
        cache_dir=str(cache_dir), source="monaihosting", download_fn=failing_download
    )

    with pytest.raises(BundleResolutionError) as exc_info:
        registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert exc_info.value.__cause__ is boom


def test_download_producing_no_bundle_dir_raises(cache_dir: Path) -> None:
    """A download that completes without materialising the bundle dir fails."""
    downloader = FakeDownloader(create_dir=False)
    registry, _ = _make_registry(cache_dir, downloader)

    with pytest.raises(BundleResolutionError, match="no bundle"):
        registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)


def test_resolve_reads_version_from_metadata(cache_dir: Path) -> None:
    """The bundle version comes from configs/metadata.json (RF-009)."""
    downloader = FakeDownloader(metadata={"version": "1.2.3"})
    registry, _ = _make_registry(cache_dir, downloader)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert ref.version == "1.2.3"


def test_resolve_missing_metadata_version_falls_back_to_empty(cache_dir: Path) -> None:
    """metadata.json without a 'version' key yields version='' (edge case)."""
    downloader = FakeDownloader(metadata={"name": _BUNDLE_NAME})
    registry, _ = _make_registry(cache_dir, downloader)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert ref.version == ""
    assert ref.local_path == str(cache_dir / _BUNDLE_NAME)


def test_resolve_corrupt_metadata_falls_back_to_empty_version(cache_dir: Path) -> None:
    """A corrupt (non-JSON) metadata.json yields version='' without failing."""
    bundle_dir = _materialise_bundle(cache_dir)
    (bundle_dir / "configs" / "metadata.json").write_text("{not valid json")
    registry, downloader = _make_registry(cache_dir)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert downloader.calls == []  # metadata.json exists → cache hit
    assert ref.version == ""


def test_resolve_non_string_metadata_version_falls_back_to_empty(
    cache_dir: Path,
) -> None:
    """A non-string 'version' value in metadata.json yields version=''."""
    downloader = FakeDownloader(metadata={"version": 5})
    registry, _ = _make_registry(cache_dir, downloader)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert ref.version == ""


# ---------------------------------------------------------------------------
# HU2 — reject unknown analysis types (fail-safe)
# ---------------------------------------------------------------------------


def test_resolve_unknown_type_raises(
    cache_dir: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """A type without a catalogue entry raises UnknownAnalysisTypeError and
    never attempts a download (RF-005)."""
    monkeypatch.setattr(bundle_registry, "_BUNDLE_CATALOG", {})
    registry, downloader = _make_registry(cache_dir)

    with pytest.raises(UnknownAnalysisTypeError, match="ct_spleen_segmentation"):
        registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert downloader.calls == []
    assert not cache_dir.exists()  # no cache side effects either


def test_catalog_covers_all_analysis_types() -> None:
    """Every AnalysisType member has a catalogue entry (no orphan enum members)."""
    for analysis_type in AnalysisType:
        assert analysis_type in bundle_registry._BUNDLE_CATALOG


# ---------------------------------------------------------------------------
# HU3 — reuse cached bundles (no re-download)
# ---------------------------------------------------------------------------


def test_bundle_cached_not_redownloaded(cache_dir: Path) -> None:
    """A second resolution of the same type does not download again (RF-007)."""
    registry, downloader = _make_registry(cache_dir)

    first = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)
    second = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert len(downloader.calls) == 1
    assert second == first


def test_preexisting_cache_skips_download(cache_dir: Path) -> None:
    """A bundle cached by a previous process is reused without any download."""
    _materialise_bundle(cache_dir)  # cache populated before the registry exists
    registry, downloader = _make_registry(cache_dir)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert downloader.calls == []
    assert ref.name == _BUNDLE_NAME
    assert ref.version == _BUNDLE_VERSION
    assert ref.local_path == str(cache_dir / _BUNDLE_NAME)


def test_partial_cache_triggers_redownload(cache_dir: Path) -> None:
    """A bundle directory without metadata.json (partial download) is re-downloaded."""
    _materialise_bundle(cache_dir, write_metadata=False)  # no metadata.json
    registry, downloader = _make_registry(cache_dir)

    ref = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert len(downloader.calls) == 1
    assert ref.version == _BUNDLE_VERSION
