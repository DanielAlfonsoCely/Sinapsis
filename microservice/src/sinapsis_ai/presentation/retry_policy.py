"""Retry policy for transient errors in the Presentation layer.

Encapsulates bounded retry logic with backoff for errors that may be
transient (network issues, storage temporarily unavailable). Non-transient
errors are re-raised immediately without any retry.

Classification (from domain/errors.py docstrings):
  Transient  → BundleResolutionError, ImageAccessError
  Non-transient → InvalidMessageError, UnknownAnalysisTypeError, InferenceError
    * InferenceError is caught and handled by AnalysisService itself; it never
      propagates to the consumer as a raw exception, so it is treated as
      non-transient here for safety.

This module belongs to the Presentation layer because it controls what happens
to the AMQP message (ack / nack / retry) — a transport-level decision.
It does NOT import pika directly; the consumer owns the channel.
"""

from __future__ import annotations

import logging
import time
from collections.abc import Callable
from typing import TypeVar

from sinapsis_ai.domain.errors import BundleResolutionError, ImageAccessError

logger = logging.getLogger(__name__)

T = TypeVar("T")

# Errors that warrant a retry (transient by nature).
_TRANSIENT_ERRORS: tuple[type[Exception], ...] = (
    BundleResolutionError,
    ImageAccessError,
)


class RetryPolicy:
    """Execute a callable with bounded retries for transient domain errors.

    Args:
        max_retries: Maximum number of times to call the callable (including
            the first attempt). ``1`` means no retries; ``0`` means call once
            and raise immediately on any error.
        backoff_s: Seconds to sleep between attempts (default 0.5 s).
            Set to ``0.0`` in tests to avoid real sleeps.
    """

    def __init__(self, max_retries: int = 3, backoff_s: float = 0.5) -> None:
        self._max_retries = max_retries
        self._backoff_s = backoff_s

    def execute(self, fn: Callable[[], T]) -> T:
        """Call *fn* up to *max_retries* times for transient errors.

        Non-transient exceptions are re-raised immediately without retry.

        Args:
            fn: Zero-argument callable to invoke.

        Returns:
            The return value of *fn* on success.

        Raises:
            The last exception raised by *fn* after exhausting retries
            (transient), or the first exception if non-transient.
        """
        last_exc: Exception | None = None
        attempts = max(self._max_retries, 1) if self._max_retries > 0 else 1

        for attempt in range(1, attempts + 1):
            try:
                return fn()
            except _TRANSIENT_ERRORS as exc:
                last_exc = exc
                if attempt >= attempts:
                    # Retries exhausted — propagate to caller (→ nack).
                    break
                logger.warning(
                    "Transient error on attempt=%d max_retries=%d error=%s: %s",
                    attempt,
                    self._max_retries,
                    type(exc).__name__,
                    exc,
                )
                if self._backoff_s > 0:
                    time.sleep(self._backoff_s)
            except Exception:
                # Non-transient: re-raise immediately, no retry.
                raise

        assert last_exc is not None  # always set when we reach here
        raise last_exc
