# Arquitectura del Proyecto — SINAPSIS AI Microservice

> Estructura interna del microservicio de inferencia MONAI (`microservice/`).
> Para el contexto del sistema completo (frontend, BFF, backend Go, cola, datos) ver
> [`GLOBAL_ARCHITECTURE.md`](./GLOBAL_ARCHITECTURE.md), que es la **fuente de verdad** de
> la arquitectura global.

> **Nota de estado**: este documento describe la estructura **planeada** (documentación
> primero). El código se produce por versiones según [`CHANGELOG.md`](./CHANGELOG.md).

---

## 1. Propósito

Servicio Python **stateless** que:

1. **Consume** solicitudes de análisis desde RabbitMQ.
2. **Resuelve y carga** el modelo MONAI pre-entrenado (bundle) adecuado al tipo de análisis.
3. **Ejecuta la inferencia** (pre-proceso → red → inferer → post-proceso).
4. **Publica** el resultado de vuelta a RabbitMQ (o un mensaje de error).

**No** entrena modelos, **no** toma decisiones clínicas y **no** almacena estado clínico.
El backend (Go) ya validó pre-diagnóstico, rol del doctor y auditoría antes de encolar.

---

## 2. Arquitectura por capas (simple)

El microservicio sigue una **arquitectura por capas simple**. Cada capa depende **solo de
la capa inferior**; las dependencias fluyen **hacia abajo**:

```
┌──────────────────────────────────────────────────────────┐
│  Presentation  (presentation/)                            │  ← borde de entrada
│  Consumidor RabbitMQ + esquemas de mensaje                │
└───────────────────────────┬──────────────────────────────┘
                            │ llama a
┌───────────────────────────▼──────────────────────────────┐
│  Application  (application/)                              │  ← orquestación / casos de uso
│  analysis_service: coordina el flujo de análisis          │
└───────────────────────────┬──────────────────────────────┘
                            │ usa
┌───────────────────────────▼──────────────────────────────┐
│  Domain  (domain/)                                        │  ← entidades + reglas + errores
│  models, errors  (sin dependencias externas)              │
└───────────────────────────▲──────────────────────────────┘
                            │ produce/consume entidades del dominio
┌───────────────────────────┴──────────────────────────────┐
│  Infrastructure  (infrastructure/)                        │  ← detalles externos
│  RabbitMQ (publisher), MONAI (bundles/inferencia),        │
│  almacenamiento de imágenes, logging, configuración       │
└──────────────────────────────────────────────────────────┘
```

- **Presentation** conoce Application; no sabe de MONAI ni de torch.
- **Application** orquesta usando componentes de Infrastructure que le **inyecta** el
  composition root (`main.py`); construye las entidades del **Domain**.
- **Domain** es el centro puro: no importa `pika`, `monai` ni `pydantic-settings`.
- **Infrastructure** implementa los detalles técnicos y traduce hacia/desde el Domain.

Regla práctica: **una capa nunca importa una capa superior**. Presentation → Application →
Domain; Infrastructure → Domain.

---

## 3. Árbol de directorios

