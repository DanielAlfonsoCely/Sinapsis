# Plan de Implementación: Motor de Inferencia + Caso de Uso de Análisis (v0.4.0)

**Fecha**: 2026-07-05
**Especificación**: [spec.md](./spec.md)

## Resumen

Implementar el corazón del microservicio: `application/analysis_service.py` (caso de uso
que orquesta bundle → imagen → inferencia → artefactos → resultado → publicación),
`infrastructure/inference_engine.py` (ejecuta el pipeline del bundle MONAI) e
`infrastructure/image_store.py` (backend local de imágenes/artefactos por URI, S3
diferido). Torch/MONAI mockeados en unit tests; smoke real en integración.

**Enfoque técnico**: la capa Application define **puertos** (`typing.Protocol`) para sus
cuatro colaboradores y no importa nada de infraestructura (DIP). Se agrega la entidad
`InferenceOutput` al dominio (archivos de predicción por tipo + métricas). El motor MONAI
usa una **fábrica de workflows inyectable** (mismo patrón que `download_fn` en v0.3.0):
el default hace import perezoso de `monai.bundle.create_workflow`; los unit tests
inyectan fakes → sin red y sin cargar torch. El image store local resuelve URIs
`file://`/rutas y persiste artefactos bajo un directorio raíz configurado por
constructor.

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12 (gestor `uv`, layout `src/`)
**Dependencias Principales**: `monai`/`torch` (ya presentes desde v0.3.0); **nueva**:
`nibabel>=5` (I/O NIfTI — requerida por MONAI para leer/escribir `.nii.gz` en runtime y
para generar el fixture del smoke test)
**Almacenamiento**: sistema de archivos local — imágenes de entrada por URI
`file://`/ruta local; artefactos bajo un `root_dir` del store; directorio temporal de
trabajo por inferencia
**Testing**: pytest + pytest-cov, cobertura ≥ 85%; unit sin red/sin torch; integración
marcada `integration` (omitida por defecto, requiere red para descargar bundle real)
**Plataforma Objetivo**: servidor Linux (worker RabbitMQ containerizado)
**Tipo de Proyecto**: proyecto único (microservicio, arquitectura por capas simple)
**Objetivos de Rendimiento**: `duration_ms` medido por el servicio en cada análisis
(éxito y fallo); una inferencia por proceso (prefetch=1, sin concurrencia interna)
**Restricciones**: Application no importa `pika`/`monai`/`torch`/`infrastructure`;
`monai`/`torch` confinados a `infrastructure/` con imports perezosos; sin PII en logs ni
en bloques `error`; mypy `strict`
**Escala/Alcance**: 1 entidad de dominio nueva + 1 módulo de puertos + 1 caso de uso +
2 módulos de infraestructura + 4 archivos de test unitarios + 1 smoke de integración +
1 fixture binario

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v0.4.0-inference-pipeline/
├── plan.md  # Este archivo
└── spec.md
```

### Código Fuente (raíz de `microservice/`)

```text
pyproject.toml                                    # MODIFICAR: + nibabel
src/
└── sinapsis_ai/
    ├── domain/
    │   └── models.py                             # MODIFICAR: + InferenceOutput
    ├── application/
    │   ├── ports.py                              # NUEVO: Protocols (4 puertos)
    │   └── analysis_service.py                   # NUEVO: caso de uso AnalysisService
    └── infrastructure/
        ├── inference_engine.py                   # NUEVO: MonaiInferenceEngine
        └── image_store.py                        # NUEVO: LocalImageStore

tests/
├── fixtures/
│   └── tiny_volume.nii.gz                        # NUEVO: volumen sintético mínimo
├── unit/
│   ├── domain/test_models.py                     # MODIFICAR: + tests InferenceOutput
│   ├── application/
│   │   ├── __init__.py                           # NUEVO
│   │   └── test_analysis_service.py              # NUEVO
│   └── infrastructure/
│       ├── test_inference_engine.py              # NUEVO
│       └── test_image_store.py                   # NUEVO
└── integration/
    ├── __init__.py                               # NUEVO
    └── test_inference_smoke.py                   # NUEVO: @pytest.mark.integration
