"""Unit tests for presentation.retry_policy (RetryPolicy).

No real sleeps: backoff_s is set to 0 in all tests.
"""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from sinapsis_ai.domain.errors import (
    BundleResolutionError,
    ImageAccessError,
    InferenceError,
    InvalidMessageError,
    UnknownAnalysisTypeError,
)
from sinapsis_ai.presentation.retry_policy import RetryPolicy

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _policy(max_retries: int = 3) -> RetryPolicy:
    """RetryPolicy with zero backoff for fast tests."""
    return RetryPolicy(max_retries=max_retries, backoff_s=0.0)


def _callable_returning(value: object) -> MagicMock:
    fn = MagicMock(return_value=value)
    return fn


def _callable_raising(exc: Exception) -> MagicMock:
    fn = MagicMock(side_effect=exc)
    return fn


def _callable_failing_then_succeeding(
    failures: list[Exception], success_value: object = "ok"
) -> MagicMock:
    """Raises each exception in *failures* in turn, then returns *success_value*."""
    fn = MagicMock(side_effect=[*failures, success_value])
    return fn


# ---------------------------------------------------------------------------
# Tests — HU1 (RetryPolicy)
# ---------------------------------------------------------------------------


def test_retry_policy_succeeds_on_first_attempt() -> None:
    """Callable that succeeds immediately: called once, returns value."""
    policy = _policy(max_retries=3)
    fn = _callable_returning("result")

    result = policy.execute(fn)

    assert result == "result"
    fn.assert_called_once()


def test_retry_policy_retries_on_transient_bundle_error() -> None:
    """BundleResolutionError on first two calls, success on third."""
    policy = _policy(max_retries=3)
    fn = _callable_failing_then_succeeding(
        [BundleResolutionError("net err"), BundleResolutionError("net err")],
        success_value="done",
    )

    result = policy.execute(fn)

    assert result == "done"
    assert fn.call_count == 3


def test_retry_policy_retries_on_transient_image_error() -> None:
    """ImageAccessError is also retried."""
    policy = _policy(max_retries=2)
    fn = _callable_failing_then_succeeding(
        [ImageAccessError("s3 unavailable")],
        success_value="done",
    )

    result = policy.execute(fn)

    assert result == "done"
    assert fn.call_count == 2


def test_retry_policy_stops_after_max_attempts_raises() -> None:
    """Always-failing transient error: raises after exactly max_retries calls."""
    policy = _policy(max_retries=3)
    fn = _callable_raising(BundleResolutionError("persistent"))

    with pytest.raises(BundleResolutionError):
        policy.execute(fn)

    assert fn.call_count == 3


def test_retry_policy_does_not_retry_invalid_message_error() -> None:
    """InvalidMessageError is non-transient: re-raised on first attempt."""
    policy = _policy(max_retries=3)
    fn = _callable_raising(InvalidMessageError("bad json"))

    with pytest.raises(InvalidMessageError):
        policy.execute(fn)

    fn.assert_called_once()


def test_retry_policy_does_not_retry_unknown_analysis_type() -> None:
    """UnknownAnalysisTypeError is non-transient: re-raised immediately."""
    policy = _policy(max_retries=3)
    fn = _callable_raising(UnknownAnalysisTypeError("unknown"))

    with pytest.raises(UnknownAnalysisTypeError):
        policy.execute(fn)

    fn.assert_called_once()


def test_retry_policy_does_not_retry_inference_error() -> None:
    """InferenceError is handled by AnalysisService, not retried here."""
    policy = _policy(max_retries=3)
    fn = _callable_raising(InferenceError("cuda oom"))

    with pytest.raises(InferenceError):
        policy.execute(fn)

    fn.assert_called_once()


def test_retry_policy_zero_retries_raises_immediately_on_transient() -> None:
    """max_retries=0 with transient error: called once, raises immediately."""
    policy = _policy(max_retries=0)
    fn = _callable_raising(BundleResolutionError("err"))

    with pytest.raises(BundleResolutionError):
        policy.execute(fn)

    fn.assert_called_once()


def test_retry_policy_logs_each_retry(caplog: pytest.LogCaptureFixture) -> None:
    """Each retry attempt is logged with attempt number and max_retries."""
    import logging

    policy = _policy(max_retries=3)
    fn = _callable_failing_then_succeeding(
        [BundleResolutionError("err1"), BundleResolutionError("err2")],
        success_value="ok",
    )

    with caplog.at_level(
        logging.WARNING, logger="sinapsis_ai.presentation.retry_policy"
    ):
        policy.execute(fn)

    # Two retries → two warning log records
    retry_records = [r for r in caplog.records if "attempt" in r.message]
    assert len(retry_records) == 2
    assert "max_retries" in retry_records[0].message


def test_retry_policy_sleeps_between_retries(monkeypatch: pytest.MonkeyPatch) -> None:
    """backoff_s > 0 causes time.sleep to be called between retries."""
    sleep_mock = MagicMock()
    monkeypatch.setattr("sinapsis_ai.presentation.retry_policy.time.sleep", sleep_mock)

    policy = RetryPolicy(max_retries=3, backoff_s=0.5)
    fn = _callable_failing_then_succeeding(
        [BundleResolutionError("err")],
        success_value="ok",
    )

    policy.execute(fn)

    sleep_mock.assert_called_once_with(0.5)
