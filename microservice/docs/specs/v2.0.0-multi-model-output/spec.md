# Especificación de Funcionalidad: Multi-Model Output Generalizado (v2.0.0)

**Creado**: 2026-07-13

## Contexto

Actualmente el engine asume que todo bundle produce una máscara de segmentación
(`segmentation_mask`) y `metrics={}`. v2.0.0 generaliza la extracción de outputs para
soportar los tres tipos de salida clínicamente relevantes:

| Tipo | Ejemplo clínico | Output |
|------|----------------|--------|
| **Segmentación** | Tumor cerebral en MRI | `artifacts=[segmentation_mask]` + `metrics={volume_ml}` |
| **Clasificación** | Densidad mamaria en mamografía | `artifacts=[]` + `metrics={probability, class_label}` |
| **Detección** | Nódulos pulmonares en CT | `artifacts=[]` + `metrics={lesion_count, boxes:[...]}` |

Se añaden 3 nuevos `AnalysisType` con sus bundles MONAI reales, uno por tipo:
- `CT_LUNG_NODULE_DETECTION` → `lung_nodule_ct_detection` (detección)
- `MRI_BRAIN_TUMOR_SEGMENTATION` → `brats_mri_segmentation` (segmentación)
- `XR_BREAST_DENSITY_CLASSIFICATION` → `breast_density_classification` (clasificación)

El contrato de mensajes (`AnalysisResult`) **no cambia** — solo se enriquecen `artifacts`
y `metrics` según el tipo. El backend Go interpreta `metrics` usando `analysis_type`.

---

## Escenarios de Usuario y Pruebas *(obligatorio)*

### Historia de Usuario 1 - OutputExtractor por tipo de análisis (Prioridad: P1)

El sistema introduce un protocolo `OutputExtractor` con implementaciones concretas por
tipo de salida. El `MonaiInferenceEngine` delega la construcción del `InferenceOutput`
al extractor correspondiente al `AnalysisType`, eliminando el hardcoding de
`_SEGMENTATION_ARTIFACT_TYPE`.

**Por qué esta prioridad**: Es la pieza fundacional que desbloquea todas las historias
siguientes. Sin ella, añadir un nuevo tipo de modelo implica modificar el engine cada vez.

**Prueba Independiente**: Testeable con extractores mockeados y un `output_dir` temporal
con archivos de fixtures. Sin MONAI real ni downloads.

**Escenarios de Aceptación**:

1. **Escenario**: Extractor de segmentación produce máscara + métricas de volumen
   - **Dado** un `output_dir` con un archivo `.nii.gz` de máscara
   - **Cuando** se llama a `SegmentationExtractor().extract(output_dir)`
   - **Entonces** retorna `InferenceOutput(artifact_paths={"segmentation_mask": ...}, metrics={"volume_voxels": N})`

2. **Escenario**: Extractor de clasificación produce probabilidad sin artefacto
   - **Dado** un `output_dir` con un archivo `.json` con scores por clase
   - **Cuando** se llama a `ClassificationExtractor(label="density").extract(output_dir)`
   - **Entonces** retorna `InferenceOutput(artifact_paths={}, metrics={"probability": 0.87, "predicted_class": "dense"})`

3. **Escenario**: Extractor de detección produce conteo y bounding boxes
   - **Dado** un `output_dir` con un archivo `.json` con detecciones
   - **Cuando** se llama a `DetectionExtractor().extract(output_dir)`
   - **Entonces** retorna `InferenceOutput(artifact_paths={}, metrics={"lesion_count": 3, "max_diameter_mm": 12.4})`

4. **Escenario**: Engine usa el extractor correcto según el AnalysisType
   - **Dado** un `MonaiInferenceEngine` con mapa `{AnalysisType → OutputExtractor}`
   - **Cuando** se llama a `engine.run(model_ref_for_classification, image_path)`
   - **Entonces** se invoca `ClassificationExtractor`, no `SegmentationExtractor`

5. **Escenario**: Sin output_dir con archivos → InferenceError
   - **Dado** un `output_dir` vacío
   - **Cuando** cualquier extractor intenta procesar
   - **Entonces** lanza `InferenceError`