```

**Decisión de Estructura**: árbol planificado en PROJECT_ARCHITECTURE.md §3; tests espejo
por capa (AGENTS.md). El publisher real NO se implementa (v0.5.0): el puerto
`ResultPublisher` se define en Application y los tests usan dobles.

**Decisiones técnicas**:

1. **`InferenceOutput` (Domain)**: dataclass congelada con
   `artifact_paths: dict[str, str]` (tipo semántico de artefacto → ruta local del
   archivo de predicción) y `metrics: dict[str, float]`. Sin píxeles, sin deps externas.
2. **Puertos como `typing.Protocol`** en `application/ports.py`: `BundleResolver`
   (`resolve`), `ImageStore` (`fetch`, `save_artifact`), `InferenceEngine` (`run`),
   `ResultPublisher` (`publish`). `BundleRegistry` (v0.3.0) ya satisface
   estructuralmente `BundleResolver` — no se modifica.
3. **Política de errores del servicio** (DESIGN.md §7): `InferenceError` → construye y
   publica resultado `failed` con `error={"code": "INFERENCE_ERROR", "message": ...}` y
   retorna; `BundleResolutionError`/`ImageAccessError`/`UnknownAnalysisTypeError` →
   propagan sin publicar. `duration_ms` vía `time.monotonic()` en ambos caminos.
4. **Motor MONAI**: `MonaiInferenceEngine(device, workflow_factory=None)`. La fábrica
   default (import perezoso) crea el workflow de inferencia del bundle
   (`monai.bundle.create_workflow`, `workflow_type="infer"`) con `bundle_root` del
   `ModelRef`, la imagen de entrada y un directorio de salida temporal por ejecución;
   ejecuta `initialize → run → finalize` y escanea el directorio de salida para poblar
   `artifact_paths` (`segmentation_mask` → archivo producido). `metrics` queda `{}` en
   la implementación real de esta versión (las métricas derivadas, p. ej. `volume_ml`,
   se abordarán cuando se defina su cálculo clínico); los dobles de HU1 sí ejercitan el
   paso de métricas extremo a extremo.
5. **`LocalImageStore(root_dir)`**: `fetch` acepta URIs `file://` y rutas locales
   (esquemas no soportados como `s3://` → `ImageAccessError`); `save_artifact` copia el
   archivo a `<root_dir>/<study_id>/<nombre>` y devuelve URI `file://` absoluta. El
   `root_dir` es parámetro de constructor; el composition root (v0.5.0) lo derivará de
   configuración.
6. **Fixture `tiny_volume.nii.gz`**: volumen sintético (numpy + nibabel, sin datos de
   paciente) generado una vez durante la implementación y versionado (es diminuto). El
   smoke de integración lo consume con el bundle real `spleen_ct_segmentation`.

---

## Fase 1: Configuración (Infraestructura Compartida)

**Propósito**: Dependencia NIfTI y paquetes de test nuevos

- [x] T001 Agregar `nibabel>=5` a `dependencies` en `pyproject.toml` y ejecutar
      `uv sync --all-extras`
- [x] T002 Crear paquetes de test: `tests/unit/application/__init__.py` y
      `tests/integration/__init__.py`

---

## Fase 2: Fundacional (Prerequisitos Bloqueantes)

**Propósito**: Entidad `InferenceOutput` (Domain) y puertos de Application — todos los
componentes de esta versión dependen de estos contratos

**⚠️ CRÍTICO**: Ningún trabajo de historia de usuario puede comenzar hasta que esta fase
esté completa

- [x] T003 Agregar `InferenceOutput` (frozen dataclass: `artifact_paths: dict[str, str]`,
      `metrics: dict[str, float]`) a `src/sinapsis_ai/domain/models.py`
- [x] T004 Agregar tests de `InferenceOutput` en `tests/unit/domain/test_models.py`
      (construcción, defaults, inmutabilidad)