```
microservice/
├── AGENTS.md                     # Convenciones y comandos para agentes
├── pyproject.toml                # Metadatos + dependencias (gestionado por uv)
├── uv.lock                       # Lockfile reproducible
├── .python-version               # 3.12
├── .env.example                  # Plantilla de variables de entorno (sin secretos)
├── Dockerfile                    # Imagen del servicio (base MONAI/PyTorch)
├── docker-compose.yml            # RabbitMQ local + servicio para dev/integración
├── docs/
│   ├── GLOBAL_ARCHITECTURE.md    # Arquitectura del sistema completo (fuente de verdad)
│   ├── PROJECT_ARCHITECTURE.md   # Este documento
│   ├── DESIGN.md                 # Decisiones de diseño y principios
│   ├── CHANGELOG.md              # Roadmap versionado
│   └── README.md                 # Guía del microservicio
├── src/
│   └── sinapsis_ai/              # Paquete importable (layout src/)
│       ├── __init__.py
│       ├── main.py               # Composition root: cablea las capas y arranca
│       ├── config.py             # Settings (pydantic-settings) desde variables de entorno
│       │
│       ├── presentation/         # CAPA 1 · Presentación (borde de entrada)
│       │   ├── __init__.py
│       │   ├── consumer.py       # Consumidor pika: prefetch=1, valida, delega, ack/nack
│       │   └── schemas.py        # DTOs pydantic de mensajes (request/result) + (de)serialización
│       │
│       ├── application/          # CAPA 2 · Aplicación (casos de uso / orquestación)
│       │   ├── __init__.py
│       │   └── analysis_service.py # Orquesta: resuelve modelo → imagen → inferencia → resultado
│       │
│       ├── domain/              # CAPA 3 · Dominio (entidades + reglas, sin deps externas)
│       │   ├── __init__.py
│       │   ├── models.py         # AnalysisRequest, AnalysisResult, AnalysisType, ModelRef, Artifact
│       │   └── errors.py         # Jerarquía de excepciones del dominio
│       │
│       └── infrastructure/      # CAPA 4 · Infraestructura (detalles externos)
│           ├── __init__.py
│           ├── publisher.py      # Publica resultados a RabbitMQ (pika)
│           ├── bundle_registry.py# Descarga/cachea/resuelve bundles MONAI (Model Zoo)
│           ├── inference_engine.py # Ejecuta la inferencia del bundle (BundleWorkflow/ConfigParser)
│           ├── image_store.py     # Obtiene la imagen por URI y guarda artefactos de salida
│           └── logging.py         # Configuración de logging estructurado (sin PII)
└── tests/
    ├── __init__.py
    ├── conftest.py               # Fixtures compartidas (settings de prueba, mensajes, mocks)
    ├── fixtures/
    │   ├── sample_request.json   # Mensaje de solicitud de ejemplo
    │   └── tiny_volume.nii.gz     # Volumen mínimo para smoke test de inferencia
    ├── unit/                     # Espeja las capas de src/ (sin red, sin RabbitMQ real)
    │   ├── test_config.py
    │   ├── presentation/
    │   │   ├── test_consumer.py
    │   │   └── test_schemas.py
    │   ├── application/
    │   │   └── test_analysis_service.py
    │   ├── domain/
    │   │   ├── test_models.py
    │   │   └── test_errors.py
    │   └── infrastructure/
    │       ├── test_publisher.py
    │       ├── test_bundle_registry.py
    │       ├── test_inference_engine.py
    │       └── test_image_store.py
    └── integration/              # @pytest.mark.integration (omitidos por defecto)
        ├── test_rabbitmq_flow.py # Round-trip request→result contra RabbitMQ real
        └── test_inference_smoke.py# Descarga un bundle pequeño y ejecuta inferencia real
```

---

## 4. Módulos y responsabilidades (por capa)

### Raíz
| Módulo / archivo | Responsabilidad |
|------------------|-----------------|
| `main.py` | **Composition root**. Carga `Settings`, inicializa logging, construye los componentes de infraestructura, los inyecta en `AnalysisService`, monta el `Consumer` y arranca. Maneja apagado limpio (SIGTERM/SIGINT). |
| `config.py` | Define `Settings` (pydantic-settings). Única fuente de configuración; lee variables de entorno. Sin valores sensibles por defecto. |

### Capa Presentation (`presentation/`)
| Módulo / archivo | Responsabilidad |
|------------------|-----------------|
| `presentation/consumer.py` | Consumidor pika con `prefetch=1`. Recibe el mensaje, lo valida vía `schemas`, invoca `AnalysisService`, y hace `ack`/`nack` (a dead-letter en fallo no recuperable). No contiene lógica de inferencia. |
| `presentation/schemas.py` | DTOs pydantic de mensajes de entrada/salida y su validación. Traduce entre JSON del broker y entidades de `domain/`. |

### Capa Application (`application/`)
| Módulo / archivo | Responsabilidad |
|------------------|-----------------|
| `application/analysis_service.py` | **Caso de uso** central. Orquesta el flujo: resolver bundle (`bundle_registry`), obtener imagen (`image_store`), ejecutar inferencia (`inference_engine`), **construir** el `AnalysisResult` (métricas/artefactos) y publicarlo (`publisher`). No conoce `pika` ni AMQP; recibe sus dependencias inyectadas. |

### Capa Domain (`domain/`)
| Módulo / archivo | Responsabilidad |
|------------------|-----------------|
| `domain/models.py` | Entidades: `AnalysisRequest`, `AnalysisResult`, `AnalysisType` (enum), `ModelRef`, `Artifact`. Puras, independientes del transporte y de la infraestructura. |
| `domain/errors.py` | Jerarquía de excepciones: `SinapsisAIError` (base), `InvalidMessageError`, `UnknownAnalysisTypeError`, `BundleResolutionError`, `InferenceError`, `ImageAccessError`. |

