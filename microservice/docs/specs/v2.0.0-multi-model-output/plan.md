# Plan de Implementación: Multi-Model Output Generalizado (v2.0.0)

**Fecha**: 2026-07-13
**Especificación**: [spec.md](./spec.md)

---

## Resumen

Generalizar el motor de inferencia introduciendo el protocolo `OutputExtractor` con tres
implementaciones concretas (`SegmentationExtractor`, `ClassificationExtractor`,
`DetectionExtractor`), añadir tres nuevos `AnalysisType` con sus bundles MONAI reales
(uno por cada tipo de output), y cablear todo en `MonaiInferenceEngine` mediante un
catálogo inyectable. El contrato `AnalysisResult` no cambia.

---

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12
**Dependencias Principales**: `monai`, `torch`, `nibabel` (ya presentes); `numpy` para
cómputo de volumen vóxeles (disponible como dependencia transitiva de monai/torch)
**Almacenamiento**: N/A (los artefactos se procesan localmente y se referencian por URI)
**Testing**: `pytest` + `pytest-cov`; umbral ≥ 85%; extractores testeados con
`tmp_path` y fixtures de archivos sintéticos; sin MONAI real en unit tests
**Plataforma Objetivo**: Servidor Linux (contenedor Docker)
**Objetivos de Rendimiento**: Sin overhead en el camino feliz; extracción de métricas
en < 100 ms para archivos de output típicos
**Restricciones**: `application/` no importa nada de infrastructure; `domain/models.py`
no importa numpy/nibabel; extractores solo en `infrastructure/`; el mapa extractor por
defecto vive en `inference_engine.py` (composition local)
**Escala/Alcance**: 4 tipos de análisis tras v2.0.0; el catálogo es extensible O(1)

---

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v2.0.0-multi-model-output/
├── plan.md   # Este archivo
└── spec.md
```

### Código Fuente

```text
src/sinapsis_ai/
├── domain/
│   └── models.py                          # Modificar: +3 miembros a AnalysisType
└── infrastructure/
    ├── output_extractors.py               # NUEVO: OutputExtractor protocol + 3 impl
    ├── bundle_registry.py                 # Modificar: +3 entradas en _BUNDLE_CATALOG
    └── inference_engine.py                # Modificar: recibe extractor_catalog, delega

tests/
└── unit/
    └── infrastructure/
        ├── test_output_extractors.py      # NUEVO: tests exhaustivos de los 3 extractores
        ├── test_bundle_registry.py        # Ampliar: test_catalog_covers_all ya pasa
        └── test_inference_engine.py       # Ampliar: tests de dispatch por AnalysisType
