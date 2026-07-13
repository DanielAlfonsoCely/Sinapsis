"""Structured logging configuration for the SINAPSIS AI microservice.

Rules:
- Never log PII, patient data or image pixels.
- Only log opaque identifiers (request_id, study_id, correlation_id) and metadata.
"""

import logging


def configure_logging(level: str = "INFO") -> None:
    """Configure root logger with a structured format.

    Args:
        level: Logging level string (DEBUG, INFO, WARNING, ERROR, CRITICAL).
               Invalid values fall back to INFO.
    """
    numeric_level = getattr(logging, level.upper(), logging.INFO)
    logging.basicConfig(
        level=numeric_level,
        format="%(asctime)s %(levelname)-8s %(name)s %(message)s",
        datefmt="%Y-%m-%dT%H:%M:%S",
    )
    logging.getLogger(__name__).debug("Logging configured at level %s", level.upper())