### Capa Infrastructure (`infrastructure/`)
| Módulo / archivo | Responsabilidad |
|------------------|-----------------|
| `infrastructure/publisher.py` | Publica `AnalysisResult` al exchange/routing key de resultados con `delivery_mode=persistent`. |
| `infrastructure/bundle_registry.py` | Mapea `AnalysisType` → bundle MONAI. Descarga y cachea el bundle en `BUNDLE_CACHE_DIR`; devuelve un `ModelRef` listo para ejecutar. |
| `infrastructure/inference_engine.py` | Ejecuta el pipeline del bundle (pre-proceso → red → inferer → post-proceso) sobre la imagen, en `MODEL_DEVICE`. Devuelve la salida cruda del modelo. |
| `infrastructure/image_store.py` | Resuelve el `image_uri` de la solicitud, descarga la imagen de entrada y persiste los artefactos de salida; devuelve URIs. |
| `infrastructure/logging.py` | Configura logging estructurado. Prohíbe loguear PII o píxeles; solo IDs opacos y metadatos. |

---

## 5. Convenciones de nomenclatura

| Elemento | Convención | Ejemplo |
|----------|------------|---------|
| Paquete importable | `snake_case` | `sinapsis_ai` |
| Distribución (pyproject) | `kebab-case` | `sinapsis-ai` |
| Capas (paquetes) | `snake_case` | `presentation`, `application`, `domain`, `infrastructure` |
| Módulos / archivos | `snake_case.py` | `bundle_registry.py` |
| Clases | `PascalCase` | `AnalysisRequest`, `AnalysisService` |
| Funciones / variables | `snake_case` | `run_analysis`, `image_uri` |
| Constantes | `UPPER_SNAKE_CASE` | `DEFAULT_PREFETCH` |
| Enums de dominio | `PascalCase` + miembros `UPPER_SNAKE` | `AnalysisType.CT_SPLEEN_SEGMENTATION` |
| Tests | espejo de la capa: `tests/unit/<capa>/test_<módulo>.py` | `tests/unit/application/test_analysis_service.py` |
| Funciones de test | `test_<unidad>_<comportamiento>` | `test_run_analysis_publishes_result` |

Código y símbolos en **inglés**; documentación (`docs/`) en **español**.

---

## 6. Patrones de diseño aplicados

| Ubicación | Patrón | Motivo |
|-----------|--------|--------|
| Todo el paquete | **Layered Architecture (arquitectura por capas)** | Separa presentación, aplicación, dominio e infraestructura; dependencias hacia abajo. |
| `application/analysis_service.py` | **Service / Use Case** | Orquesta el flujo de análisis en un único punto, independiente del transporte. |
| `infrastructure/bundle_registry.py` | **Registry + Lazy Loading / Cache** | Resuelve y cachea bundles bajo demanda; evita re-descargas de pesos. |
| `infrastructure/inference_engine.py` | **Strategy** | El bundle (config del modelo) es la estrategia intercambiable de inferencia. |
| `presentation/schemas.py` | **DTO + Translator** | Traduce JSON del broker ↔ entidades del dominio. |
| `config.py` | **Singleton de configuración** | Una instancia de `Settings` inyectada; sin globals dispersos. |
| `domain/errors.py` | **Exception Hierarchy** | Diferencia errores recuperables (nack→retry) de no recuperables (dead-letter). |
| `main.py` | **Composition Root** | Único lugar donde se instancian y conectan las capas. |

Detalle y ejemplos en [`DESIGN.md`](./DESIGN.md).

---

## 7. Dependencias externas

| Dependencia | Capa | Propósito |
|-------------|------|-----------|
| `monai` | Infrastructure | Framework de imagen médica: transforms, redes, inferers, formato **bundle**. |
| `torch` | Infrastructure | Backend de cómputo (CPU/GPU) que ejecuta el modelo. |
| `pika` | Presentation / Infrastructure | Cliente RabbitMQ (AMQP 0-9-1): consumir (presentation) y publicar (infrastructure). |
| `pydantic` | Presentation / Domain | Modelado y validación de DTOs y entidades. |
| `pydantic-settings` | Config | Carga tipada de configuración desde variables de entorno. |
| **Dev**: `pytest`, `pytest-cov` | — | Tests y cobertura. |
| **Dev**: `ruff` | — | Lint + formato. |
| **Dev**: `mypy` | — | Chequeo estático de tipos. |
| **Dev (opcional)**: `testcontainers` | — | RabbitMQ efímero para tests de integración. |