- [x] T005 Crear `src/sinapsis_ai/application/ports.py` con los `Protocol`:
      `BundleResolver`, `ImageStore`, `InferenceEngine`, `ResultPublisher` (solo Domain
      en imports; RF-002)

**Punto de Control**: Fundación lista — verificación AGENTS.md verde; la implementación
de historias puede comenzar

---

## Fase 3: Historia de Usuario 1 - Orquestar un análisis de extremo a extremo (Prioridad: P1)

**Objetivo**: `AnalysisService.run_analysis(request)` orquesta el flujo completo con
colaboradores inyectados y publica siempre un resultado ante inferencia fallida

**Prueba Independiente**: dobles inyectados de los 4 puertos; verificar secuencia,
resultado construido/publicado y política de errores — sin MONAI ni RabbitMQ

### Implementación para Historia de Usuario 1

- [x] T006 [P] [HU1] Crear `src/sinapsis_ai/application/analysis_service.py`:
      `AnalysisService(bundle_resolver, image_store, inference_engine, publisher)` con
      `run_analysis(request: AnalysisRequest) -> AnalysisResult` — flujo feliz: resolver
      → fetch → run → save_artifact por cada entrada de `artifact_paths` → construir
      `AnalysisResult(succeeded)` con métricas/artefactos/duración → publicar (RF-001,
      RF-003) (depende de T003, T005)
- [x] T007 [HU1] Implementar política de errores en `run_analysis`: capturar
      `InferenceError` → resultado `failed` con bloque `error` sin trazas internas,
      publicar y retornar (RF-004); dejar propagar `BundleResolutionError`/
      `ImageAccessError`/`UnknownAnalysisTypeError` sin publicar (RF-005); logging con
      `request_id`/`correlation_id`/`analysis_type`/modelo/duración/estado, sin PII
      (RF-014) (depende de T006)

### Pruebas para Historia de Usuario 1

- [x] T008 [P] [HU1] Crear `tests/unit/application/test_analysis_service.py` con dobles
      de los 4 puertos y tests: `test_run_analysis_publishes_result` (flujo feliz,
      resultado publicado = retornado), `test_analysis_service_builds_metrics_and_artifacts`
      (métricas del `InferenceOutput` y artefactos con URIs del store),
      `test_inference_error_publishes_failed_result` (status, bloque error, retorno
      normal, duración > 0), `test_bundle_resolution_error_propagates_without_publish`,
      `test_image_access_error_propagates_without_publish` (depende de T007)

**Punto de Control**: HU1 completamente funcional y testeable de forma independiente —
verificación AGENTS.md verde

---

## Fase 4: Historia de Usuario 2 - Ejecutar el pipeline de inferencia del bundle (Prioridad: P2)

**Objetivo**: `MonaiInferenceEngine.run(model, image_path)` ejecuta el workflow del
bundle en `MODEL_DEVICE` y devuelve `InferenceOutput`; fallos → `InferenceError`

**Prueba Independiente**: fábrica de workflows fake inyectada; verificar parámetros de
creación (bundle_root, imagen, device), ciclo `initialize/run/finalize`, escaneo de
salidas y traducción de errores

### Implementación para Historia de Usuario 2

- [x] T009 [P] [HU2] Crear `src/sinapsis_ai/infrastructure/inference_engine.py`:
      `MonaiInferenceEngine(device, workflow_factory=None)` con fábrica default de
      import perezoso (`monai.bundle.create_workflow`, `workflow_type="infer"`);
      `run(model: ModelRef, image_path: str) -> InferenceOutput` crea un directorio de
      salida temporal por ejecución, ejecuta el ciclo del workflow, escanea el
      directorio y devuelve `artifact_paths={"segmentation_mask": <archivo>}` (RF-006,
      RF-011)
- [x] T010 [HU2] Traducción de errores: cualquier excepción de la fábrica o del ciclo de
      ejecución → `raise InferenceError(...) from exc`; ejecución sin archivos de salida
      producidos → `InferenceError`; logging operativo sin PII (RF-007) (depende de T009)

### Pruebas para Historia de Usuario 2

