# Especificación de Funcionalidad: Motor de Inferencia + Caso de Uso de Análisis (v0.4.0)

**Creado**: 2026-07-05

## Escenarios de Usuario y Pruebas *(obligatorio)*

> El "usuario" directo de esta funcionalidad es el consumidor RabbitMQ (v0.5.0), que
> delegará cada solicitud validada al caso de uso. Esta versión entrega el corazón del
> servicio: el `AnalysisService` (Application) que orquesta el flujo completo, y los dos
> componentes de infraestructura que le faltaban — `image_store` (imágenes/artefactos por
> URI) e `inference_engine` (ejecución del bundle MONAI). Torch/MONAI se mockean en unit
> tests; la ejecución real se cubre con un smoke test de integración.

### Historia de Usuario 1 - Orquestar un análisis de extremo a extremo (Prioridad: P1)

Como sistema, cuando recibo una `AnalysisRequest` validada, necesito un caso de uso único
que orqueste el flujo completo — resolver el bundle, obtener la imagen, ejecutar la
inferencia, persistir los artefactos, construir el `AnalysisResult` (con métricas,
artefactos, duración y trazabilidad) y publicarlo — para que el backend reciba siempre un
resultado, incluso cuando la inferencia falla.

**Por qué esta prioridad**: Es el caso de uso central del microservicio (DESIGN.md §4,
patrón Service/Use Case). Con dobles inyectados es un MVP demostrable por sí solo: define
el contrato de colaboración de todas las piezas sin requerir MONAI ni RabbitMQ reales.

**Prueba Independiente**: Puede probarse completamente inyectando dobles de registry,
image store, engine y publisher, y verificando la secuencia de orquestación y el
`AnalysisResult` construido/publicado. No requiere las HU2/HU3 (implementaciones reales).

**Escenarios de Aceptación**:

1. **Escenario**: Flujo feliz produce y publica un resultado exitoso
   - **Dado** un `AnalysisService` con dobles inyectados que resuelven el bundle,
     entregan la imagen, ejecutan la inferencia y guardan artefactos
   - **Cuando** se ejecuta el análisis de una `AnalysisRequest` válida
   - **Entonces** se publica un `AnalysisResult` con `status="succeeded"`, el `ModelRef`
     resuelto, los artefactos con sus URIs, las métricas de la inferencia,
     `duration_ms > 0` y los `request_id`/`study_id` de la solicitud original

2. **Escenario**: Fallo de inferencia produce resultado fallido (no excepción)
   - **Dado** un `AnalysisService` cuyo motor de inferencia lanza `InferenceError`
   - **Cuando** se ejecuta el análisis
   - **Entonces** se publica un `AnalysisResult` con `status="failed"` y bloque `error`
     (código y mensaje), el caso de uso retorna normalmente (sin propagar la excepción) y
     la trazabilidad (`request_id`, modelo resuelto, duración) se preserva

3. **Escenario**: Errores previos a la inferencia se propagan al llamador
   - **Dado** un `AnalysisService` cuyo registry lanza `BundleResolutionError` (o cuyo
     image store lanza `ImageAccessError`)
   - **Cuando** se ejecuta el análisis
   - **Entonces** la excepción del dominio se propaga sin publicar resultado (la política
     de reintento/dead-letter es responsabilidad del consumidor, v0.5.0, DESIGN.md §7)

---

### Historia de Usuario 2 - Ejecutar el pipeline de inferencia del bundle (Prioridad: P2)

Como caso de uso de análisis, necesito un motor que ejecute el pipeline completo del
bundle MONAI resuelto (pre-proceso → red → inferer → post-proceso) sobre una imagen local,
en el dispositivo configurado (`MODEL_DEVICE`), y me devuelva la salida del modelo con sus
métricas, para transformar imágenes de entrada en evidencia clínica de soporte.

**Por qué esta prioridad**: Es la capacidad que da sentido al servicio (ejecutar modelos),
pero su contrato ya quedó definido por la HU1 — puede desarrollarse y testearse de forma
independiente con MONAI mockeado.

**Prueba Independiente**: Puede probarse invocando el motor con un `ModelRef` y una ruta
de imagen, con el workflow de MONAI mockeado, verificando que se configura el bundle con
los parámetros correctos (ruta local, dispositivo) y que la salida/las métricas se
devuelven; y que los fallos se traducen a `InferenceError`.

**Escenarios de Aceptación**:

1. **Escenario**: Ejecución del bundle devuelve la salida de inferencia
   - **Dado** un motor configurado con `MODEL_DEVICE` y un `ModelRef` con `local_path`
     válido
   - **Cuando** se ejecuta la inferencia sobre una imagen de entrada local
   - **Entonces** el pipeline del bundle se ejecuta con esa imagen y dispositivo, y se
     devuelve la salida del modelo (predicción persistible + métricas calculadas)