---

## 8. Configuración (variables de entorno)

Definidas en `config.py`; plantilla en `.env.example`. **Sin secretos en git.**

| Variable | Propósito | Default |
|----------|-----------|---------|
| `RABBITMQ_URL` | URL AMQP completa con credenciales | *(requerida)* |
| `RABBITMQ_REQUEST_QUEUE` | Cola de solicitudes de análisis | `ai.analysis.requests` |
| `RABBITMQ_RESULT_EXCHANGE` | Exchange donde se publican resultados | `sinapsis.ai` |
| `RABBITMQ_RESULT_ROUTING_KEY` | Routing key de resultados | `ai.analysis.result` |
| `RABBITMQ_PREFETCH` | Mensajes en vuelo por worker | `1` |
| `BUNDLE_CACHE_DIR` | Directorio de caché de bundles MONAI | `./.bundle_cache` |
| `BUNDLE_SOURCE` | Fuente de descarga (`monaihosting`/`huggingface`) | `monaihosting` |
| `ALLOWED_ANALYSIS_TYPES` | Tipos de análisis habilitados (→ bundles) | *(requerida)* |
| `MODEL_DEVICE` | Dispositivo de inferencia (`cuda`/`cpu`) | `cpu` |
| `IMAGE_STORAGE_BACKEND` | Backend de imágenes/artefactos (URI scheme) | *(requerida)* |
| `LOG_LEVEL` | Nivel de logging | `INFO` |

---

## 9. Interfaces expuestas y consumidas

Este servicio **no expone HTTP**. Su única interfaz es **asíncrona vía RabbitMQ**, en la
capa de presentación.

### Consume — `AnalysisRequest` (cola `RABBITMQ_REQUEST_QUEUE`)

```json
{
  "request_id": "uuid",
  "study_id": "opaque-id",
  "patient_ref": "opaque-id",        // identificador opaco, sin PII
  "analysis_type": "ct_spleen_segmentation",
  "image_uri": "s3://bucket/study/scan.nii.gz",
  "requested_by": "doctor-opaque-id", // ya autorizado por el backend
  "correlation_id": "uuid",
  "issued_at": "2026-01-01T12:00:00Z"
}
```

### Publica — `AnalysisResult` (exchange `RABBITMQ_RESULT_EXCHANGE`)

```json
{
  "request_id": "uuid",
  "study_id": "opaque-id",
  "status": "succeeded",              // o "failed"
  "model": { "name": "spleen_ct_segmentation", "version": "0.5.3" },
  "artifacts": [
    { "type": "segmentation_mask", "uri": "s3://bucket/study/mask.nii.gz" }
  ],
  "metrics": { "volume_ml": 210.4 },
  "error": null,                       // { "code": "...", "message": "..." } si status=failed
  "processed_at": "2026-01-01T12:00:42Z",
  "duration_ms": 42000
}
```

El contrato de mensajes es propiedad compartida con el backend (Go); cambios requieren
versionar el esquema.

---

## 10. Flujo de datos principal (a través de las capas)

```
RabbitMQ (request)
   │  AnalysisRequest (JSON)
   ▼
[Presentation]  consumer.py ──(valida)──► schemas.py ──► domain.AnalysisRequest
   │
   ▼
[Application]   analysis_service.py  (orquesta)
   │                 │
   │                 ├─► [Infra] bundle_registry.py ─(resuelve+cachea)─► ModelRef
   │                 ├─► [Infra] image_store.py ─(descarga imagen por URI)─► entrada local
   │                 ├─► [Infra] inference_engine.py ─(pre→red→infer→post)─► salida del modelo
   │                 ├─► construye domain.AnalysisResult (métricas/artefactos)
   │                 ├─► [Infra] image_store.py ─(guarda artefactos)─► URIs
   │                 └─► [Infra] publisher.py ─────────────► RabbitMQ (result)
   ▼
[Presentation]  consumer.py ──► ack   (nack→dead-letter si error no recuperable)
```

Detalle de decisiones (por qué capas, por qué prefetch=1, por qué stateless, manejo de
errores) en [`DESIGN.md`](./DESIGN.md).