```

**Decisión de Estructura**: Los extractores viven en `infrastructure/output_extractors.py`
porque leen archivos producidos por MONAI (detalle técnico de infraestructura). El mapa
`_EXTRACTOR_CATALOG` vive en `inference_engine.py` junto al engine que lo usa — es su
composition local, no del composition root global.

---

## Fase 1: Domain — Nuevos AnalysisType

**Propósito**: Añadir los 3 nuevos miembros al enum. Es el primer paso porque todos los
demás módulos dependen de que `AnalysisType` los conozca. Sin esto nada más compila.

**CRITICO**: Fases 2, 3 y 4 dependen de esta fase.

- [ ] T001 Modificar `src/sinapsis_ai/domain/models.py`: añadir a `AnalysisType`:
  ```python
  CT_LUNG_NODULE_DETECTION          = "ct_lung_nodule_detection"
  MRI_BRAIN_TUMOR_SEGMENTATION      = "mri_brain_tumor_segmentation"
  XR_BREAST_DENSITY_CLASSIFICATION  = "xr_breast_density_classification"
  ```

**Punto de Control**: `uv run mypy src` pasa; `uv run python -m pytest tests/unit/domain/ -q` pasa

---

## Fase 2: Infrastructure — OutputExtractor protocol + 3 implementaciones (Fundacional)

**Propósito**: Crear el protocolo y las 3 implementaciones concretas. Es el prerequisito
bloqueante para modificar el engine (Fase 3) y para añadir los bundles (Fase 4).

**CRITICO**: Las fases 3 y 4 no pueden comenzar hasta completar esta fase.

### Pruebas para OutputExtractor

- [ ] T002 [P] [HU1] Escribir `tests/unit/infrastructure/test_output_extractors.py`:

  **SegmentationExtractor**:
  - `test_segmentation_extractor_finds_nifti_and_computes_volume` — `output_dir` con
    `.nii.gz` sintético (bytes mínimos válidos); verifica `artifact_paths["segmentation_mask"]`
    apunta al archivo y `metrics["volume_voxels"] >= 0`
  - `test_segmentation_extractor_empty_dir_raises` — `output_dir` vacío → `InferenceError`
  - `test_segmentation_extractor_no_nifti_raises` — solo `.txt` → `InferenceError`

  **ClassificationExtractor**:
  - `test_classification_extractor_reads_json_scores` — JSON `{"A": 0.1, "B": 0.7, "C": 0.15, "D": 0.05}`
    → `metrics={"probability": 0.7, "predicted_class": "B"}`, `artifact_paths={}`
  - `test_classification_extractor_empty_dir_raises` → `InferenceError`
  - `test_classification_extractor_invalid_json_raises` → `InferenceError`

  **DetectionExtractor**:
  - `test_detection_extractor_reads_json_detections` — JSON con lista de detecciones
    → `metrics={"lesion_count": 3, "max_diameter_mm": 12.4}`, `artifact_paths={}`
  - `test_detection_extractor_empty_detections_returns_zero_count` — lista vacía
    → `metrics={"lesion_count": 0, "max_diameter_mm": 0.0}`
  - `test_detection_extractor_empty_dir_raises` → `InferenceError`

### Implementación de OutputExtractor

- [ ] T003 [P] [HU1] Implementar `src/sinapsis_ai/infrastructure/output_extractors.py`:

  ```python
  # Protocolo (runtime-checkable structural type)
  class OutputExtractor(Protocol):
      def extract(self, output_dir: Path) -> InferenceOutput: ...

  # Segmentación: busca primer .nii.gz, calcula vóxeles no-cero
  class SegmentationExtractor:
      def extract(self, output_dir: Path) -> InferenceOutput: ...

  # Clasificación: lee scores JSON, extrae clase con max probabilidad
  class ClassificationExtractor:
      def extract(self, output_dir: Path) -> InferenceOutput: ...

  # Detección: lee JSON de detecciones, extrae lesion_count y max_diameter_mm
  class DetectionExtractor:
      def extract(self, output_dir: Path) -> InferenceOutput: ...
  ```

  Notas de implementación:
  - `SegmentationExtractor`: importa `numpy` de forma lazy (solo en el método extract)
    para no penalizar el arranque; usa `np.count_nonzero` sobre el array del NIfTI
  - `ClassificationExtractor`: busca el primer `.json` en `output_dir` con el formato
    `{class_label: score}` y retorna la clase de mayor score
  - `DetectionExtractor`: busca el primer `.json` con estructura de lista de detecciones;
    cada detección puede tener `diameter_mm` opcional; si no hay detecciones → count=0
  - Todos: si `output_dir` vacío o sin el archivo esperado → `InferenceError`

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_output_extractors.py -q` pasa

---

## Fase 3: Infrastructure — Engine con dispatch por AnalysisType (HU1)

**Propósito**: Modificar `MonaiInferenceEngine` para recibir un `extractor_catalog`
inyectable y delegar la extracción del output al extractor correcto según `AnalysisType`.

**Prueba Independiente**: `FakeWorkflowFactory` existente + extractores mockeados o reales.

### Pruebas para el Engine con dispatch

- [ ] T004 [P] [HU1] Ampliar `tests/unit/infrastructure/test_inference_engine.py`:
  - `test_engine_dispatches_to_correct_extractor` — engine con dos extractores mockeados
    en el catálogo; verifica que se llama al extractor correcto según `model.name`
  - `test_engine_unknown_analysis_type_raises_inference_error` — engine sin extractor
    para el tipo → `InferenceError`
  - `test_engine_uses_segmentation_extractor_for_spleen` — verifica el mapa por defecto
    asigna `SegmentationExtractor` a `CT_SPLEEN_SEGMENTATION`
  - Actualizar `test_engine_runs_workflow_and_returns_output` para pasar `AnalysisType`
    al engine (firma de `run` cambia de `ModelRef` a `(ModelRef, AnalysisType)`)

### Implementación del Engine con dispatch

- [ ] T005 [HU1] Modificar `src/sinapsis_ai/infrastructure/inference_engine.py`:
  - Añadir parámetro `extractor_catalog: dict[AnalysisType, OutputExtractor] | None`
    al constructor (default `None` → usa `_EXTRACTOR_CATALOG` por defecto)
  - Añadir parámetro `analysis_type: AnalysisType` a `run()`
  - Dentro de `run()`: buscar el extractor en el catálogo; si no existe → `InferenceError`;
    llamar `extractor.extract(Path(output_dir))` en lugar del hardcoding actual
  - Definir `_EXTRACTOR_CATALOG: dict[AnalysisType, OutputExtractor]` en el módulo:
    ```python
    _EXTRACTOR_CATALOG = {
        AnalysisType.CT_SPLEEN_SEGMENTATION:         SegmentationExtractor(),
        AnalysisType.CT_LUNG_NODULE_DETECTION:        DetectionExtractor(),
        AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION:    SegmentationExtractor(),
        AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION: ClassificationExtractor(),
    }
    ```

