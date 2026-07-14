# Especificación de Funcionalidad: Capa Domain + Contrato de Mensajes v0.2.0

**Creado**: 2026-07-05

## Escenarios de Usuario y Pruebas *(obligatorio)*

### Historia de Usuario 1 - Entidades del dominio puras (Prioridad: P1)

Un desarrollador que trabaja en capas superiores (Application, Presentation) dispone de
entidades del dominio bien definidas y validadas (`AnalysisType`, `AnalysisRequest`,
`AnalysisResult`, `ModelRef`, `Artifact`) que puede construir, inspeccionar y pasar entre
capas sin importar ningún detalle externo (ni pika, ni monai, ni pydantic-settings).

**Por qué esta prioridad**: Es el núcleo del sistema. Sin modelos de dominio no hay
contrato interno ni externo. Toda capa superior depende de estas entidades para tipado,
validación de datos y reglas de negocio.

**Prueba Independiente**: `uv run pytest tests/unit/domain/test_models.py` pasa sin
RabbitMQ, sin MONAI y sin ninguna variable de entorno especial.

**Escenarios de Aceptación**:

1. **Escenario**: Construcción válida de AnalysisRequest
   - **Dado** un conjunto de campos válidos (request_id UUID, study_id, patient_ref,
     analysis_type conocido, image_uri, requested_by, correlation_id, issued_at)
   - **Cuando** se construye un `AnalysisRequest`
   - **Entonces** el objeto existe sin error y sus atributos son accesibles con los
     tipos correctos

2. **Escenario**: AnalysisType cubre los tipos conocidos
   - **Dado** el enum `AnalysisType`
   - **Cuando** se accede al miembro `CT_SPLEEN_SEGMENTATION`
   - **Entonces** su valor de cadena es `"ct_spleen_segmentation"` (snake_case, para
     mapeo directo con el JSON del broker y el nombre del bundle MONAI)

3. **Escenario**: AnalysisResult con status succeeded
   - **Dado** un `ModelRef` con name y version, una lista de `Artifact`, métricas dict y
     `processed_at` datetime
   - **Cuando** se construye `AnalysisResult(status="succeeded", ...)`
   - **Entonces** `result.status == "succeeded"`, `result.error is None` y los artefactos
     y métricas son accesibles

4. **Escenario**: AnalysisResult con status failed
   - **Dado** un error dict con `code` y `message`
   - **Cuando** se construye `AnalysisResult(status="failed", error={...})`
   - **Entonces** `result.error` contiene el bloque de error y los artefactos pueden ser
     lista vacía

---

### Historia de Usuario 2 - Jerarquía de errores del dominio (Prioridad: P2)

Un desarrollador puede distinguir entre errores recuperables y no recuperables del
microservicio consultando la jerarquía de excepciones del dominio, sin conocer detalles de
RabbitMQ ni MONAI.

**Por qué esta prioridad**: La jerarquía de errores es lo que permite a la capa
Presentation implementar la política de ack/nack/dead-letter correctamente. Sin ella la
lógica de recuperación de fallos no puede existir. Es fundacional para v0.5.0.

**Prueba Independiente**: `uv run pytest tests/unit/domain/test_errors.py` pasa sin ningún
mock ni infraestructura.

**Escenarios de Aceptación**:

1. **Escenario**: Todas las excepciones heredan de SinapsisAIError
   - **Dado** las clases `InvalidMessageError`, `UnknownAnalysisTypeError`,
     `BundleResolutionError`, `ImageAccessError`, `InferenceError`
   - **Cuando** se verifica `isinstance(exc, SinapsisAIError)` para cada una
   - **Entonces** todas retornan `True`

2. **Escenario**: Captura selectiva por tipo
   - **Dado** un bloque `except InvalidMessageError`
   - **Cuando** se lanza `InvalidMessageError("bad payload")`
   - **Entonces** es capturada por `InvalidMessageError` y también por `SinapsisAIError`,
     pero NO por `UnknownAnalysisTypeError`

