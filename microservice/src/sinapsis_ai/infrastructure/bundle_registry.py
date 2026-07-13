"""MONAI bundle registry: resolves an AnalysisType to a locally cached bundle.

Infrastructure layer — depends ONLY on the domain layer (never on application/
or presentation/). The `monai` import is lazy and confined to the default
download function, so unit tests can inject a fake downloader and run without
network access and without loading monai/torch.

Responsibilities:
  * Map each supported AnalysisType to its MONAI bundle name (single catalogue).
  * Download the bundle via `monai.bundle` into BUNDLE_CACHE_DIR (once).
  * Reuse already-cached bundles (cache hit = no re-download).
  * Return a domain ModelRef (name, version, local cache path).
  * Translate failures into the domain error hierarchy.
"""

from __future__ import annotations

import json
import logging
from collections.abc import Callable
from pathlib import Path

from sinapsis_ai.domain.errors import (
    BundleResolutionError,
    UnknownAnalysisTypeError,
)
from sinapsis_ai.domain.models import AnalysisType, ModelRef

logger = logging.getLogger(__name__)

# Signature of a bundle download function: (bundle_name, cache_dir, source) -> None.
# Injected into BundleRegistry (DIP); the default implementation uses monai.bundle.
DownloadFn = Callable[[str, Path, str], None]

# Single source of truth mapping analysis types to MONAI bundle names.
# Adding a new analysis type = add the enum member + one entry here (DESIGN.md §11).
_BUNDLE_CATALOG: dict[AnalysisType, str] = {
    AnalysisType.CT_SPLEEN_SEGMENTATION: "spleen_ct_segmentation",
    AnalysisType.CT_LUNG_NODULE_DETECTION: "lung_nodule_ct_detection",
    AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION: "brats_mri_segmentation",
    AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION: "breast_density_classification",
}

# Standard MONAI bundle layout: <bundle_dir>/configs/metadata.json.
# Its presence is the verifiable "fully cached" criterion (partial downloads
# without it are treated as not cached and re-downloaded).
_METADATA_RELATIVE_PATH = Path("configs") / "metadata.json"


def _monai_download(bundle_name: str, cache_dir: Path, source: str) -> None:
    """Download a bundle using monai.bundle (lazy import — production path only)."""
    from monai.bundle.scripts import download

    download(name=bundle_name, bundle_dir=str(cache_dir), source=source)


class BundleRegistry:
    """Resolves analysis types to ready-to-use MONAI bundles with local caching.

    Args:
        cache_dir: Directory where bundles are downloaded and cached
            (Settings.bundle_cache_dir). Created on demand if missing.
        source: Bundle download source (Settings.bundle_source,
            e.g. "monaihosting" or "huggingface").
        download_fn: Optional download function override (used by tests to
            avoid network access). Defaults to a monai.bundle-based downloader.
    """

    def __init__(
        self,
        cache_dir: str,
        source: str,
        download_fn: DownloadFn | None = None,
    ) -> None:
        self._cache_dir = Path(cache_dir)
        self._source = source
        self._download_fn: DownloadFn = (
            download_fn if download_fn is not None else _monai_download
        )

    def resolve(self, analysis_type: AnalysisType) -> ModelRef:
        """Resolve an analysis type to a locally available bundle.

        Returns:
            A ModelRef with the bundle name, its version (from metadata.json,
            empty string if unavailable) and the local cache path.

        Raises:
            UnknownAnalysisTypeError: The analysis type has no catalogue entry.
                No download is attempted.
            BundleResolutionError: The bundle could not be downloaded or the
                download did not produce a usable bundle directory.
        """
        bundle_name = _BUNDLE_CATALOG.get(analysis_type)
        if bundle_name is None:
            raise UnknownAnalysisTypeError(
                f"No bundle registered for analysis type {analysis_type!r}"
            )

        bundle_dir = self._cache_dir / bundle_name

        if self._is_cached(bundle_dir):
            logger.info(
                "Bundle cache hit: analysis_type=%s bundle=%s",
                analysis_type.value,
                bundle_name,
            )
        else:
            logger.info(
                "Bundle cache miss, downloading: analysis_type=%s bundle=%s source=%s",
                analysis_type.value,
                bundle_name,
                self._source,
            )
            self._download(bundle_name, bundle_dir)

        version = self._read_version(bundle_dir)
        logger.info(
            "Resolved bundle: analysis_type=%s bundle=%s version=%s",
            analysis_type.value,
            bundle_name,
            version or "<unknown>",
        )
        return ModelRef(name=bundle_name, version=version, local_path=str(bundle_dir))

    def _download(self, bundle_name: str, bundle_dir: Path) -> None:
        """Download the bundle into the cache dir, translating failures."""
        self._cache_dir.mkdir(parents=True, exist_ok=True)
        try:
            self._download_fn(bundle_name, self._cache_dir, self._source)
        except Exception as exc:
            raise BundleResolutionError(
                f"Failed to download bundle {bundle_name!r} "
                f"from source {self._source!r}"
            ) from exc
        if not bundle_dir.is_dir():
            raise BundleResolutionError(
                f"Download of bundle {bundle_name!r} completed but no bundle "
                f"directory was produced at {bundle_dir}"
            )

    @staticmethod
    def _is_cached(bundle_dir: Path) -> bool:
        """A bundle is fully cached iff its metadata.json exists (standard layout)."""
        return (bundle_dir / _METADATA_RELATIVE_PATH).is_file()

    @staticmethod
    def _read_version(bundle_dir: Path) -> str:
        """Read the bundle version from metadata.json; empty string on any failure.

        The version is informational (the local path is what matters for
        execution), so a missing/corrupt metadata file never fails resolution.
        """
        metadata_path = bundle_dir / _METADATA_RELATIVE_PATH
        try:
            metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            return ""
        version = metadata.get("version", "")
        return version if isinstance(version, str) else ""
