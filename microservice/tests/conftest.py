"""Shared pytest fixtures for the SINAPSIS AI test suite."""

import pytest

from sinapsis_ai.config import Settings

# Minimal valid environment — no .env file read, no real credentials.
_BASE_ENV = {
    "RABBITMQ_URL": "amqp://guest:guest@localhost:5672/",
    "ALLOWED_ANALYSIS_TYPES": "ct_spleen_segmentation",
    "IMAGE_STORAGE_BACKEND": "s3",
}


@pytest.fixture()
def test_settings(monkeypatch: pytest.MonkeyPatch) -> Settings:
    """Return a Settings instance built from a minimal fake environment.

    Does NOT read any .env file from disk.
    """
    for key, value in _BASE_ENV.items():
        monkeypatch.setenv(key, value)
    # Disable .env file loading for tests
    return Settings(_env_file=None)  # type: ignore[call-arg]