3. **Escenario**: Mensaje de error preservado
   - **Dado** `InferenceError("model exploded")`
   - **Cuando** se accede a `str(exc)`
   - **Entonces** contiene el mensaje original

---

### Historia de Usuario 3 - DTOs de mensajes y traducción al dominio (Prioridad: P3)

La capa Presentation puede validar un JSON crudo de RabbitMQ y obtener un
`AnalysisRequest` de dominio, y puede serializar un `AnalysisResult` de dominio a JSON
siguiendo el contrato del §9 de `PROJECT_ARCHITECTURE.md`, sin conocer pika ni MONAI.

**Por qué esta prioridad**: Es el contrato externo compartido con el backend Go. Aunque
indispensable para v0.5.0, las entidades de dominio (HU1) son la base más urgente. Los
DTOs dependen de que los modelos de dominio existan.

**Prueba Independiente**: `uv run pytest tests/unit/presentation/test_schemas.py` pasa con
un `sample_request.json` de fixture, sin RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: JSON válido → AnalysisRequest
   - **Dado** el JSON de `tests/fixtures/sample_request.json` con todos los campos del
     contrato (§9 PROJECT_ARCHITECTURE)
   - **Cuando** se parsea con el DTO de request de `presentation/schemas.py`
   - **Entonces** se obtiene un `AnalysisRequest` de dominio con todos los campos
     correctamente tipados

2. **Escenario**: JSON sin image_uri → InvalidMessageError
   - **Dado** un JSON de request sin el campo `image_uri`
   - **Cuando** se intenta parsear con el DTO de request
   - **Entonces** se lanza (o envuelve en) `InvalidMessageError`

3. **Escenario**: JSON con analysis_type desconocido → UnknownAnalysisTypeError
   - **Dado** un JSON con `"analysis_type": "unknown_type_xyz"`
   - **Cuando** se intenta parsear con el DTO de request
   - **Entonces** se lanza (o envuelve en) `UnknownAnalysisTypeError`

4. **Escenario**: AnalysisResult succeeded → JSON del contrato
   - **Dado** un `AnalysisResult` con `status="succeeded"`, `model`, `artifacts`,
     `metrics`, `processed_at` y `duration_ms`
   - **Cuando** se serializa con el DTO de result
   - **Entonces** el JSON resultante contiene todos los campos del contrato (§9) con
     `"error": null`

5. **Escenario**: AnalysisResult failed → JSON con bloque error
   - **Dado** un `AnalysisResult` con `status="failed"` y `error={"code": "...", "message": "..."}`
   - **Cuando** se serializa con el DTO de result
   - **Entonces** el JSON contiene el bloque `error` completo

---

### Casos Límite

- `AnalysisType` con un valor de cadena que no está en el enum → `UnknownAnalysisTypeError`
  (no silencio, no None).
- `AnalysisRequest` con `image_uri` vacío (`""`) → error de validación.
- `AnalysisRequest` con `issued_at` en formato inválido (no ISO 8601) → error de validación.
- `AnalysisResult` con `status` distinto de `"succeeded"` / `"failed"` → error de
  validación (enum estricto).
- `ModelRef` con `version` vacío → admisible (algunos bundles no tienen versión semántica).
- JSON con campos extra desconocidos en el request → ignorados (schema permisivo en extras)
  para compatibilidad forward con el backend Go.
- `metrics` en `AnalysisResult` puede ser un dict vacío `{}` (análisis sin métricas
  calculadas).
- `artifacts` en `AnalysisResult` puede ser lista vacía `[]` (fallo antes de producir
  artefactos).

---

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El sistema DEBE definir `AnalysisType` como enum con al menos el miembro
  `CT_SPLEEN_SEGMENTATION = "ct_spleen_segmentation"`.
- **RF-002**: El sistema DEBE definir `AnalysisRequest` con los campos: `request_id` (str/UUID),
  `study_id` (str), `patient_ref` (str, opaco), `analysis_type` (`AnalysisType`),
  `image_uri` (str, no vacío), `requested_by` (str), `correlation_id` (str/UUID),
  `issued_at` (datetime).
