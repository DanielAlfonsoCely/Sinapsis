# CHANGELOG — SINAPSIS AI Microservice

Roadmap versionado del microservicio de inferencia MONAI. Sigue **versionado semántico**.
Todas las versiones inician como **PENDIENTE**. Cada versión es desplegable de forma
independiente y construye sobre la anterior.

Comandos de verificación tomados de [`AGENTS.md`](../AGENTS.md); ejecutar **en este orden**:
`ruff format --check` → `ruff check` → `mypy src` → `pytest`.

---

## [v0.1.0] — PENDIENTE
> **Resumen**: Scaffolding. Estructura del proyecto, tooling (uv/ruff/mypy/pytest), CI y
> configuración base. Sin lógica de inferencia todavía.

### Código a Producir
- `pyproject.toml` (proyecto `sinapsis-ai`, layout src/, deps base y dev), `uv.lock`,
  `.python-version` (3.12).
- `src/sinapsis_ai/__init__.py`, `src/sinapsis_ai/main.py` (esqueleto que arranca y sale
  limpio).
- `src/sinapsis_ai/config.py` (`Settings` con pydantic-settings + campos del §8 de
  PROJECT_ARCHITECTURE).
- Paquetes de las capas con `__init__.py`: `presentation/`, `application/`, `domain/`,
  `infrastructure/`.
- `src/sinapsis_ai/infrastructure/logging.py` (setup de logging estructurado).
- `.env.example`, `.gitignore` (ignora `.env`, `.bundle_cache/`, artefactos).
- Configuración de `ruff`, `mypy` y markers de pytest (`integration`) en `pyproject.toml`.
- CI (`.github/workflows/ci.yml`) que corre la secuencia de verificación.

### Infraestructura de Testing
- `tests/__init__.py`, `tests/conftest.py` (fixture de `Settings` de prueba).
- `tests/unit/test_config.py`.

### Escenarios de Test Específicos
- `test_settings_loads_from_env` — carga valores desde variables de entorno.
- `test_settings_missing_required_raises` — falta `RABBITMQ_URL` → error de validación
  (fail-fast).
- `test_settings_defaults` — `RABBITMQ_PREFETCH=1`, `MODEL_DEVICE=cpu` por defecto.

### Verificación
```bash
uv sync
uv run ruff format --check .
uv run ruff check .
uv run mypy src
uv run pytest
```

---

## [v0.2.0] — PENDIENTE
> **Resumen**: Capa Domain + contrato de mensajes (Presentation). Entidades y DTOs de
> request/result con validación, sin ejecutar inferencia ni conectar a RabbitMQ.

### Código a Producir
- `src/sinapsis_ai/domain/models.py` (`AnalysisType`, `AnalysisRequest`, `AnalysisResult`,
  `ModelRef`, `Artifact`).
- `src/sinapsis_ai/domain/errors.py` (jerarquía completa: `SinapsisAIError` y subclases).
- `src/sinapsis_ai/presentation/schemas.py` (DTOs pydantic de mensajes + traducción a/desde
  `domain/`).

### Infraestructura de Testing
- `tests/unit/domain/test_models.py`, `tests/unit/domain/test_errors.py`,
  `tests/unit/presentation/test_schemas.py`.
- `tests/fixtures/sample_request.json`.

### Escenarios de Test Específicos
- `test_request_schema_valid_payload` — JSON válido → `AnalysisRequest`.
- `test_request_schema_rejects_missing_image_uri` → `InvalidMessageError`.
- `test_request_schema_rejects_unknown_analysis_type` → `UnknownAnalysisTypeError`.
- `test_result_schema_serializes_success` — `AnalysisResult(status=succeeded)` → JSON del
  contrato (§9 PROJECT_ARCHITECTURE).
- `test_result_schema_serializes_failure` — incluye bloque `error`.
- `test_errors_hierarchy` — todas heredan de `SinapsisAIError`.

### Verificación
```bash
uv run ruff format --check . && uv run ruff check .
uv run mypy src
uv run pytest tests/unit/test_schemas.py -q
```

---

## [v0.3.0] — COMPLETADO (2026-07-05)
> **Resumen**: Infraestructura de modelos MONAI. Registry que mapea `AnalysisType →
> bundle`, descarga y cachea bundles. Sin ejecución de inferencia real en unit tests (mock).

### Código a Producir
- `src/sinapsis_ai/infrastructure/bundle_registry.py` (mapeo, descarga vía `monai.bundle`,
  caché en `BUNDLE_CACHE_DIR`, devuelve `ModelRef`).

### Infraestructura de Testing
- `tests/unit/infrastructure/test_bundle_registry.py` (descarga de MONAI mockeada).

### Escenarios de Test Específicos
- `test_resolve_known_type_returns_modelref`.
- `test_resolve_unknown_type_raises` → `UnknownAnalysisTypeError`.
- `test_bundle_cached_not_redownloaded` — segunda resolución no vuelve a descargar.
- `test_download_failure_raises_bundle_resolution_error`.

### Verificación
```bash
uv run mypy src
uv run pytest tests/unit/infrastructure/test_bundle_registry.py -q
```

---

