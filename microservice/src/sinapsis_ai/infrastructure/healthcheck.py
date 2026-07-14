"""Liveness and readiness healthcheck (Infrastructure layer).

Checks whether the microservice is alive (RabbitMQ connection is open) and
ready (bundle cache directory exists), then writes the result to a file so
Docker's HEALTHCHECK directive can read it with a simple ``cat`` command.

Runs in a background daemon thread started by the composition root (main.py).
This keeps the pika blocking event loop free and avoids any concurrency issues.

Design notes:
  - No PII is logged: only boolean states and paths.
  - The background thread is daemon=True so it does not prevent process exit.
  - Writing to a file (rather than exposing an HTTP endpoint) keeps the
    surface minimal — no new network port, no new dependency.
"""

from __future__ import annotations

import logging
import threading
import time
from pathlib import Path

import pika

logger = logging.getLogger(__name__)


class HealthCheck:
    """Liveness/readiness probe that writes its result to a file.

    Args:
        connection: The active pika BlockingConnection to probe.
        bundle_cache_dir: Path to the bundle cache directory (readiness probe).
        health_file: Path to the file where the health status is written.
            Docker HEALTHCHECK can read it with ``CMD cat <health_file> || exit 1``.
    """

    def __init__(
        self,
        connection: pika.BlockingConnection,
        bundle_cache_dir: str,
        health_file: str,
    ) -> None:
        self._connection = connection
        self._bundle_cache_dir = Path(bundle_cache_dir)
        self._health_file = Path(health_file)

    def is_alive(self) -> bool:
        """Return True if the RabbitMQ connection is open."""
        return bool(self._connection.is_open)

    def is_ready(self) -> bool:
        """Return True if the bundle cache directory exists."""
        return self._bundle_cache_dir.exists()

    def run_check(self) -> None:
        """Run both probes and write the result to *health_file*.

        Writes ``"ok"`` if both liveness and readiness pass, ``"fail"`` otherwise.
        Failures are logged at WARNING level without PII.
        """
        alive = self.is_alive()
        ready = self.is_ready()
        healthy = alive and ready
        status = "ok" if healthy else "fail"

        if not healthy:
            logger.warning(
                "Health check failed: alive=%s ready=%s cache_dir=%s",
                alive,
                ready,
                self._bundle_cache_dir,
            )
        else:
            logger.debug("Health check passed: alive=%s ready=%s", alive, ready)

        try:
            self._health_file.write_text(status)
        except OSError as exc:
            logger.error("Could not write health file %s: %s", self._health_file, exc)

    def start_background(self, interval_s: float = 30.0) -> threading.Thread:
        """Start a daemon thread that calls run_check() every *interval_s* seconds.

        Args:
            interval_s: Seconds between health checks (default 30 s).

        Returns:
            The started Thread object (daemon=True).
        """

        def _loop() -> None:
            while True:
                self.run_check()
                time.sleep(interval_s)

        thread = threading.Thread(target=_loop, daemon=True, name="healthcheck")
        thread.start()
        logger.info(
            "HealthCheck background thread started: interval_s=%.1f health_file=%s",
            interval_s,
            self._health_file,
        )
        return thread