- **RF-003**: El sistema DEBE definir `ModelRef` con `name` (str) y `version` (str).
- **RF-004**: El sistema DEBE definir `Artifact` con `type` (str) y `uri` (str).
- **RF-005**: El sistema DEBE definir `AnalysisResult` con `request_id`, `study_id`,
  `status` (enum: `"succeeded"` | `"failed"`), `model` (`ModelRef`), `artifacts`
  (`list[Artifact]`), `metrics` (`dict[str, float]`), `error` (dict opcional con `code`
  y `message` o `None`), `processed_at` (datetime) y `duration_ms` (int).
- **RF-006**: Las entidades de dominio NO DEBEN importar `pika`, `monai`, `pydantic-settings`
  ni ninguna dependencia externa al paquete estándar de Python y a `pydantic` (si se
  usa para validación interna).
- **RF-007**: El sistema DEBE definir `SinapsisAIError` como excepción base y las
  subclases: `InvalidMessageError`, `UnknownAnalysisTypeError`, `BundleResolutionError`,
  `ImageAccessError`, `InferenceError` en `domain/errors.py`.
- **RF-008**: El sistema DEBE definir en `presentation/schemas.py` un DTO de request que
  valide el JSON entrante y lo traduzca a `AnalysisRequest`; un campo inválido o faltante
  DEBE resultar en `InvalidMessageError`.
- **RF-009**: El sistema DEBE definir en `presentation/schemas.py` un DTO de result que
  serialice `AnalysisResult` a JSON siguiendo exactamente el contrato del §9 de
  `PROJECT_ARCHITECTURE.md`.
- **RF-010**: Un `analysis_type` desconocido en el JSON del request DEBE resultar en
  `UnknownAnalysisTypeError`, no en un error genérico de pydantic.
- **RF-011**: El archivo `tests/fixtures/sample_request.json` DEBE contener un mensaje de
  solicitud válido completo usable como fixture en tests.

### Entidades Clave

- **`AnalysisType`**: Enum de tipos de análisis soportados. Cada miembro mapea
  `UPPER_SNAKE` → `"snake_case"` (valor = nombre del bundle MONAI).
- **`AnalysisRequest`**: Solicitud de análisis validada. Sin PII: `patient_ref` es ID
  opaco. `image_uri` referencia la imagen por URI (no incrusta píxeles).
- **`ModelRef`**: Bundle MONAI resuelto. Identifica el modelo que ejecutó la inferencia.
- **`Artifact`**: Salida persistida del análisis (ej. máscara de segmentación). Solo URI,
  no bytes.
- **`AnalysisResult`**: Resultado completo del análisis. Contiene estado, modelo,
  artefactos, métricas y error opcional. Es el contrato de salida compartido con el backend.
- **`SinapsisAIError` y subclases**: Jerarquía que permite a la Presentation distinguir
  errores de dead-letter (no recuperables) de errores transitorios.

---

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: `uv run pytest tests/unit/domain/ tests/unit/presentation/test_schemas.py -v`
  → todos los tests pasan (mínimo 6 tests: 4 domain/models, 1 domain/errors, al menos 5
  presentation/schemas).
- **CE-002**: `uv run mypy src` → código 0, sin errores en `domain/models.py`,
  `domain/errors.py` y `presentation/schemas.py`.
- **CE-003**: `uv run ruff format --check . && uv run ruff check .` → código 0.
- **CE-004**: Las entidades de dominio son importables sin ninguna variable de entorno
  definida: `uv run python -c "from sinapsis_ai.domain.models import AnalysisType; print(AnalysisType.CT_SPLEEN_SEGMENTATION)"`.
- **CE-005**: El JSON de `tests/fixtures/sample_request.json` es parseado correctamente
  por el DTO de request y produce un `AnalysisRequest` válido.
- **CE-006**: La jerarquía de errores permite captura con `except SinapsisAIError` para
  todos los tipos de error del dominio.
