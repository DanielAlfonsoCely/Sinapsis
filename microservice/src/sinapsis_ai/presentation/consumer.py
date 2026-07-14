"""RabbitMQ message consumer (Presentation layer).

Receives analysis request messages from the broker, validates them via the
schemas module, delegates to AnalysisService (wrapped in RetryPolicy) and
emits the appropriate AMQP acknowledgement.

Ack/nack policy (DESIGN.md §7):
  * InvalidMessageError / UnknownAnalysisTypeError → nack(requeue=False) → dead-letter.
    Non-transient: retrying can never succeed.
  * BundleResolutionError / ImageAccessError → RetryPolicy retries up to max_retries;
    if retries are exhausted → nack(requeue=False) → dead-letter.
  * InferenceError → handled internally by AnalysisService; returns a failed
    AnalysisResult → consumer acks.
  * Any other unexpected exception → nack(requeue=False) and log; prevents crash.
  * Normal completion (succeeded or service-handled failure) → ack.

Design notes:
  - The channel and RetryPolicy are injected by the composition root (main.py).
  - prefetch=1 is intentional: one heavy inference per worker (DESIGN.md §3).
  - _inflight flag lets the shutdown handler wait for the in-flight message.
"""

from __future__ import annotations

import logging

import pika
import pika.adapters.blocking_connection
import pika.spec

from sinapsis_ai.application.analysis_service import AnalysisService
from sinapsis_ai.domain.errors import (
    BundleResolutionError,
    ImageAccessError,
    InvalidMessageError,
    UnknownAnalysisTypeError,
)
from sinapsis_ai.presentation.retry_policy import RetryPolicy
from sinapsis_ai.presentation.schemas import parse_request

logger = logging.getLogger(__name__)


class Consumer:
    """Stateless RabbitMQ consumer that delegates analysis to AnalysisService.

    Args:
        service: The AnalysisService use-case orchestrator (injected).
        channel: An open pika BlockingChannel (injected by composition root).
        retry_policy: RetryPolicy for transient errors (injected). Defaults to
            a policy with max_retries=3 and backoff_s=0.5.
    """

    def __init__(
        self,
        service: AnalysisService,
        channel: pika.adapters.blocking_connection.BlockingChannel,
        retry_policy: RetryPolicy | None = None,
    ) -> None:
        self._service = service
        self._channel = channel
        self._retry_policy = retry_policy or RetryPolicy()
        self._inflight: bool = False

    @property
    def inflight(self) -> bool:
        """True while a message is being processed."""
        return self._inflight

    def start_consuming(self, queue: str, prefetch: int = 1) -> None:
        """Configure QoS and register the message callback.

        Does NOT start the blocking event loop; call
        ``channel.start_consuming()`` after this in the composition root so
        that signal handlers can be registered between the two calls.

        Args:
            queue: The AMQP queue name to consume from.
            prefetch: Number of unacknowledged messages at a time (always 1).
        """
        self._channel.basic_qos(prefetch_count=prefetch)
        self._channel.basic_consume(
            queue=queue,
            on_message_callback=self._on_message,
        )
        logger.info(
            "Consumer registered: queue=%s prefetch=%d",
            queue,
            prefetch,
        )

    def _on_message(
        self,
        ch: pika.adapters.blocking_connection.BlockingChannel,
        method: pika.spec.Basic.Deliver,
        properties: pika.spec.BasicProperties,
        body: bytes,
    ) -> None:
        """AMQP message callback — validate, delegate (with retries) and ack/nack."""
        delivery_tag = method.delivery_tag
        self._inflight = True

        try:
            # --- Validation (Presentation → Domain) ---
            try:
                request = parse_request(body)
            except UnknownAnalysisTypeError as exc:
                logger.warning(
                    "Dead-lettering message (unknown analysis type): %s "
                    "delivery_tag=%s",
                    exc,
                    delivery_tag,
                )
                ch.basic_nack(delivery_tag=delivery_tag, requeue=False)
                return
            except InvalidMessageError as exc:
                logger.warning(
                    "Dead-lettering message (invalid payload): %s delivery_tag=%s",
                    exc,
                    delivery_tag,
                )
                ch.basic_nack(delivery_tag=delivery_tag, requeue=False)
                return

            # --- Use-case delegation with retry policy ---
            try:
                result = self._retry_policy.execute(
                    lambda: self._service.run_analysis(request)
                )
            except (BundleResolutionError, ImageAccessError) as exc:
                logger.error(
                    "Transient error exhausted retries: request_id=%s "
                    "correlation_id=%s error=%s: %s — dead-lettering",
                    request.request_id,
                    request.correlation_id,
                    type(exc).__name__,
                    exc,
                )
                ch.basic_nack(delivery_tag=delivery_tag, requeue=False)
                return
            except Exception as exc:  # noqa: BLE001
                logger.error(
                    "Unexpected error: request_id=%s correlation_id=%s: %s",
                    request.request_id,
                    request.correlation_id,
                    exc,
                    exc_info=True,
                )
                ch.basic_nack(delivery_tag=delivery_tag, requeue=False)
                return

            # --- Ack: succeeded or service-handled failure ---
            logger.info(
                "Message processed: request_id=%s status=%s delivery_tag=%s",
                result.request_id,
                result.status,
                delivery_tag,
            )
            ch.basic_ack(delivery_tag=delivery_tag)

        finally:
            self._inflight = False
