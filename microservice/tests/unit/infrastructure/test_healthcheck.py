"""Unit tests for infrastructure.healthcheck (HealthCheck).

The pika connection is mocked; filesystem operations use tmp_path.
No real RabbitMQ connection is opened.
"""

from __future__ import annotations

import time
from pathlib import Path
from unittest.mock import MagicMock

from sinapsis_ai.infrastructure.healthcheck import HealthCheck

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


def _open_connection() -> MagicMock:
    conn = MagicMock()
    conn.is_open = True
    return conn


def _closed_connection() -> MagicMock:
    conn = MagicMock()
    conn.is_open = False
    return conn


# ---------------------------------------------------------------------------
# Tests — HU2
# ---------------------------------------------------------------------------


def test_healthcheck_alive_with_open_connection(tmp_path: Path) -> None:
    """is_alive returns True when connection.is_open is True."""
    hc = HealthCheck(
        connection=_open_connection(),
        bundle_cache_dir=str(tmp_path),
        health_file=str(tmp_path / "health"),
    )
    assert hc.is_alive() is True


def test_healthcheck_alive_false_when_connection_closed(tmp_path: Path) -> None:
    """is_alive returns False when connection.is_open is False."""
    hc = HealthCheck(
        connection=_closed_connection(),
        bundle_cache_dir=str(tmp_path),
        health_file=str(tmp_path / "health"),
    )
    assert hc.is_alive() is False


def test_healthcheck_ready_true_when_cache_dir_exists(tmp_path: Path) -> None:
    """is_ready returns True when bundle_cache_dir exists."""
    hc = HealthCheck(
        connection=_open_connection(),
        bundle_cache_dir=str(tmp_path),
        health_file=str(tmp_path / "health"),
    )
    assert hc.is_ready() is True


def test_healthcheck_ready_false_when_cache_dir_missing(tmp_path: Path) -> None:
    """is_ready returns False when bundle_cache_dir does not exist."""
    hc = HealthCheck(
        connection=_open_connection(),
        bundle_cache_dir=str(tmp_path / "nonexistent"),
        health_file=str(tmp_path / "health"),
    )
    assert hc.is_ready() is False


def test_healthcheck_writes_ok_to_file_when_healthy(tmp_path: Path) -> None:
    """run_check writes 'ok' when alive and ready."""
    health_file = tmp_path / "health"
    hc = HealthCheck(
        connection=_open_connection(),
        bundle_cache_dir=str(tmp_path),
        health_file=str(health_file),
    )
    hc.run_check()

    assert health_file.read_text() == "ok"


def test_healthcheck_writes_fail_when_connection_closed(tmp_path: Path) -> None:
    """run_check writes 'fail' when connection is closed."""
    health_file = tmp_path / "health"
    hc = HealthCheck(
        connection=_closed_connection(),
        bundle_cache_dir=str(tmp_path),
        health_file=str(health_file),
    )
    hc.run_check()

    assert health_file.read_text() == "fail"


def test_healthcheck_writes_fail_when_cache_missing(tmp_path: Path) -> None:
    """run_check writes 'fail' when bundle cache dir is missing."""
    health_file = tmp_path / "health"
    hc = HealthCheck(
        connection=_open_connection(),
        bundle_cache_dir=str(tmp_path / "no_cache"),
        health_file=str(health_file),
    )
    hc.run_check()

    assert health_file.read_text() == "fail"


def test_healthcheck_start_background_runs_check(tmp_path: Path) -> None:
    """start_background starts a daemon thread that writes the health file."""
    health_file = tmp_path / "health"
    hc = HealthCheck(
        connection=_open_connection(),
        bundle_cache_dir=str(tmp_path),
        health_file=str(health_file),
    )
    thread = hc.start_background(interval_s=0.05)

    assert thread.daemon is True

    # Wait for the health file to be written with non-empty content
    deadline = time.monotonic() + 2.0
    content = ""
    while time.monotonic() < deadline:
        if health_file.exists():
            content = health_file.read_text()
            if content in ("ok", "fail"):
                break
        time.sleep(0.05)

    assert content in ("ok", "fail"), (
        f"Health file content {content!r} not 'ok' or 'fail' within timeout"
    )
