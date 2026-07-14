# Diseño — SINAPSIS AI Microservice

> Decisiones de diseño, principios y estrategia de testing del microservicio de
> inferencia MONAI. Complementa [`PROJECT_ARCHITECTURE.md`](./PROJECT_ARCHITECTURE.md)
> (qué existe) explicando **por qué** existe así.

---

## 1. Filosofía de diseño

**Qué ES este servicio:**

- Un **ejecutor de modelos pre-entrenados** MONAI, aislado y reemplazable.
- Un **consumidor de cola** stateless: entra una solicitud, sale un resultado.
- Un **componente de soporte a la decisión**: produce evidencia (segmentaciones,
  métricas), nunca un veredicto clínico.

**Qué NO ES:**

- No entrena ni afina modelos.
- No aplica reglas clínicas (pre-diagnóstico, rol del doctor, auditoría): eso vive en el
  backend Go (ver [`GLOBAL_ARCHITECTURE.md`](./GLOBAL_ARCHITECTURE.md) §4).
- No expone API HTTP ni mantiene estado entre mensajes.
- No conoce PII: opera con identificadores opacos y referencias de imagen por URI.

**Principio rector:** *la inferencia es lenta y pesada; debe poder fallar, reintentarse y
escalarse sin poner en riesgo el sistema clínico.* De ahí el desacople por RabbitMQ y el
diseño stateless.

---

## 2. Arquitectura por capas (simple)

Se elige una **arquitectura por capas simple** por encima de estilos más elaborados: el
servicio es pequeño y su flujo es lineal (recibir → orquestar → ejecutar → publicar), así
que cuatro capas con dependencias hacia abajo son suficientes y fáciles de razonar.

```
Presentation  →  Application  →  Domain
                                   ▲
Infrastructure  ───────────────────┘
```

| Capa | Paquete | Rol | Conoce a | NO conoce |
|------|---------|-----|----------|-----------|
| Presentation | `presentation/` | Borde de entrada: consumidor RabbitMQ + DTOs | Application, Domain | MONAI, torch |
| Application | `application/` | Orquestación del caso de uso | Domain, (Infra inyectada) | AMQP, `pika` |
| Domain | `domain/` | Entidades y reglas puras | — (nada externo) | Todo lo externo |
| Infrastructure | `infrastructure/` | Detalles técnicos (RabbitMQ, MONAI, storage, logging) | Domain | Application, Presentation |

**Regla de dependencia:** una capa **nunca** importa una capa superior. Presentation →
Application → Domain; Infrastructure → Domain. El `main.py` (composition root) es el único
lugar que conoce todas las capas y las cablea.

---

## 3. Principios de diseño aplicados

### Separación por capas (dependencias hacia abajo)
El flujo cruza las capas en un solo sentido. Presentation traduce el mensaje y delega;
Application orquesta; Domain modela; Infrastructure ejecuta lo técnico. Esto mantiene el
núcleo (`domain/`, `application/`) libre de detalles de RabbitMQ y MONAI.

### Single Responsibility (SRP)
Cada módulo tiene una razón de cambio: `bundle_registry.py` cambia si cambia la resolución/
caché de modelos; `inference_engine.py` si cambia la ejecución; `presentation/` si cambia
el transporte. El caso de uso (`analysis_service.py`) no se toca al cambiar de broker.

### Dependency Inversion (DIP)
`application/` **no importa `pika` ni `monai`**: recibe sus colaboradores de infraestructura
inyectados por el composition root. Esto permite testear el caso de uso con dobles y
sustituir infraestructura sin tocar el dominio ni la orquestación.

### KISS + prefetch=1
Se elige un consumidor **síncrono** (`pika`) con `prefetch=1` en lugar de concurrencia
interna compleja: una inferencia pesada por proceso, y se escala **horizontalmente** con
más réplicas. Menos superficie de error que un pool de inferencias concurrentes peleando
por la GPU.

### Fail-safe defaults
La configuración no trae credenciales por defecto; los tipos de análisis permitidos son
explícitos (`ALLOWED_ANALYSIS_TYPES`). Un `analysis_type` desconocido se rechaza a
dead-letter, no se ejecuta “a ciegas”.

### DRY en contratos
El contrato de mensajes vive en un solo lugar (`presentation/schemas.py`) y se traduce a
entidades de `domain/`; ni el consumidor ni el servicio de aplicación reimplementan
validación.

---

## 4. Patrones de diseño (con referencia a archivos)

