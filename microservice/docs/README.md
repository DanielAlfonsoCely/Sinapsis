# SINAPSIS AI Microservice

![status](https://img.shields.io/badge/status-scaffolding-lightgrey)
![version](https://img.shields.io/badge/version-v0.1.0--pending-blue)
![python](https://img.shields.io/badge/python-3.12-blue)
![package manager](https://img.shields.io/badge/deps-uv-purple)
![inference](https://img.shields.io/badge/inference-MONAI-e11d63)

Microservicio **Python** que ejecuta **modelos MONAI pre-entrenados** para análisis de
imagen médica dentro de **SINAPSIS** (*Sistema Inteligente de Análisis de Patrones de Salud
Integrados*). Corre como **consumidor de RabbitMQ**: recibe una solicitud, ejecuta la
inferencia y publica el resultado. Es **stateless** y **no toma decisiones clínicas**.

> Documentación relacionada:
> [`GLOBAL_ARCHITECTURE.md`](./GLOBAL_ARCHITECTURE.md) (sistema completo) ·
> [`PROJECT_ARCHITECTURE.md`](./PROJECT_ARCHITECTURE.md) (estructura interna) ·
> [`DESIGN.md`](./DESIGN.md) (decisiones de diseño) ·
> [`CHANGELOG.md`](./CHANGELOG.md) (roadmap).

---

## Estado de versiones

| Versión | Objetivo | Estado |
|---------|----------|--------|
| v0.1.0 | Scaffolding (tooling, config, CI) | PENDIENTE |
| v0.2.0 | Domain + contrato de mensajes (schemas) | PENDIENTE |
| v0.3.0 | Infra: registry/caché de bundles MONAI | PENDIENTE |
| v0.4.0 | Infra inferencia + Application service | PENDIENTE |
| v0.5.0 | Presentation (consumer) + publisher + wiring | PENDIENTE |
| v1.0.0 | Endurecimiento para producción | PENDIENTE |

Detalle por versión en [`CHANGELOG.md`](./CHANGELOG.md).

---

## Arquitectura

Posición del servicio en el sistema (detalle en [`GLOBAL_ARCHITECTURE.md`](./GLOBAL_ARCHITECTURE.md)):

```
backend (Go) ──image──► RabbitMQ ──► [ AI microservice · MONAI ]
     ▲                     │
     └───────async─────────┘  (result)
```

El backend valida pre-diagnóstico, rol del doctor y auditoría **antes** de encolar; este
servicio solo ejecuta el modelo. Internamente sigue una **arquitectura por capas simple**
(dependencias hacia abajo):

```
Presentation (consumer + schemas)
      │
      ▼
Application  (analysis_service)  ──► Domain (models, errors)
      │
      ▼
Infrastructure (publisher · bundle_registry · inference_engine · image_store · logging)
```

Flujo dentro del servicio:

```
consumer(valida) → analysis_service(orquesta):
    bundle_registry(resuelve modelo) → image_store(imagen)
    → inference_engine(inferencia) → construye AnalysisResult → publisher
```

---

## Stack tecnológico

| Área | Tecnología | Justificación |
|------|------------|---------------|
| Lenguaje | Python 3.12 | Ecosistema de imagen médica / ML |
| Gestor de entorno | uv | Rápido y reproducible (`uv.lock`) |
| Inferencia | MONAI + PyTorch | Ejecuta bundles pre-entrenados del Model Zoo |
| Mensajería | pika (RabbitMQ) | Desacople asíncrono del backend |
| Config/validación | pydantic + pydantic-settings | Tipado y carga desde entorno |
| Tests | pytest + pytest-cov | Unit + integración marcada |
| Lint/format | ruff | Formato y linting |
| Tipos | mypy | Chequeo estático |

---

## Instalación y ejecución

Requiere **Python 3.12** y **uv** instalado. Para integración, Docker.

```bash
uv sync                              # instalar/sincronizar entorno
cp .env.example .env                 # configurar variables (no commitear .env)

docker compose up -d rabbitmq        # infraestructura local (RabbitMQ)
uv run python -m sinapsis_ai.main    # arrancar el consumidor
```

---

## Interfaz (asíncrona vía RabbitMQ)

No expone HTTP. Consume `AnalysisRequest` y publica `AnalysisResult`
(esquemas completos en [`PROJECT_ARCHITECTURE.md`](./PROJECT_ARCHITECTURE.md) §9).

**Consume** (cola `RABBITMQ_REQUEST_QUEUE`):
```json
{ "request_id": "uuid", "study_id": "opaque", "patient_ref": "opaque",
  "analysis_type": "ct_spleen_segmentation", "image_uri": "s3://.../scan.nii.gz",
  "requested_by": "doctor-opaque", "correlation_id": "uuid", "issued_at": "..." }
```

**Publica** (exchange `RABBITMQ_RESULT_EXCHANGE`):
```json
{ "request_id": "uuid", "study_id": "opaque", "status": "succeeded",
  "model": {"name": "spleen_ct_segmentation", "version": "0.5.3"},
  "artifacts": [{"type": "segmentation_mask", "uri": "s3://.../mask.nii.gz"}],
  "metrics": {"volume_ml": 210.4}, "error": null,
  "processed_at": "...", "duration_ms": 42000 }
```

---

## Flujo principal

```
Backend (Go)          RabbitMQ            AI microservice (MONAI)
     │  publish request   │                        │
     │───────────────────►│  deliver               │
     │                    │───────────────────────►│  valida, resuelve bundle
     │                    │                        │  descarga imagen (URI)
     │                    │                        │  inferencia (pre→red→infer→post)
     │                    │◄───────────────────────│  publish result
     │◄───────────────────│  deliver result (async)│  ack
```

---

## Testing

Orden de verificación (de [`AGENTS.md`](../AGENTS.md)):

```bash
uv run ruff format --check .
uv run ruff check .
uv run mypy src
uv run pytest                                 # unit (integración omitida por defecto)
```

- Un solo test:
  `uv run pytest tests/unit/application/test_analysis_service.py::test_run_analysis_publishes_result -q`
- Cobertura: `uv run pytest --cov=sinapsis_ai --cov-report=term-missing`
- Integración (RabbitMQ real): `docker compose up -d rabbitmq && uv run pytest -m integration`

Convenciones: `tests/unit/` **espeja las capas** de `src/`
(`tests/unit/<capa>/test_<módulo>.py`); integración marcada `@pytest.mark.integration`.

---

## Patrones de diseño

| Patrón | Dónde |
|--------|-------|
| Layered Architecture | `presentation/` → `application/` → `domain/`; `infrastructure/` → `domain/` |
| Service / Use Case | `application/analysis_service.py` |
| Registry + Cache | `infrastructure/bundle_registry.py` |
| Strategy | `infrastructure/inference_engine.py` (bundle = estrategia) |
| DTO + Translator | `presentation/schemas.py` |
| Composition Root | `main.py` |
| Exception Hierarchy | `domain/errors.py` |

Detalle en [`DESIGN.md`](./DESIGN.md).

---

## Extensibilidad

Agregar un tipo de análisis (resumen; ver [`DESIGN.md`](./DESIGN.md) §11):
1. Nuevo miembro en `AnalysisType` (`domain/models.py`).
2. Mapeo `AnalysisType → bundle` en `infrastructure/bundle_registry.py`.
3. Extender la construcción del resultado en `application/analysis_service.py` si la salida
   es nueva.
4. Habilitarlo en `ALLOWED_ANALYSIS_TYPES` y añadir tests espejo por capa.

Escalar: más réplicas del consumidor (cada una `prefetch=1`); no subir `prefetch`.

---

## Estructura de directorios (resumen)

```
src/sinapsis_ai/            # arquitectura por capas (deps hacia abajo)
├── main.py                 # composition root + arranque
├── config.py               # settings (env)
├── presentation/           # consumer, schemas (RabbitMQ, borde de entrada)
├── application/            # analysis_service (orquestación / caso de uso)
├── domain/                 # models, errors (núcleo puro)
└── infrastructure/         # publisher, bundle_registry, inference_engine, image_store, logging
tests/unit/{presentation,application,domain,infrastructure}/  ·  tests/integration/  ·  tests/fixtures/
```

Árbol completo en [`PROJECT_ARCHITECTURE.md`](./PROJECT_ARCHITECTURE.md) §3.

---

## Variables de entorno

| Variable | Propósito | Default |
|----------|-----------|---------|
| `RABBITMQ_URL` | URL AMQP con credenciales | *(requerida)* |
| `RABBITMQ_REQUEST_QUEUE` | Cola de solicitudes | `ai.analysis.requests` |
| `RABBITMQ_RESULT_EXCHANGE` | Exchange de resultados | `sinapsis.ai` |
| `RABBITMQ_RESULT_ROUTING_KEY` | Routing key de resultados | `ai.analysis.result` |
| `RABBITMQ_PREFETCH` | Mensajes en vuelo por worker | `1` |
| `BUNDLE_CACHE_DIR` | Caché de bundles MONAI | `./.bundle_cache` |
| `BUNDLE_SOURCE` | Fuente de descarga | `monaihosting` |
| `ALLOWED_ANALYSIS_TYPES` | Tipos habilitados (→ bundles) | *(requerida)* |
| `MODEL_DEVICE` | `cuda` / `cpu` | `cpu` |
| `IMAGE_STORAGE_BACKEND` | Backend de imágenes/artefactos | *(requerida)* |
| `LOG_LEVEL` | Nivel de logging | `INFO` |

No commitear `.env`. Plantilla en `.env.example`.
