"""Local filesystem image store: fetches input images and persists artifacts.

Infrastructure layer — depends ONLY on the domain layer. Implements the local
backend for v0.4.0 (URIs `file://` and plain local paths); remote backends
(e.g. S3) are deferred to future versions and must be selected via
IMAGE_STORAGE_BACKEND in the composition root.

All failures (missing image, unsupported URI scheme, persistence errors) are
translated to the domain's ImageAccessError.
"""

from __future__ import annotations

import logging
import shutil
from pathlib import Path
from urllib.parse import urlparse
from urllib.request import url2pathname

from sinapsis_ai.domain.errors import ImageAccessError

logger = logging.getLogger(__name__)

_SUPPORTED_SCHEMES = ("", "file")


class LocalImageStore:
    """Resolves image URIs on the local filesystem and persists artifacts.

    Args:
        root_dir: Root directory where output artifacts are persisted, one
            subdirectory per study. Created on demand.
        uploads_dir: Root directory where input images are stored.
    """

    def __init__(self, root_dir: str, uploads_dir: str = "/app/uploads") -> None:
        self._root_dir = Path(root_dir)
        self._uploads_dir = Path(uploads_dir)

    def fetch(self, image_uri: str) -> str:
        """Materialise the referenced image locally and return its local path.

        Accepts `file://` URIs, plain local paths, or just filenames (which
        are searched for in `uploads_dir`).

        Raises:
            ImageAccessError: Unsupported URI scheme or missing image.
        """
        local_path = self._to_local_path(image_uri)

        # If it's just a filename, check the uploads directory
        if local_path.name == image_uri and not local_path.is_absolute():
            local_path = self._uploads_dir / local_path

        if not local_path.is_file():
            raise ImageAccessError(f"Image not found at path {local_path!r} (uri={image_uri!r})")
        logger.info("Image fetched: path=%s", local_path)
        return str(local_path)

    def save_artifact(self, study_id: str, artifact_type: str, local_path: str) -> str:
        """Persist a produced artifact under the store root and return its URI.

        The artifact is copied to `<root_dir>/<study_id>/<original name>` and
        addressed with an absolute `file://` URI.

        Raises:
            ImageAccessError: The source file is missing or the copy failed.
        """
        source = Path(local_path)
        destination_dir = self._root_dir / study_id
        destination = destination_dir / source.name
        try:
            destination_dir.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, destination)
        except OSError as exc:
            raise ImageAccessError(
                f"Failed to persist artifact {artifact_type!r} for study {study_id!r}"
            ) from exc
        uri = destination.resolve().as_uri()
        logger.info(
            "Artifact persisted: study_id=%s type=%s uri=%s",
            study_id,
            artifact_type,
            uri,
        )
        return uri

    @staticmethod
    def _to_local_path(image_uri: str) -> Path:
        """Translate a supported URI (file:// or plain path) to a local Path.

        Raises:
            ImageAccessError: The URI scheme is not supported by this backend.
        """
        parsed = urlparse(image_uri)
        if parsed.scheme not in _SUPPORTED_SCHEMES:
            raise ImageAccessError(
                f"Unsupported URI scheme {parsed.scheme!r} for the local "
                f"storage backend (uri={image_uri!r})"
            )
        if parsed.scheme == "file":
            return Path(url2pathname(parsed.path))
        return Path(image_uri)