| Patrón | Dónde | Detalle |
|--------|-------|---------|
| **Layered Architecture** | todo el paquete | Cuatro capas con dependencias hacia abajo (§2). |
| **Service / Use Case** | `application/analysis_service.py` | Un único orquestador del flujo de análisis, independiente del transporte. |
| **Registry + Cache** | `infrastructure/bundle_registry.py` | `AnalysisType → bundle`. Descarga una vez, cachea en `BUNDLE_CACHE_DIR`, reutiliza. |
| **Strategy** | `infrastructure/inference_engine.py` | El bundle MONAI (config + pesos) es la estrategia de inferencia intercambiable. |
| **DTO + Translator** | `presentation/schemas.py` | Traduce entre JSON del broker y entidades del dominio. |
| **Composition Root** | `main.py` | Único punto de construcción y cableado de las capas. |
| **Exception Hierarchy** | `domain/errors.py` | Clasifica errores para decidir ack/nack/dead-letter. |

---

## 5. Modelos de dominio

Definidos en `domain/models.py` (capa Domain, independientes del transporte y la infra):

- **`AnalysisType`** *(enum)* — catálogo de análisis soportados; cada miembro mapea a un
  bundle MONAI concreto (p. ej. `CT_SPLEEN_SEGMENTATION → "spleen_ct_segmentation"`).
- **`AnalysisRequest`** — solicitud validada: `request_id`, `study_id`, `patient_ref`
  (opaco), `analysis_type`, `image_uri`, `requested_by`, `correlation_id`, `issued_at`.
- **`ModelRef`** — bundle resuelto: `name`, `version`, ruta local en caché.
- **`Artifact`** — salida persistida: `type` (p. ej. `segmentation_mask`), `uri`.
- **`AnalysisResult`** — resultado: `status`, `model` (`ModelRef`), `artifacts`, `metrics`,
  `error?`, `processed_at`, `duration_ms`.

Regla: las entidades del dominio **no** contienen PII ni bytes de imagen; solo IDs opacos,
métricas y URIs.

---

## 6. Estrategia de testing

### Cobertura y comandos
- Framework: **pytest** + **pytest-cov**. Umbral objetivo: **≥ 85%** en `src/sinapsis_ai`.
- Comando de cobertura:
  `uv run pytest --cov=sinapsis_ai --cov-report=term-missing`.
- Orden de verificación (de [`AGENTS.md`](../AGENTS.md)): `ruff format --check` →
  `ruff check` → `mypy src` → `pytest`.

### Estructura (espejo de capas)
- `tests/unit/` **espeja las capas** de `src/`: `tests/unit/<capa>/test_<módulo>.py`
  (p. ej. `tests/unit/application/test_analysis_service.py`). Sin red, sin RabbitMQ real,
  sin descargas: todo mockeado.
- `tests/integration/` marcados `@pytest.mark.integration`, **omitidos por defecto**
  (config en `pyproject.toml`). Requieren RabbitMQ real (`docker compose up -d rabbitmq`)
  y/o descarga de un bundle pequeño.

### Cómo se testea cada capa
- **Domain**: tests puros de entidades y jerarquía de errores; sin mocks.
- **Application** (`test_analysis_service.py`): se inyectan **dobles** de infraestructura
  (registry, image_store, engine, publisher) y se verifica la orquestación y la construcción
  del `AnalysisResult`, sin tocar MONAI ni RabbitMQ.
- **Presentation** (`test_consumer.py`, `test_schemas.py`): canal `pika` mockeado; se
  verifica validación, delegación al service y `ack`/`nack`.
- **Infrastructure**: descargas MONAI y I/O de imágenes mockeadas en unit; ejecución real
  solo en integración.

### Mocks y datos de prueba
- **RabbitMQ**: en unit tests se mockea el canal `pika`; no se abre conexión real.
- **MONAI/torch**: en unit tests, `bundle_registry` e `inference_engine` se mockean para no
  descargar pesos ni ejecutar la red.
- **Fixtures**: `tests/fixtures/sample_request.json` (mensaje válido) y `tiny_volume.nii.gz`
  (volumen mínimo) para el smoke test de integración.

### Aislamiento
- Cada test construye su propia `Settings` de prueba (sin leer `.env` real).
- Sin estado global compartido; los dobles se inyectan vía el composition root o
  directamente en el SUT.
- Los tests de integración limpian colas/exchanges que crean.