- [x] T011 [P] [HU2] Crear `tests/unit/infrastructure/test_inference_engine.py` con
      fábrica fake (materializa archivos de salida) y tests:
      `test_engine_runs_workflow_and_returns_output` (params correctos + artifact_paths),
      `test_inference_engine_failure_raises` (workflow lanza → `InferenceError` con
      causa), `test_engine_no_outputs_raises_inference_error` (depende de T010)

**Punto de Control**: HU1 y HU2 funcionan de forma independiente — verificación verde

---

## Fase 5: Historia de Usuario 3 - Obtener imágenes y persistir artefactos por URI (Prioridad: P3)

**Objetivo**: `LocalImageStore` materializa imágenes desde URIs `file://`/rutas locales y
persiste artefactos devolviendo URIs `file://`

**Prueba Independiente**: `tmp_path` como raíz; fetch de archivo existente/ausente,
esquema no soportado, y round-trip de save_artifact

### Implementación para Historia de Usuario 3

- [x] T012 [P] [HU3] Crear `src/sinapsis_ai/infrastructure/image_store.py`:
      `LocalImageStore(root_dir)` con `fetch(image_uri) -> str` (acepta `file://` y
      rutas; valida existencia; esquema no soportado o archivo ausente →
      `ImageAccessError`) y `save_artifact(study_id, artifact_type, local_path) -> str`
      (copia a `<root_dir>/<study_id>/`, crea directorios, devuelve URI `file://`;
      fallos → `ImageAccessError`) (RF-008, RF-009, RF-010)

### Pruebas para Historia de Usuario 3

- [x] T013 [P] [HU3] Crear `tests/unit/infrastructure/test_image_store.py` con tests:
      `test_image_store_fetch_existing_returns_local_path` (ruta y URI `file://`),
      `test_image_store_fetch_missing_raises` → `ImageAccessError`,
      `test_image_store_unsupported_scheme_raises` (`s3://...`) → `ImageAccessError`,
      `test_image_store_save_artifact_persists_and_returns_uri` (archivo copiado + URI
      resoluble), `test_image_store_save_artifact_failure_raises` (origen inexistente)
      (depende de T012)

**Punto de Control**: HU1–HU3 funcionan de forma independiente — verificación verde

---

## Fase 6: Historia de Usuario 4 - Smoke test de inferencia real (Prioridad: P4)

**Objetivo**: test de integración que descarga el bundle real y ejecuta el pipeline sobre
un volumen sintético mínimo

**Prueba Independiente**: `uv run pytest -m integration tests/integration/test_inference_smoke.py`
con red disponible

### Implementación para Historia de Usuario 4

- [x] T014 [P] [HU4] Generar `tests/fixtures/tiny_volume.nii.gz`: volumen sintético
      pequeño (numpy + nibabel, affine identidad, sin datos de paciente), versionable
- [x] T015 [HU4] Crear `tests/integration/test_inference_smoke.py` marcado
      `@pytest.mark.integration`: `test_inference_smoke_downloads_bundle_and_runs` —
      cadena real `BundleRegistry` → `LocalImageStore` → `MonaiInferenceEngine` sobre el
      fixture; asserts: sin excepciones y `artifact_paths` no vacío (RF-013)
      (depende de T014)

**Punto de Control**: smoke ejecutable bajo `-m integration`; `uv run pytest` por defecto
lo omite (CE-005)

---

## Fase 7: Acabado y Preocupaciones Transversales

**Propósito**: verificación integral, cobertura, aislamiento de imports y documentación

- [x] T016 Ejecutar suite completa en orden AGENTS.md: `uv run ruff format --check .` →
      `uv run ruff check .` → `uv run mypy src` →
      `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` (≥ 85%, CE-002/CE-004);
      corregir hallazgos
- [x] T017 Verificar CE-003: importar `application/` e `infrastructure/` nuevos no carga
      `monai`/`torch`/`pika` (script de verificación de imports perezosos)
- [x] T018 Ejecutar smoke de integración (`uv run pytest -m integration`) si hay red
      disponible; documentar resultado (CE-005)
