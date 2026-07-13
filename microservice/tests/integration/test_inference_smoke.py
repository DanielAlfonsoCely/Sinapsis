"""Integration smoke test: real MONAI bundle inference end to end.

Marked @pytest.mark.integration (skipped by default): requires network access
to download the real spleen_ct_segmentation bundle (~100 MB on first run) and
enough CPU/RAM to execute one sliding-window inference over the tiny synthetic
volume fixture.

Run with:
    uv run pytest -m integration tests/integration/test_inference_smoke.py -q
"""

from __future__ import annotations

from pathlib import Path

import pytest

from sinapsis_ai.domain.models import AnalysisType
from sinapsis_ai.infrastructure.bundle_registry import BundleRegistry
from sinapsis_ai.infrastructure.image_store import LocalImageStore
from sinapsis_ai.infrastructure.inference_engine import MonaiInferenceEngine

_FIXTURES_DIR = Path(__file__).parent.parent / "fixtures"
_TINY_VOLUME = _FIXTURES_DIR / "tiny_volume.nii.gz"


@pytest.mark.integration
def test_inference_smoke_downloads_bundle_and_runs(tmp_path: Path) -> None:
    """Real chain registry → image store → engine over the synthetic volume."""
    registry = BundleRegistry(
        cache_dir=str(tmp_path / "bundle_cache"), source="monaihosting"
    )
    store = LocalImageStore(root_dir=str(tmp_path / "artifacts"))
    engine = MonaiInferenceEngine(device="cpu")

    model = registry.resolve(AnalysisType.CT_SPLEEN_SEGMENTATION)
    assert Path(model.local_path).is_dir()

    image_path = store.fetch(str(_TINY_VOLUME))
    output = engine.run(model, image_path, AnalysisType.CT_SPLEEN_SEGMENTATION)

    assert output.artifact_paths, "inference produced no output files"
    for artifact_type, local_path in output.artifact_paths.items():
        produced = Path(local_path)
        assert produced.is_file()
        assert produced.stat().st_size > 0
        # Persist through the store to validate the full artifact flow.
        uri = store.save_artifact("smoke-study", artifact_type, local_path)
        assert uri.startswith("file://")
