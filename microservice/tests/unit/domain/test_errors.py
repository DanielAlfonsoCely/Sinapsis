"""Unit tests for sinapsis_ai.domain.errors.

Verifies the exception hierarchy defined in CHANGELOG v0.2.0:
  - test_errors_hierarchy
  - test_invalid_message_error_is_not_unknown_analysis_type
  - test_error_message_preserved
"""

import pytest

from sinapsis_ai.domain.errors import (
    BundleResolutionError,
    ImageAccessError,
    InferenceError,
    InvalidMessageError,
    SinapsisAIError,
    UnknownAnalysisTypeError,
)

_ALL_SUBCLASSES = [
    InvalidMessageError,
    UnknownAnalysisTypeError,
    BundleResolutionError,
    ImageAccessError,
    InferenceError,
]


def test_errors_hierarchy() -> None:
    """All domain exceptions are instances of SinapsisAIError."""
    for cls in _ALL_SUBCLASSES:
        exc = cls("test")
        assert isinstance(exc, SinapsisAIError), (
            f"{cls.__name__} is not a subclass of SinapsisAIError"
        )
        assert isinstance(exc, Exception)


def test_invalid_message_error_selective_catch() -> None:
    """InvalidMessageError is not caught as UnknownAnalysisTypeError."""
    exc = InvalidMessageError("bad payload")

    assert isinstance(exc, InvalidMessageError)
    assert isinstance(exc, SinapsisAIError)
    assert not isinstance(exc, UnknownAnalysisTypeError)

    # Verify try/except block behaviour
    caught_as_invalid = False
    caught_as_unknown = False

    try:
        raise exc
    except UnknownAnalysisTypeError:
        caught_as_unknown = True
    except InvalidMessageError:
        caught_as_invalid = True

    assert caught_as_invalid
    assert not caught_as_unknown


def test_error_message_preserved() -> None:
    """The error message passed to the constructor is accessible via str()."""
    message = "model exploded during inference"
    exc = InferenceError(message)
    assert message in str(exc)


@pytest.mark.parametrize("cls", _ALL_SUBCLASSES)
def test_each_subclass_is_catchable_as_base(cls: type[SinapsisAIError]) -> None:
    """Each subclass can be caught with 'except SinapsisAIError'."""
    caught = False
    try:
        raise cls("test error")
    except SinapsisAIError:
        caught = True
    assert caught