- [x] T019 Actualizar `docs/CHANGELOG.md`: marcar v0.4.0 como COMPLETADO con fecha

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Configuración (Fase 1)**: sin dependencias — puede comenzar de inmediato
- **Fundacional (Fase 2)**: depende de Fase 1 — BLOQUEA todas las historias (entidad y
  puertos son los contratos de todo lo demás)
- **Historias de Usuario (Fases 3–6)**: dependen de la Fase 2
  - HU1–HU3 son independientes entre sí (archivos distintos); se ejecutan
    secuencialmente por prioridad P1 → P2 → P3
  - HU4 (smoke) depende de las implementaciones reales de HU2 y HU3 y del registry
    v0.3.0
- **Acabado (Fase 7)**: depende de todas las historias completas

### Dependencias entre Historias de Usuario

- **HU1 (P1)**: solo depende de Fundacional — usa dobles de los 4 puertos
- **HU2 (P2)**: solo depende de Fundacional — testeable con fábrica fake, sin HU1/HU3
- **HU3 (P3)**: solo depende de Fundacional — testeable con `tmp_path`, sin HU1/HU2
- **HU4 (P4)**: integra HU2 + HU3 + registry (v0.3.0); no usa HU1 (el smoke valida la
  cadena de infraestructura real, no la orquestación)

### Dentro de Cada Historia

- Modelos antes que servicios: `InferenceOutput` y puertos (Fase 2) antes que todo
- Implementación antes que pruebas: T006–T007 → T008; T009–T010 → T011; T012 → T013;
  T014 → T015
- Historia completa (verificación verde) antes de la siguiente prioridad

## Notas

- [P] = tarea prioritaria dentro de la historia; [HU#] = trazabilidad a la historia
- El puerto `ResultPublisher` queda definido pero su implementación real (pika) es
  alcance de v0.5.0; los tests de HU1 usan un doble que captura el resultado publicado
- S3 diferido explícitamente (aclaración del desarrollador en el spec, RF-010)
- El smoke de integración requiere red y descarga el bundle real (~100 MB); se ejecuta
  solo bajo `-m integration`
- Commit sugerido por fase o grupo lógico; los commits los decide el desarrollador

---

## Ejecución Completada

**Fecha**: 2026-07-05
**Estado**: Todas las fases implementadas y verificadas

- Tests unitarios: 69 passed, 1 deselected (integración)
- Smoke de integración: 1 passed (bundle real `spleen_ct_segmentation` v0.6.1,
  inferencia CPU sobre `tiny_volume.nii.gz`)
- Cobertura global: 91.67% (≥ 85%)
- Verificación AGENTS.md: ruff format ✓ · ruff check ✓ · mypy src ✓ · pytest ✓
- Imports perezosos verificados: importar application/infrastructure no carga
  monai/torch/pika/ignite

**Desviaciones del plan y decisiones tomadas durante la ejecución**:
- Dependencias adicionales no previstas, requeridas por la descarga y ejecución real
  de bundles con monai 1.6: `requests`, `huggingface_hub` (la descarga
  `monaihosting` se sirve vía HF Hub) y `pytorch-ignite` (requerido por los engines
  del bundle, declarado en su `metadata.json`). `tensorboard` NO fue necesario para
  inferencia.
- `pyproject.toml`: `addopts` ahora incluye `-m 'not integration'` — el marker estaba
  declarado pero NO deseleccionado por defecto; sin este cambio `uv run pytest`
  habría ejecutado el smoke (descarga real). Un `-m integration` explícito lo
  sobreescribe.
- `filterwarnings`: se agregaron ignores dirigidos a warnings de terceros
  (torch/monai/huggingface_hub/ignite); los warnings de `sinapsis_ai` siguen
  fallando tests.
- La fábrica de workflows agrega el override `checkpointloader#map_location=<device>`:
  los checkpoints del Model Zoo se guardan en CUDA y sin él la carga falla en
  workers CPU-only.
- Fixture `tiny_volume.nii.gz` (82 KB): volumen sintético int16 64×64×32 con blob
  elipsoidal, generado con numpy+nibabel (seed fija, sin datos de paciente).
