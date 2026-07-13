"""Domain exception hierarchy for the SINAPSIS AI microservice.

All exceptions inherit from SinapsisAIError, allowing callers to catch
the base class for any domain error or a specific subclass for targeted handling.

Usage in the Presentation layer (consumer.py):
  - InvalidMessageError, UnknownAnalysisTypeError → nack without requeue (dead-letter)
  - BundleResolutionError, ImageAccessError, InferenceError → may allow retry

This module has ZERO external dependencies (stdlib only).
"""


class SinapsisAIError(Exception):
    """Base exception for all domain errors in the SINAPSIS AI microservice."""


class InvalidMessageError(SinapsisAIError):
    """The incoming message payload does not conform to the expected schema.

    Non-recoverable: re-queueing the same message will keep failing.
    Action: nack without requeue → dead-letter queue.
    """


class UnknownAnalysisTypeError(SinapsisAIError):
    """The requested analysis_type is not in the set of allowed types.

    Non-recoverable: the analysis type will not become valid by retrying.
    Action: nack without requeue → dead-letter queue.
    """


class BundleResolutionError(SinapsisAIError):
    """The MONAI bundle for the requested analysis type could not be resolved.

    May be transient (network issue during download) or permanent (bundle removed).
    Action: retry policy applies; dead-letter after max attempts.
    """


class ImageAccessError(SinapsisAIError):
    """The input image could not be retrieved or an output artifact could not be saved.

    May be transient (storage temporarily unavailable) or permanent (URI not found).
    Action: retry policy applies; dead-letter after max attempts.
    """


class InferenceError(SinapsisAIError):
    """The MONAI inference pipeline raised an exception during execution.

    The AnalysisService catches this and publishes an AnalysisResult with
    status='failed' and an error block. The consumer then acks the message so
    the backend receives the failure notification and traceability is preserved.
    """