---

### Historia de Usuario 2 - Nuevo AnalysisType: detección de nódulos pulmonares (Prioridad: P1)

`CT_LUNG_NODULE_DETECTION` con bundle `lung_nodule_ct_detection`. Produce métricas de
conteo (`lesion_count`) y diámetro máximo (`max_diameter_mm`) a partir del JSON de
detecciones del bundle. Sin artefacto de imagen.

**Por qué esta prioridad**: Es el tipo de output más diferente al actual (detección vs
segmentación) — valida que la abstracción es suficientemente general.

**Prueba Independiente**: Testeable con `BundleRegistry` y `DetectionExtractor` mockeados.
Un fixture JSON de detecciones es suficiente.

**Escenarios de Aceptación**:

1. **Escenario**: Request de detección produce resultado con métricas
   - **Dado** un `AnalysisRequest` con `analysis_type=ct_lung_nodule_detection`
   - **Cuando** el pipeline completa con un JSON de detecciones mockeado
   - **Entonces** `AnalysisResult.metrics` contiene `lesion_count` y `max_diameter_mm`
     y `AnalysisResult.artifacts` está vacío

2. **Escenario**: `ct_lung_nodule_detection` está en el catálogo del bundle registry
   - **Dado** un `BundleRegistry`
   - **Cuando** se llama a `resolve(AnalysisType.CT_LUNG_NODULE_DETECTION)`
   - **Entonces** retorna un `ModelRef` con `name="lung_nodule_ct_detection"`

---

### Historia de Usuario 3 - Nuevo AnalysisType: segmentación de tumor cerebral (Prioridad: P2)

`MRI_BRAIN_TUMOR_SEGMENTATION` con bundle `brats_mri_segmentation`. Produce una máscara
de segmentación multi-label (tumor core, edema, enhancing) como artefacto, más métricas
de volumen por región.

**Por qué esta prioridad**: Extiende el tipo de output ya existente (segmentación) pero
con múltiples regiones — valida que `SegmentationExtractor` puede generalizar.

**Prueba Independiente**: Testeable con `output_dir` mockeado conteniendo un `.nii.gz`.

**Escenarios de Aceptación**:

1. **Escenario**: Request de segmentación cerebral produce máscara y volúmenes
   - **Dado** un `AnalysisRequest` con `analysis_type=mri_brain_tumor_segmentation`
   - **Cuando** el pipeline completa con un archivo `.nii.gz` mockeado
   - **Entonces** `AnalysisResult.artifacts` contiene `segmentation_mask` y
     `AnalysisResult.metrics` contiene `volume_voxels`

---

### Historia de Usuario 4 - Nuevo AnalysisType: clasificación de densidad mamaria (Prioridad: P2)

`XR_BREAST_DENSITY_CLASSIFICATION` con bundle `breast_density_classification`. Produce
probabilidades por clase (A/B/C/D según BI-RADS) y la clase predicha. Sin artefacto.

**Por qué esta prioridad**: Valida el tipo de output de clasificación de extremo a extremo.

**Prueba Independiente**: Testeable con fixture JSON de scores por clase.

**Escenarios de Aceptación**:

1. **Escenario**: Request de clasificación produce probabilidades y clase predicha
   - **Dado** un `AnalysisRequest` con `analysis_type=xr_breast_density_classification`
   - **Cuando** el pipeline completa con un JSON de scores mockeado
   - **Entonces** `AnalysisResult.metrics` contiene `probability` (float 0-1),
     `predicted_class` (str) y `artifacts` está vacío

---

### Casos Límite

- ¿Qué ocurre si el `output_dir` contiene archivos de un tipo inesperado (e.g. `.txt`)?
  → El extractor lanza `InferenceError` con mensaje descriptivo.
- ¿Qué ocurre si el JSON de clasificación tiene formato incorrecto?
  → `InferenceError` con causa explícita.
- ¿Qué ocurre si un nuevo `AnalysisType` no tiene extractor registrado?
  → `InferenceError` al intentar extraer el output (fallo explícito, no silencioso).
