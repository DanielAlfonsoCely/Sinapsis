"""RabbitMQ result publisher (Infrastructure layer).

Publishes an AnalysisResult as a persistent AMQP message to the configured
exchange. Implements the ResultPublisher port defined in application/ports.py.

Design notes:
  - delivery_mode=2 (persistent): results survive broker restarts before the
    backend Go service consumes them.
  - Serialisation is delegated to presentation/schemas.serialise_result so the
    JSON contract lives in exactly one place.
  - The channel is injected by the composition root (main.py); this class never
    opens connections — that is main.py's responsibility.
  - No PII is logged: only request_id, study_id and status.
"""

from __future__ import annotations

import json
import logging

import pika
import pika.spec

from sinapsis_ai.domain.models import AnalysisResult
from sinapsis_ai.presentation.schemas import serialise_result

logger = logging.getLogger(__name__)


class RabbitMQPublisher:
    """Publishes AnalysisResult messages to RabbitMQ persistently.

    Args:
        channel: An open pika BlockingChannel.
        result_exchange: Exchange name where results are published.
        result_routing_key: Routing key for result messages.
    """

    def __init__(
        self,
        channel: pika.adapters.blocking_connection.BlockingChannel,
        result_exchange: str,
        result_routing_key: str,
    ) -> None:
        self._channel = channel
        self._exchange = result_exchange
        self._routing_key = result_routing_key

    def publish(self, result: AnalysisResult) -> None:
        """Serialise and publish *result* as a persistent AMQP message.

        Args:
            result: The domain AnalysisResult to publish.

        Raises:
            pika.exceptions.AMQPError: If the publish fails at the transport
                level (caller — the consumer — decides how to handle it).
        """
        body = json.dumps(serialise_result(result)).encode("utf-8")
        self._channel.basic_publish(
            exchange=self._exchange,
            routing_key=self._routing_key,
            body=body,
            properties=pika.BasicProperties(
                content_type="application/json",
                delivery_mode=2,  # persistent
            ),
        )
        logger.info(
            "Result published: request_id=%s study_id=%s status=%s exchange=%s",
            result.request_id,
            result.study_id,
            result.status,
            self._exchange,
        )
