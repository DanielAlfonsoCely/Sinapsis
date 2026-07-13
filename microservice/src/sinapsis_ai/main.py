"""Composition root for the SINAPSIS AI microservice.

This is the ONLY module that:
  * Loads configuration (Settings).
  * Initialises logging.
  * Declares AMQP topology (DLX, request queue with DLX binding).
  * Instantiates all infrastructure components.
  * Injects them into the application layer (AnalysisService).
  * Mounts the presentation layer (Consumer + RetryPolicy) and starts the loop.
  * Runs the HealthCheck in a background thread.
  * Handles SIGTERM/SIGINT for a clean shutdown (waits for in-flight message).

No other module should import pika.BlockingConnection, infrastructure
classes, or know about the wiring between layers (DESIGN.md §4).
"""

from __future__ import annotations

import logging
import signal
import sys
import time
from types import FrameType

import pika

from sinapsis_ai.application.analysis_service import AnalysisService
from sinapsis_ai.config import Settings
from sinapsis_ai.infrastructure.bundle_registry import BundleRegistry
from sinapsis_ai.infrastructure.healthcheck import HealthCheck
from sinapsis_ai.infrastructure.image_store import LocalImageStore
from sinapsis_ai.infrastructure.inference_engine import MonaiInferenceEngine
from sinapsis_ai.infrastructure.logging import configure_logging
from sinapsis_ai.infrastructure.publisher import RabbitMQPublisher
from sinapsis_ai.presentation.consumer import Consumer
from sinapsis_ai.presentation.retry_policy import RetryPolicy

logger = logging.getLogger(__name__)


def _declare_topology(
    channel: pika.adapters.blocking_connection.BlockingChannel,
    request_queue: str,
    dlx_name: str,
    result_exchange: str,
) -> None:
    """Declare all required AMQP entities.

    - Dead-letter exchange (fanout, durable).
    - Result exchange (direct, durable) where AnalysisResult messages are published.
    - Request queue bound to the DLX so unroutable/rejected messages go there.

    Args:
        channel: Open pika channel.
        request_queue: Name of the queue from which requests are consumed.
        dlx_name: Name of the dead-letter exchange.
        result_exchange: Name of the exchange where results are published.
    """
    channel.exchange_declare(
        exchange=dlx_name,
        exchange_type="fanout",
        durable=True,
    )
    channel.exchange_declare(
        exchange=result_exchange,
        exchange_type="direct",
        durable=True,
    )
    channel.queue_declare(
        queue=request_queue,
        durable=True,
        arguments={"x-dead-letter-exchange": dlx_name},
    )
    logger.info(
        "AMQP topology declared: queue=%s result_exchange=%s dlx=%s",
        request_queue,
        result_exchange,
        dlx_name,
    )


def main() -> None:
    """Entry point: configure, wire layers and start the blocking consumer loop."""
    # --- Configuration (fail-fast if required env vars are missing) ---
    settings = Settings()  # type: ignore[call-arg]
    configure_logging(settings.log_level)

    logger.info(
        "SINAPSIS AI microservice starting: device=%s queue=%s prefetch=%d "
        "max_retries=%d",
        settings.model_device,
        settings.rabbitmq_request_queue,
        settings.rabbitmq_prefetch,
        settings.rabbitmq_max_retries,
    )

    # --- Transport layer: single shared connection + channel ---
    connection = pika.BlockingConnection(pika.URLParameters(settings.rabbitmq_url))
    channel = connection.channel()

    # --- AMQP topology: DLX + request queue with dead-letter binding ---
    _declare_topology(
        channel=channel,
        request_queue=settings.rabbitmq_request_queue,
        dlx_name=settings.rabbitmq_dlx_name,
        result_exchange=settings.rabbitmq_result_exchange,
    )

    # --- Infrastructure components ---
    bundle_registry = BundleRegistry(
        cache_dir=settings.bundle_cache_dir,
        source=settings.bundle_source,
    )
    image_store = LocalImageStore(root_dir=settings.bundle_cache_dir)
    inference_engine = MonaiInferenceEngine(device=settings.model_device)
    publisher = RabbitMQPublisher(
        channel=channel,
        result_exchange=settings.rabbitmq_result_exchange,
        result_routing_key=settings.rabbitmq_result_routing_key,
    )

    # --- Application layer ---
    service = AnalysisService(
        bundle_resolver=bundle_registry,
        image_store=image_store,
        inference_engine=inference_engine,
        publisher=publisher,
    )

    # --- Presentation layer ---
    retry_policy = RetryPolicy(max_retries=settings.rabbitmq_max_retries)
    consumer = Consumer(service=service, channel=channel, retry_policy=retry_policy)
    consumer.start_consuming(
        queue=settings.rabbitmq_request_queue,
        prefetch=settings.rabbitmq_prefetch,
    )

    # --- HealthCheck (background thread) ---
    healthcheck = HealthCheck(
        connection=connection,
        bundle_cache_dir=settings.bundle_cache_dir,
        health_file=settings.health_file,
    )
    healthcheck.start_background()

    # --- Graceful shutdown on SIGTERM / SIGINT ---
    def _shutdown(signum: int, frame: FrameType | None) -> None:
        sig_name = signal.Signals(signum).name
        logger.info("Received %s — waiting for in-flight message to finish.", sig_name)

        deadline = time.monotonic() + settings.shutdown_timeout_s
        while consumer.inflight and time.monotonic() < deadline:
            time.sleep(0.1)

        if consumer.inflight:
            logger.warning(
                "Shutdown timeout (%ds) exceeded with message still in flight — "
                "closing anyway.",
                settings.shutdown_timeout_s,
            )
        else:
            logger.info("No in-flight message — closing cleanly.")

        try:
            channel.stop_consuming()
            connection.close()
        except Exception as exc:  # noqa: BLE001
            logger.warning("Error during shutdown: %s", exc)
        sys.exit(0)

    signal.signal(signal.SIGTERM, _shutdown)
    signal.signal(signal.SIGINT, _shutdown)

    logger.info("Consumer started — waiting for messages.")
    channel.start_consuming()


if __name__ == "__main__":
    main()
