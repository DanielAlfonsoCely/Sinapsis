# Plan de Implementación: Capa Domain + Contrato de Mensajes v0.2.0

**Fecha**: 2026-07-05
**Especificación**: [spec.md](./spec.md)

## Resumen

Implementar la capa Domain completa (`domain/models.py`, `domain/errors.py`) y el contrato
de mensajes Presentation (`presentation/schemas.py`), junto con sus tests unitarios y el
fixture `sample_request.json`. Sin inferencia, sin RabbitMQ real. Al finalizar,
`uv run pytest tests/unit/domain/ tests/unit/presentation/test_schemas.py` pasa en verde y
`mypy src` no reporta errores.

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12
**Dependencias Principales**: `pydantic>=2.7` (ya instalado en v0.1.0); sin nuevas deps de producción
**Almacenamiento**: N/A (no hay persistencia en v0.2.0)
**Testing**: `pytest`; espejo de capas `tests/unit/<capa>/test_<módulo>.py`; sin mocks (domain puro)
**Plataforma Objetivo**: servidor Linux (contenedor)
**Objetivos de Rendimiento**: N/A (modelos de datos, no inferencia)
**Restricciones**: `domain/` NO importa `pika`, `monai`, `pydantic-settings`; solo stdlib + `pydantic` opcional
**Escala/Alcance**: 3 módulos nuevos + 3 archivos de tests + 1 fixture JSON

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v0.2.0-domain-contracts/
├── plan.md   # Este archivo
└── spec.md
```

### Código Fuente

```text
src/sinapsis_ai/
├── domain/
│   ├── __init__.py          # ya existe (v0.1.0)
│   ├── models.py            # NUEVO: AnalysisType, AnalysisRequest, ModelRef, Artifact, AnalysisResult
│   └── errors.py            # NUEVO: SinapsisAIError + 5 subclases
└── presentation/
    ├── __init__.py          # ya existe (v0.1.0)
    └── schemas.py           # NUEVO: AnalysisRequestDTO, AnalysisResultDTO + traducción

tests/
├── fixtures/
│   └── sample_request.json  # NUEVO: mensaje de solicitud válido completo
└── unit/
    ├── domain/
    │   ├── __init__.py      # NUEVO
    │   ├── test_models.py   # NUEVO: tests de entidades
    │   └── test_errors.py   # NUEVO: tests de jerarquía de errores
    └── presentation/
        ├── __init__.py      # NUEVO
        └── test_schemas.py  # NUEVO: tests de DTOs y traducción
