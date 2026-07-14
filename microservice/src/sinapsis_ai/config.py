"""Application configuration loaded from environment variables.

All settings are read via pydantic-settings. No sensitive defaults.
The process fails fast at startup if required variables are missing.

Note on ALLOWED_ANALYSIS_TYPES:
  pydantic-settings v2 parses list fields as JSON by default.
  To support the more ergonomic comma-separated format in .env files
  (e.g. ``ALLOWED_ANALYSIS_TYPES=ct_spleen_segmentation,mr_brain``),
  we receive the raw value as a str and split it via a model validator.
"""

from pydantic import model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Typed configuration for the SINAPSIS AI microservice.

    Required variables (no defaults — process will not start without them):
      RABBITMQ_URL, ALLOWED_ANALYSIS_TYPES, IMAGE_STORAGE_BACKEND
    """

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # --- RabbitMQ ---
    rabbitmq_url: str
    rabbitmq_request_queue: str = "ai.analysis.requests"
    rabbitmq_result_exchange: str = "sinapsis.ai"
    rabbitmq_result_routing_key: str = "ai.analysis.result"
    rabbitmq_prefetch: int = 1
    rabbitmq_max_retries: int = 3
    rabbitmq_dlx_name: str = "sinapsis.ai.dlx"

    # --- MONAI Bundles ---
    bundle_cache_dir: str = "./.bundle_cache"
    bundle_source: str = "monaihosting"

    # Received as a comma-separated string from env; converted to list by validator.
    allowed_analysis_types: str

    # --- Inference ---
    model_device: str = "cpu"

    # --- Image storage ---
    image_storage_backend: str

    # --- Logging ---
    log_level: str = "INFO"

    # --- Production hardening (v1.0.0) ---
    shutdown_timeout_s: int = 30
    health_file: str = "/tmp/sinapsis_health"

    # Parsed list, populated by the model validator below.
    allowed_analysis_types_list: list[str] = []

    @model_validator(mode="after")
    def parse_and_validate_analysis_types(self) -> "Settings":
        """Split the comma-separated ALLOWED_ANALYSIS_TYPES into a validated list."""
        items = [t.strip() for t in self.allowed_analysis_types.split(",") if t.strip()]
        if not items:
            raise ValueError(
                "ALLOWED_ANALYSIS_TYPES must contain at least one non-empty value"
            )
        self.allowed_analysis_types_list = items
        return self