- [ ] T006 [HU1] Actualizar `src/sinapsis_ai/application/analysis_service.py`:
  - `run_analysis()` llama a `self._inference_engine.run(model, image_path, request.analysis_type)`
    (nuevo parámetro `analysis_type`)
  - El port `InferenceEngine` en `application/ports.py` debe actualizarse con la nueva firma

- [ ] T007 [HU1] Actualizar `src/sinapsis_ai/application/ports.py`:
  - Añadir `analysis_type: AnalysisType` al método `run()` del protocolo `InferenceEngine`

**Punto de Control**: `uv run pytest tests/unit/ -q` pasa (todos los tests existentes + nuevos)

---

## Fase 4: Infrastructure — Bundle catalog (+3 bundles)

**Propósito**: Registrar los 3 nuevos bundles en `_BUNDLE_CATALOG`. El test
`test_catalog_covers_all_analysis_types` ya existente fallará hasta completar esta fase,
lo cual es el comportamiento esperado (test-driven).

- [ ] T008 [P] Modificar `src/sinapsis_ai/infrastructure/bundle_registry.py`:
  añadir a `_BUNDLE_CATALOG`:
  ```python
  AnalysisType.CT_LUNG_NODULE_DETECTION:         "lung_nodule_ct_detection",
  AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION:     "brats_mri_segmentation",
  AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION: "breast_density_classification",
  ```

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_bundle_registry.py -q`
pasa incluyendo `test_catalog_covers_all_analysis_types`

---

## Fase 5: Acabado y verificación final

- [ ] T009 Ejecutar `uv run ruff format --check .` y corregir
- [ ] T010 Ejecutar `uv run ruff check .` y corregir
- [ ] T011 Ejecutar `uv run mypy src` y corregir errores de tipo
- [ ] T012 Ejecutar `uv run pytest` y verificar 100% passing
- [ ] T013 Ejecutar `uv run pytest --cov=sinapsis_ai --cov-report=term-missing`
  y verificar cobertura ≥ 85%
- [ ] T014 Actualizar `docs/CHANGELOG.md`: añadir entrada `[v2.0.0]` como `COMPLETADO`

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Fase 1 (Domain)**: Sin dependencias — primer paso obligatorio
- **Fase 2 (OutputExtractor)**: Depende de Fase 1 (usa los nuevos `AnalysisType`) — BLOQUEA Fase 3
- **Fase 3 (Engine dispatch)**: Depende de Fase 2 (usa `OutputExtractor`) + Fase 1
- **Fase 4 (Bundle catalog)**: Depende de Fase 1 (usa nuevos `AnalysisType`); paralela a Fase 2/3
- **Fase 5 (Acabado)**: Depende de todas las fases anteriores

### Dependencias entre Historias de Usuario

- **HU1 (OutputExtractor + Engine)**: Depende de Fase 1 — fundacional para HU2/3/4
- **HU2 (CT_LUNG_NODULE_DETECTION)**: Depende de HU1 + Fase 4 — testeable independientemente
- **HU3 (MRI_BRAIN_TUMOR_SEGMENTATION)**: Depende de HU1 + Fase 4 — testeable independientemente
- **HU4 (XR_BREAST_DENSITY_CLASSIFICATION)**: Depende de HU1 + Fase 4 — testeable independientemente

### Dentro de Cada Historia

- Domain (enum) antes que Infrastructure
- Protocolo antes que implementaciones
- Implementaciones antes que catálogo del engine
- Tests antes que implementación (TDD)

---

## Ejecución Completada

**Fecha**: 2026-07-13
**Estado**: Todas las fases implementadas y verificadas

- Tests: 133 passing (unit), 3 deselected (integration)
- Cobertura: 90.15% (umbral: 85%)
- Lint: OK, sin errores
- Types: OK, sin errores (21 source files)

---

## Notas

- `[P]` = tarea prioritaria para la historia
- La firma de `engine.run()` cambia: añade `analysis_type: AnalysisType` — esto es un
  cambio de API interno (no del contrato de mensajes). `AnalysisService` ya tiene
  `request.analysis_type` disponible para pasarlo.
- `SegmentationExtractor` importa `numpy` y `nibabel` de forma lazy dentro del método
  `extract()` para no bloquear el arranque del worker (los workers que solo clasifican
  no necesitan nibabel al boot).
- `ClassificationExtractor` y `DetectionExtractor` solo usan `json` stdlib — sin imports pesados.
- `test_catalog_covers_all_analysis_types` en `test_bundle_registry.py` fallará
  intencionalmente entre Fase 1 y Fase 4 — es el canario que confirma que Fase 4 completa el trabajo.
- Los cambios NO se commitean automáticamente.