```

**Decisión de Estructura**: Espejo de capas estricto según DESIGN.md §6. Los tests de
`domain/` no usan mocks (entidades puras). Los tests de `presentation/schemas.py` usan la
fixture JSON y construyen entidades de dominio directamente.

---

## Fase 1: Infraestructura de tests (prerequisito)

**Propósito**: Crear los directorios y `__init__.py` de tests necesarios, y la fixture
JSON, antes de escribir cualquier módulo o test.

- [x] T001 [P] Crear `tests/fixtures/` y `tests/fixtures/sample_request.json` con un
  mensaje de solicitud válido completo (todos los campos del contrato §9 de
  PROJECT_ARCHITECTURE, `analysis_type: "ct_spleen_segmentation"`).
- [x] T002 [P] Crear `tests/unit/domain/__init__.py` (vacío).
- [x] T003 [P] Crear `tests/unit/presentation/__init__.py` (vacío).

**Punto de Control**: estructura de directorios lista; `uv run pytest --collect-only`
no reporta errores de importación.

---

## Fase 2: Fundacional — Capa Domain

**Propósito**: Implementar los módulos de dominio puros que todas las demás capas usan.

**CRITICO**: `presentation/schemas.py` (HU3) no puede existir sin los modelos y errores
del dominio.

- [x] T004 [P] Crear `src/sinapsis_ai/domain/errors.py` con:
  - `SinapsisAIError(Exception)` — base
  - `InvalidMessageError(SinapsisAIError)` — payload no cumple el esquema
  - `UnknownAnalysisTypeError(SinapsisAIError)` — analysis_type no permitido
  - `BundleResolutionError(SinapsisAIError)` — no se pudo descargar/cargar el bundle
  - `ImageAccessError(SinapsisAIError)` — no se pudo leer/escribir la imagen
  - `InferenceError(SinapsisAIError)` — falló la ejecución del modelo
- [x] T005 [P] Crear `src/sinapsis_ai/domain/models.py` con:
  - `AnalysisType(str, Enum)` — `CT_SPLEEN_SEGMENTATION = "ct_spleen_segmentation"`
  - `AnalysisRequest` — dataclass o pydantic model con todos los campos de RF-002
  - `ModelRef` — dataclass o pydantic model: `name: str`, `version: str`
  - `Artifact` — dataclass o pydantic model: `type: str`, `uri: str`
  - `AnalysisResult` — dataclass o pydantic model con todos los campos de RF-005
    incluyendo `status: Literal["succeeded", "failed"]` y `error: dict | None`

**Decisión de implementación**: usar `dataclasses` + validación mínima para mantener
`domain/` sin deps de pydantic (pydantic vive en Presentation). Si se necesita validación
en `AnalysisRequest` (ej. `image_uri` no vacío), usar `__post_init__` con raise explícito.

**Punto de Control**: `uv run python -c "from sinapsis_ai.domain.models import AnalysisType; print(AnalysisType.CT_SPLEEN_SEGMENTATION)"` imprime `ct_spleen_segmentation` sin error.

---

## Fase 3: Historia de Usuario 2 — Jerarquía de errores (Prioridad: P2)

*(implementada en Fase 2 como prerequisito; aquí se valida con tests)*

**Objetivo**: Tests unitarios que verifican la jerarquía de excepciones del dominio.

**Prueba Independiente**: `uv run pytest tests/unit/domain/test_errors.py -v`

### Pruebas para HU2

- [x] T006 [P] [HU2] Crear `tests/unit/domain/test_errors.py` con:
  - `test_errors_hierarchy` — verifica que todas las subclases son instancias de
    `SinapsisAIError` via `isinstance`.
  - `test_invalid_message_error_is_not_unknown_analysis_type` — captura selectiva: un
    `InvalidMessageError` no es `UnknownAnalysisTypeError`.
  - `test_error_message_preserved` — `str(InferenceError("msg"))` contiene `"msg"`.

**Punto de Control**: `uv run pytest tests/unit/domain/test_errors.py -v` → 3 tests passing.

---

## Fase 4: Historia de Usuario 1 — Entidades del dominio (Prioridad: P1)

**Objetivo**: Tests unitarios que verifican construcción, campos y validación de las
entidades de dominio.

**Prueba Independiente**: `uv run pytest tests/unit/domain/test_models.py -v`

### Pruebas para HU1

- [x] T007 [P] [HU1] Crear `tests/unit/domain/test_models.py` con:
  - `test_analysis_type_values` — `AnalysisType.CT_SPLEEN_SEGMENTATION == "ct_spleen_segmentation"`.
  - `test_analysis_request_construction` — construcción válida y acceso a campos.
  - `test_analysis_request_rejects_empty_image_uri` — `image_uri=""` lanza `ValueError`.
  - `test_analysis_result_succeeded` — `status="succeeded"`, `error is None`.
  - `test_analysis_result_failed` — `status="failed"`, bloque `error` presente.
  - `test_model_ref_and_artifact_construction` — construcción básica de `ModelRef` y `Artifact`.

**Punto de Control**: `uv run pytest tests/unit/domain/test_models.py -v` → 6 tests passing.

---

## Fase 5: Historia de Usuario 3 — DTOs y contrato de mensajes (Prioridad: P3)

**Objetivo**: `presentation/schemas.py` valida JSON de request → `AnalysisRequest` y
serializa `AnalysisResult` → JSON del contrato.

**Prueba Independiente**: `uv run pytest tests/unit/presentation/test_schemas.py -v`

### Pruebas para HU3

- [x] T008 [P] [HU3] Crear `tests/unit/presentation/test_schemas.py` con:
  - `test_request_schema_valid_payload` — `sample_request.json` → `AnalysisRequest` válido.
  - `test_request_schema_rejects_missing_image_uri` → `InvalidMessageError`.
  - `test_request_schema_rejects_unknown_analysis_type` → `UnknownAnalysisTypeError`.
  - `test_result_schema_serializes_success` — `AnalysisResult(status="succeeded")` → JSON
    con `"error": null` y todos los campos del contrato.
  - `test_result_schema_serializes_failure` — `AnalysisResult(status="failed", error={...})`
    → JSON con bloque `error`.

### Implementación para HU3

- [x] T009 [P] [HU3] Crear `src/sinapsis_ai/presentation/schemas.py` con:
  - `AnalysisRequestDTO(BaseModel)` — pydantic model que valida el JSON del broker:
    campos según el contrato de request de §9, validador que convierte `analysis_type`
    string a `AnalysisType` (lanzando `UnknownAnalysisTypeError` si no existe), validador
    que verifica `image_uri` no vacío (lanzando `InvalidMessageError` si falla). Método
    `.to_domain() -> AnalysisRequest`.
  - `AnalysisResultDTO` — función o clase que convierte `AnalysisResult` de dominio a
    `dict` serializable a JSON siguiendo exactamente el contrato de §9 (campos:
    `request_id`, `study_id`, `status`, `model`, `artifacts`, `metrics`, `error`,
    `processed_at`, `duration_ms`).
  - Función helper `parse_request(raw_json: str | bytes) -> AnalysisRequest` que envuelve
    errores de validación pydantic en `InvalidMessageError`.

**Punto de Control**: `uv run pytest tests/unit/presentation/test_schemas.py -v` → 5 tests passing.

---

## Fase 6: Verificación final

**Propósito**: Suite completa en verde, format/lint/types sin errores.

- [x] T010 Ejecutar `uv run ruff format --check . && uv run ruff check .` → código 0.
- [x] T011 Ejecutar `uv run mypy src` → código 0, sin errores en los 3 módulos nuevos.
- [x] T012 Ejecutar `uv run pytest` → 33 tests passing; integración omitida por defecto.
- [x] T013 Verificar importación sin variables de entorno:
  `uv run python -c "from sinapsis_ai.domain.models import AnalysisType; from sinapsis_ai.domain.errors import SinapsisAIError; print('OK')"`.

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Fase 1 (Infra de tests)**: Sin dependencias — crear primero.
- **Fase 2 (Fundacional — Domain)**: Depende de Fase 1. BLOQUEA Fases 3, 4 y 5.
- **Fase 3 (HU2 — Errores)**: Depende de Fase 2 (`errors.py` ya existe).
- **Fase 4 (HU1 — Modelos)**: Depende de Fase 2 (`models.py` ya existe).
- **Fase 5 (HU3 — Schemas)**: Depende de Fase 2 (necesita modelos y errores).
- **Fase 6 (Verificación)**: Depende de Fases 3, 4 y 5.

### Dependencias entre Historias de Usuario

- **HU1 (P1)**: Solo depende de Fase 2 fundacional — testeable con `pytest tests/unit/domain/test_models.py`.
- **HU2 (P2)**: Solo depende de Fase 2 fundacional — testeable con `pytest tests/unit/domain/test_errors.py`.
- **HU3 (P3)**: Depende de HU1 y HU2 (schemas usa modelos y errores de dominio) — testeable con `pytest tests/unit/presentation/test_schemas.py`.

### Dentro de Cada Fase

- Errores antes que modelos (los modelos pueden referenciar errores en validaciones).
- Modelos antes que schemas (schemas importa modelos y errores).
- Implementación antes que tests.
- Tests antes que verificación final.

## Notas

- [P] = tarea prioritaria / bloqueante dentro de la fase.
- [HU#] = mapeo a historia de usuario del spec.
- `domain/` usa `dataclasses` estándar de Python para mantener cero dependencias externas.
  Si se prefiere pydantic en el dominio para validación automática, es una decisión de
  diseño válida, pero requiere asegurar que `domain/` solo importe `pydantic` (no
  `pydantic-settings`).
- `presentation/schemas.py` SÍ puede usar pydantic `BaseModel` (es la capa de traducción).
- Los errores de validación de pydantic DEBEN ser envueltos en excepciones del dominio
  antes de propagarse fuera de `schemas.py` — nunca exponer `ValidationError` de pydantic
  a capas superiores.
- `issued_at` en `AnalysisRequest` se recibe como string ISO 8601 en el JSON y se convierte
  a `datetime` en el DTO.
- `sample_request.json` debe usar UUIDs reales (no placeholders) para que los tests sean
  deterministas.
- `AnalysisType` usa `StrEnum` (Python 3.11+) en lugar de `str, Enum` (ruff UP042).
- `datetime.now(datetime.UTC)` en lugar del deprecated `datetime.utcnow()` (ruff UP017).
- `UnknownAnalysisTypeError` se re-lanza directamente desde el validator de pydantic; los
  demás `ValidationError` se envuelven en `InvalidMessageError`.

## Ejecución Completada

**Fecha**: 2026-07-05
**Estado**: Todas las fases implementadas y verificadas.