### Escenarios clave a cubrir
- Mensaje inválido / `analysis_type` desconocido → `InvalidMessageError` /
  `UnknownAnalysisTypeError` → dead-letter (no crash del worker).
- Fallo al descargar imagen (`ImageAccessError`) → resultado `failed` o nack según política.
- Inferencia lanza excepción (`InferenceError`) → se publica resultado `failed` con `error`.
- Round-trip feliz: request → result publicado con artefactos y métricas.

---

## 7. Manejo de errores

Jerarquía en `domain/errors.py` (capa Domain):

```
SinapsisAIError                (base)
├── InvalidMessageError        # payload no cumple el esquema → dead-letter
├── UnknownAnalysisTypeError   # analysis_type no permitido → dead-letter
├── BundleResolutionError      # no se pudo descargar/cargar el bundle
├── ImageAccessError           # no se pudo leer/escribir la imagen por URI
└── InferenceError             # falló la ejecución del modelo
```

Política de acuse (ack) en `presentation/consumer.py`:

- **No recuperable** (mensaje mal formado, tipo desconocido): `nack` sin requeue →
  **dead-letter**. Reintentar no ayuda.
- **Potencialmente transitorio** (broker/almacenamiento intermitente): política de reintento
  acotada; tras agotarla, dead-letter.
- **Fallo de inferencia sobre un mensaje válido**: el `AnalysisService` construye un
  `AnalysisResult` con `status=failed` y `error`, el `publisher` lo **publica**, y el
  consumer hace `ack` (el backend decide qué hacer). No se pierde trazabilidad.

Objetivo: **un mensaje malo nunca detiene al worker** ni se procesa en bucle infinito.

---

## 8. Logging y observabilidad

- Logging **estructurado** configurado en `infrastructure/logging.py`, nivel por `LOG_LEVEL`.
- Se registran `request_id`, `correlation_id`, `analysis_type`, modelo, duración y estado.
- **Prohibido** loguear PII, contenido del paciente o píxeles de imagen (regla del sistema,
  ver [`GLOBAL_ARCHITECTURE.md`](./GLOBAL_ARCHITECTURE.md)). Solo identificadores opacos.
- `correlation_id` se propaga desde la solicitud al resultado para rastreo end-to-end con
  el backend.

---

## 9. Configuración y secretos

- Toda la configuración proviene de **variables de entorno** vía `pydantic-settings`
  (`config.py`). Plantilla documentada en `.env.example`.
- **Nunca** se hardcodean credenciales ni URLs de RabbitMQ; `.env` no se commitea.
- `Settings` se valida al arranque (**fail-fast**): si falta `RABBITMQ_URL` o
  `ALLOWED_ANALYSIS_TYPES`, el proceso no arranca.

---

## 10. Seguridad

- **Sin PII**: el mensaje referencia la imagen por URI y usa IDs opacos; el servicio no
  recibe ni almacena datos identificables.
- **Superficie mínima**: sin API HTTP entrante; solo AMQP autenticado por `RABBITMQ_URL`.
- **Confianza en el borde clínico**: la autorización (rol del doctor) y el pre-diagnóstico
  ya fueron validados por el backend antes de encolar; este servicio no re-implementa
  autorización clínica, pero **sí** rechaza tipos de análisis no permitidos.
- **Licencias de modelos**: cada bundle del Model Zoo tiene su licencia; se respeta la
  incluida en el bundle. MONAI no garantiza idoneidad diagnóstica → salida es soporte.

---

## 11. Extensibilidad

**Agregar un nuevo tipo de análisis:**
1. Añadir el miembro a `AnalysisType` en `domain/models.py` (Domain).
2. Registrar el mapeo `AnalysisType → bundle` en `infrastructure/bundle_registry.py` (Infra).
3. Si la salida es nueva (p. ej. clasificación en vez de segmentación), extender la
   construcción del resultado en `application/analysis_service.py` (Application).
4. Habilitarlo en `ALLOWED_ANALYSIS_TYPES`.
5. Añadir unit tests espejo por capa y, si aplica, un smoke test de integración.

**Cambiar de broker o de almacenamiento:** sustituir el módulo correspondiente en
`presentation/` (consumo) o `infrastructure/` (publicación/almacenamiento); Application y
Domain no cambian.

**Escalar:** desplegar más réplicas del consumidor (cada una `prefetch=1`); RabbitMQ
reparte la carga. No subir `prefetch` para “paralelizar” inferencias en un mismo proceso.