2. **Escenario**: Fallo del pipeline se traduce a InferenceError
   - **Dado** un motor cuyo workflow MONAI lanza una excepción durante la ejecución
   - **Cuando** se ejecuta la inferencia
   - **Entonces** se lanza `InferenceError` con la causa original encadenada (nunca se
     filtra una excepción cruda de torch/monai a las capas superiores)

---

### Historia de Usuario 3 - Obtener imágenes y persistir artefactos por URI (Prioridad: P3)

Como caso de uso de análisis, necesito un almacén que materialice la imagen referenciada
por `image_uri` en una ruta local de trabajo y que persista los artefactos de salida
(p. ej. máscara de segmentación) devolviendo sus URIs, para que el mensaje nunca
transporte píxeles y los resultados sean direccionables por el backend.

**Por qué esta prioridad**: Necesaria para la ejecución real de extremo a extremo, pero
sus contratos (entrada: URI → ruta local; salida: archivo local → URI) están definidos por
la HU1 y es la pieza más sustituible (backend de storage configurable).

**Prueba Independiente**: Puede probarse con URIs de archivos locales (`file://` o rutas)
sobre `tmp_path`: obtener una imagen existente devuelve una ruta local legible; una URI
inexistente lanza `ImageAccessError`; guardar un artefacto devuelve una URI y el archivo
persiste.

**Escenarios de Aceptación**:

1. **Escenario**: Imagen existente se materializa localmente
   - **Dado** una imagen accesible en la URI indicada
   - **Cuando** el almacén obtiene la imagen
   - **Entonces** devuelve una ruta local desde la que el motor puede leer la imagen

2. **Escenario**: Imagen inexistente lanza ImageAccessError
   - **Dado** una URI que no apunta a ninguna imagen accesible
   - **Cuando** el almacén intenta obtenerla
   - **Entonces** lanza `ImageAccessError` (con la causa encadenada si existe)

3. **Escenario**: Artefacto de salida se persiste y devuelve URI
   - **Dado** un archivo de salida producido por la inferencia
   - **Cuando** el almacén lo persiste como artefacto del estudio
   - **Entonces** devuelve la URI del artefacto persistido, apta para el bloque
     `artifacts` del resultado

---

### Historia de Usuario 4 - Smoke test de inferencia real (Prioridad: P4)

Como desarrollador del servicio, necesito un test de integración (marcado
`@pytest.mark.integration`, omitido por defecto) que descargue un bundle pequeño real y
ejecute el pipeline completo sobre un volumen mínimo (`tiny_volume.nii.gz`), para validar
que la cadena registry → image store → engine funciona con MONAI de verdad y no solo con
mocks.

**Por qué esta prioridad**: Red de seguridad valiosa pero no bloqueante: requiere red y
descarga de pesos, por eso se excluye de la suite por defecto (DESIGN.md §6).

**Prueba Independiente**: `uv run pytest -m integration tests/integration/test_inference_smoke.py`
con red disponible; verifica que se produce una salida de inferencia sin excepciones.

**Escenarios de Aceptación**:

1. **Escenario**: Pipeline real de extremo a extremo sobre un volumen mínimo
   - **Dado** conectividad de red y el fixture `tests/fixtures/tiny_volume.nii.gz`
   - **Cuando** se resuelve el bundle real, se obtiene el volumen y se ejecuta la
     inferencia
   - **Entonces** el pipeline completa sin excepciones y produce una salida de modelo no
     vacía

---

### Casos Límite

- ¿Qué ocurre si la inferencia falla después de obtener la imagen? → Resultado `failed`
  publicado con `error`; los artefactos quedan vacíos y las métricas vacías.
- ¿Qué ocurre si falla el guardado de un artefacto (tras inferencia exitosa)? →
  `ImageAccessError` se propaga (transitorio: el consumidor decidirá reintento en v0.5.0);
  no se publica un resultado parcial.
- ¿Qué ocurre con un `ModelRef` cuyo `local_path` no existe o está corrupto al ejecutar?
  → El motor lo traduce a `InferenceError` (fallo de ejecución del modelo).
- ¿Qué ocurre con una URI de esquema no soportado por el backend de storage configurado?
  → `ImageAccessError`.
- ¿Qué ocurre con la duración cuando la inferencia falla? → `duration_ms` refleja el
  tiempo transcurrido hasta el fallo (trazabilidad de rendimiento incluso en errores).
- El resultado `failed` publicado NUNCA incluye trazas internas ni rutas locales en el
  bloque `error` (solo código de error del dominio y mensaje breve, sin PII).

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: `application/analysis_service.py` DEBE exponer un caso de uso que reciba
  una `AnalysisRequest` y orqueste: resolución de bundle → obtención de imagen →
  inferencia → persistencia de artefactos → construcción del `AnalysisResult` →
  publicación.
- **RF-002**: La capa Application NO DEBE importar `pika`, `monai`, `torch` ni módulos de
  `infrastructure/`; sus colaboradores (registry, image store, engine, publisher) se
  definen como contratos abstractos (protocolos) en Application y se inyectan desde el
  composition root (DIP, DESIGN.md §3).