- ¿Pueden coexistir artefacto y métricas? → Sí: la segmentación cerebral tiene ambos.
- ¿El volumen en vóxeles es suficiente sin voxel spacing? → Sí para v2.0.0;
  el volumen en mm³ requiere metadata del NIfTI (extensión futura).
- ¿El bundle `breast_density_classification` escribe JSON o tensor? → El extractor
  maneja ambos formatos buscando primero `.json`, luego `.pt` (tensores de scores).

---

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El sistema DEBE definir un protocolo `OutputExtractor` con método
  `extract(output_dir: Path) -> InferenceOutput` en `infrastructure/output_extractors.py`.
- **RF-002**: El sistema DEBE implementar `SegmentationExtractor`: busca el primer
  `.nii.gz` en `output_dir`, lo asigna como `segmentation_mask` y computa `volume_voxels`
  (conteo de vóxeles no-cero).
- **RF-003**: El sistema DEBE implementar `ClassificationExtractor`: lee el archivo de
  scores del bundle (`.json`), extrae la clase con mayor probabilidad y su score.
- **RF-004**: El sistema DEBE implementar `DetectionExtractor`: lee el JSON de
  detecciones del bundle, extrae `lesion_count` y `max_diameter_mm` (si disponible).
- **RF-005**: `MonaiInferenceEngine` DEBE recibir un mapa
  `dict[AnalysisType, OutputExtractor]` inyectado (DIP); si no se inyecta, usa el mapa
  por defecto definido en el módulo.
- **RF-006**: `domain/models.py` DEBE añadir los tres nuevos miembros a `AnalysisType`:
  `CT_LUNG_NODULE_DETECTION`, `MRI_BRAIN_TUMOR_SEGMENTATION`,
  `XR_BREAST_DENSITY_CLASSIFICATION`.
- **RF-007**: `infrastructure/bundle_registry.py` DEBE registrar los tres nuevos bundles
  en `_BUNDLE_CATALOG`.
- **RF-008**: Los nuevos `AnalysisType` DEBEN ser reconocidos por `presentation/schemas.py`
  automáticamente (sin cambios, ya itera `AnalysisType` dinámicamente).
- **RF-009**: El contrato de mensajes `AnalysisResult` NO DEBE cambiar; solo se enriquecen
  `artifacts` y `metrics` según el tipo.
- **RF-010**: La cobertura global DEBE mantenerse ≥ 85% tras añadir los nuevos módulos.

### Entidades Clave

- **`OutputExtractor`** (`infrastructure/output_extractors.py`): protocolo que abstrae
  la extracción de `InferenceOutput` a partir del directorio de salida del bundle.
- **`SegmentationExtractor`**: implementación para bundles que producen máscaras NIfTI.
- **`ClassificationExtractor`**: implementación para bundles que producen scores por clase.
- **`DetectionExtractor`**: implementación para bundles que producen bounding boxes / detecciones.
- **`_EXTRACTOR_CATALOG`** (`inference_engine.py`): mapa `AnalysisType → OutputExtractor`
  que es la fuente de verdad de qué extractor usar por tipo de análisis.

---

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: `uv run pytest` pasa al 100% con los nuevos tests de extractores y tipos.
- **CE-002**: `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` reporta ≥ 85%.
- **CE-003**: `uv run mypy src` pasa sin errores en los nuevos módulos.
- **CE-004**: `uv run ruff format --check . && uv run ruff check .` sin hallazgos.
- **CE-005**: Un request con `analysis_type=ct_lung_nodule_detection` produce
  `metrics.lesion_count` en el `AnalysisResult` (verificable end-to-end con mocks).
- **CE-006**: Un request con `analysis_type=xr_breast_density_classification` produce
  `metrics.probability` y `metrics.predicted_class` con `artifacts=[]`.
- **CE-007**: Un request con `analysis_type=mri_brain_tumor_segmentation` produce
  `artifacts=[{type: segmentation_mask, ...}]` y `metrics.volume_voxels`.
- **CE-008**: Añadir un cuarto tipo en el futuro requiere solo: (1) miembro en
  `AnalysisType`, (2) entrada en `_BUNDLE_CATALOG`, (3) entrada en `_EXTRACTOR_CATALOG`.
  Ningún otro archivo debe modificarse.