## [v0.4.0] — COMPLETADO (2026-07-05)
> **Resumen**: Motor de inferencia (Infra) + caso de uso (Application). Ejecuta el pipeline
> del bundle y el `AnalysisService` orquesta y construye el `AnalysisResult`. Torch/MONAI
> mockeados en unit; smoke real en integración.

### Código a Producir
- `src/sinapsis_ai/infrastructure/image_store.py` (obtener imagen por URI, guardar artefactos).
- `src/sinapsis_ai/infrastructure/inference_engine.py` (ejecuta bundle: pre → red → inferer → post).
- `src/sinapsis_ai/application/analysis_service.py` (orquesta el flujo y construye
  `AnalysisResult` con métricas/artefactos).

### Infraestructura de Testing
- `tests/unit/infrastructure/test_image_store.py`,
  `tests/unit/infrastructure/test_inference_engine.py`,
  `tests/unit/application/test_analysis_service.py`.
- `tests/integration/test_inference_smoke.py` (`@pytest.mark.integration`).
- `tests/fixtures/tiny_volume.nii.gz`.

### Escenarios de Test Específicos
- `test_run_analysis_publishes_result` — service con dobles inyectados orquesta y produce
  el `AnalysisResult`.
- `test_image_store_fetch_missing_raises` → `ImageAccessError`.
- `test_inference_engine_failure_raises` → `InferenceError`.
- `test_analysis_service_builds_metrics_and_artifacts`.
- **(integración)** `test_inference_smoke_downloads_bundle_and_runs` — descarga un bundle
  pequeño y ejecuta sobre `tiny_volume.nii.gz`.

### Verificación
```bash
uv run pytest tests/unit/ -q
# integración (requiere red + descarga de bundle):
uv run pytest -m integration tests/integration/test_inference_smoke.py -q
```

---

## [v0.5.0] — COMPLETADO (2026-07-13)
> **Resumen**: Presentation + publicación. Consumidor `pika` (`prefetch=1`), publisher de
> resultados (Infra), política de ack/nack/dead-letter. Cablea las capas en `main.py`.

### Código a Producir
- `src/sinapsis_ai/presentation/consumer.py` (consumir, validar, delegar al service, ack/nack).
- `src/sinapsis_ai/infrastructure/publisher.py` (publicar `AnalysisResult` persistente).
- `src/sinapsis_ai/main.py` (composition root: settings → capas → arranque + apagado limpio).
- `docker-compose.yml` (servicio RabbitMQ), `Dockerfile` (imagen del servicio).

### Infraestructura de Testing
- `tests/unit/presentation/test_consumer.py`,
  `tests/unit/infrastructure/test_publisher.py` (canal `pika` mockeado).
- `tests/integration/test_rabbitmq_flow.py` (`@pytest.mark.integration`).

### Escenarios de Test Específicos
- `test_consumer_valid_message_acks_and_publishes_result`.
- `test_consumer_invalid_message_deadletters` — `nack` sin requeue.
- `test_consumer_inference_error_publishes_failed_result_and_acks`.
- `test_publisher_publishes_persistent_to_result_exchange`.
- **(integración)** `test_rabbitmq_round_trip` — publicar request en cola real →
  consumidor produce y publica result en el exchange de resultados.

### Verificación
```bash
docker compose up -d rabbitmq
uv run ruff format --check . && uv run ruff check .
uv run mypy src
uv run pytest                      # unit
uv run pytest -m integration       # integración con RabbitMQ real
uv run python -m sinapsis_ai.main  # arranque manual del consumidor
```

---

## [v2.0.0] — COMPLETADO (2026-07-13)
> **Resumen**: Multi-model output generalizado. Protocolo `OutputExtractor` con
> `SegmentationExtractor`, `ClassificationExtractor` y `DetectionExtractor`. Tres nuevos
> `AnalysisType` con bundles MONAI reales: `CT_LUNG_NODULE_DETECTION`,
> `MRI_BRAIN_TUMOR_SEGMENTATION`, `XR_BREAST_DENSITY_CLASSIFICATION`.

---

## [v1.0.0] — COMPLETADO (2026-07-13)
> **Resumen**: Endurecimiento para producción. Reintentos acotados, healthcheck de
> liveness/readiness, apagado ordenado bajo carga, cobertura ≥ 85% y documentación de
> despliegue.

### Código a Producir
- Política de reintentos configurable y dead-letter exchange declarado explícitamente.
- Healthcheck (conexión a RabbitMQ / disponibilidad de caché de bundles) para orquestador.
- Manejo robusto de SIGTERM: terminar la inferencia en curso antes de cerrar.
- Métricas básicas (duración de inferencia, tasa de éxito/fallo) en logs estructurados.

### Infraestructura de Testing
- `tests/unit/presentation/test_retry_policy.py`, `tests/unit/test_main.py` (apagado).
- Ampliar integración: reconexión tras caída del broker.

### Escenarios de Test Específicos
- `test_retry_policy_stops_after_max_attempts_deadletters`.
- `test_graceful_shutdown_finishes_inflight_message`.
- `test_reconnect_after_broker_restart` (integración).
- `test_coverage_threshold_met` — cobertura global ≥ 85%.

### Verificación
```bash
uv run ruff format --check . && uv run ruff check .
uv run mypy src
uv run pytest --cov=sinapsis_ai --cov-report=term-missing
uv run pytest -m integration
```