- **RF-003**: El `AnalysisResult` exitoso DEBE incluir: `status="succeeded"`, el
  `ModelRef` resuelto, artefactos con URIs persistidas, métricas de la inferencia,
  `processed_at` (UTC) y `duration_ms` medido por el servicio.
- **RF-004**: Ante `InferenceError`, el servicio DEBE construir y publicar un
  `AnalysisResult` con `status="failed"` y bloque `error` `{code, message}`, y retornar
  normalmente (DESIGN.md §7: no se pierde trazabilidad; el consumidor hará `ack`).
- **RF-005**: Ante `BundleResolutionError`, `ImageAccessError` o
  `UnknownAnalysisTypeError` el servicio DEBE dejar propagar la excepción sin publicar
  resultado (la política de reintento/dead-letter es del consumidor, v0.5.0).
- **RF-006**: `infrastructure/inference_engine.py` DEBE ejecutar el pipeline del bundle
  MONAI (pre-proceso → red → inferer → post-proceso) usando el `local_path` del
  `ModelRef` y el dispositivo `MODEL_DEVICE`, devolviendo la salida del modelo y sus
  métricas.
- **RF-007**: El motor DEBE traducir cualquier excepción del pipeline (torch/monai) a
  `InferenceError` con la causa encadenada.
- **RF-008**: `infrastructure/image_store.py` DEBE materializar la imagen referenciada
  por `image_uri` en una ruta local de trabajo y DEBE persistir artefactos de salida
  devolviendo sus URIs.
- **RF-009**: El almacén DEBE lanzar `ImageAccessError` cuando la imagen no exista, la
  URI use un esquema no soportado o falle la persistencia de un artefacto.
- **RF-010**: El backend de almacenamiento DEBE configurarse vía
  `IMAGE_STORAGE_BACKEND` (Settings). Esta versión implementa ÚNICAMENTE el backend de
  sistema de archivos local (URIs `file://` y rutas locales); S3 u otros backends se
  difieren a versiones futuras (aclarado por el desarrollador). Un backend configurado
  no soportado DEBE fallar de forma explícita.
- **RF-011**: `monai`/`torch` DEBEN quedar confinados a `infrastructure/` con imports
  perezosos (mismo patrón que `bundle_registry`), de modo que la suite unitaria no los
  cargue.
- **RF-012**: Los unit tests DEBEN cubrir los escenarios del CHANGELOG v0.4.0:
  `test_run_analysis_publishes_result`, `test_image_store_fetch_missing_raises`,
  `test_inference_engine_failure_raises`,
  `test_analysis_service_builds_metrics_and_artifacts`.
- **RF-013**: DEBE existir `tests/integration/test_inference_smoke.py` marcado
  `@pytest.mark.integration` (omitido por defecto) que descargue un bundle pequeño real y
  ejecute la inferencia sobre `tests/fixtures/tiny_volume.nii.gz` (fixture nuevo, volumen
  mínimo generado sintéticamente, sin datos de paciente).
- **RF-014**: El logging del flujo DEBE registrar `request_id`, `correlation_id`,
  `analysis_type`, modelo, duración y estado — nunca PII ni píxeles (DESIGN.md §8).

### Entidades Clave *(si aplica)*

- **`InferenceOutput`** *(nueva, capa Domain)*: salida del motor de inferencia consumida
  por el caso de uso — referencia(s) al/los archivo(s) de predicción producidos
  localmente (a persistir como artefactos, con su tipo semántico) y métricas calculadas
  (`dict[str, float]`). Sin píxeles en memoria del dominio, sin dependencias externas.
- **Puertos de Application** *(nuevos, capa Application)*: contratos abstractos que el
  caso de uso requiere — resolutor de bundles, almacén de imágenes, motor de inferencia y
  publicador de resultados. Las implementaciones reales viven en `infrastructure/`
  (v0.3.0/v0.4.0) y `presentation`/`infrastructure` (publisher, v0.5.0).

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: Los cuatro escenarios unitarios del CHANGELOG v0.4.0 (RF-012) pasan en sus
  archivos espejo (`tests/unit/application/test_analysis_service.py`,
  `tests/unit/infrastructure/test_image_store.py`,
  `tests/unit/infrastructure/test_inference_engine.py`).
- **CE-002**: La secuencia completa de verificación pasa sin errores:
  `uv run ruff format --check .` → `uv run ruff check .` → `uv run mypy src` →
  `uv run pytest`.
- **CE-003**: La suite unitaria completa corre sin red y sin cargar `monai`/`torch`
  (imports perezosos verificables), y sin RabbitMQ real.
- **CE-004**: Cobertura global ≥ 85% incluyendo los tres módulos nuevos.
- **CE-005**: `uv run pytest` (por defecto) omite el smoke de integración; con
  `-m integration` y red disponible, el smoke descarga el bundle y completa la
  inferencia sin excepciones.
- **CE-006**: Los tests de v0.1.0–v0.3.0 siguen pasando sin cambios de contrato.
