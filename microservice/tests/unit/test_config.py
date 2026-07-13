"""Unit tests for sinapsis_ai.config.Settings.

Covers the three scenarios specified in CHANGELOG v0.1.0:
  - test_settings_loads_from_env
  - test_settings_missing_required_raises
  - test_settings_defaults
"""

import pytest
from pydantic import ValidationError

from sinapsis_ai.config import Settings

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_REQUIRED = {
    "RABBITMQ_URL": "amqp://guest:guest@localhost:5672/",
    "ALLOWED_ANALYSIS_TYPES": "ct_spleen_segmentation",
    "IMAGE_STORAGE_BACKEND": "s3",
}


def _make_settings(monkeypatch: pytest.MonkeyPatch, **overrides: str) -> Settings:
    """Build a Settings instance from environment variables only (no .env file)."""
    env = {**_REQUIRED, **overrides}
    for key, value in env.items():
        monkeypatch.setenv(key, value)
    return Settings(_env_file=None)  # type: ignore[call-arg]


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


def test_settings_loads_from_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Settings reads values correctly from environment variables."""
    monkeypatch.setenv("LOG_LEVEL", "DEBUG")
    settings = _make_settings(monkeypatch)

    assert settings.rabbitmq_url == "amqp://guest:guest@localhost:5672/"
    assert "ct_spleen_segmentation" in settings.allowed_analysis_types_list
    assert settings.image_storage_backend == "s3"
    assert settings.log_level == "DEBUG"


def test_settings_missing_required_raises(monkeypatch: pytest.MonkeyPatch) -> None:
    """Missing RABBITMQ_URL raises a ValidationError (fail-fast on startup)."""
    monkeypatch.setenv("ALLOWED_ANALYSIS_TYPES", "ct_spleen_segmentation")
    monkeypatch.setenv("IMAGE_STORAGE_BACKEND", "s3")
    monkeypatch.delenv("RABBITMQ_URL", raising=False)

    with pytest.raises(ValidationError):
        Settings(_env_file=None)  # type: ignore[call-arg]


def test_settings_defaults(monkeypatch: pytest.MonkeyPatch) -> None:
    """RABBITMQ_PREFETCH defaults to 1 and MODEL_DEVICE defaults to 'cpu'."""
    settings = _make_settings(monkeypatch)

    assert settings.rabbitmq_prefetch == 1
    assert settings.model_device == "cpu"
    assert settings.rabbitmq_request_queue == "ai.analysis.requests"
    assert settings.bundle_source == "monaihosting"
