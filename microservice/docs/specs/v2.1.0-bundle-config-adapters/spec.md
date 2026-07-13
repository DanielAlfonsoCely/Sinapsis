# Especificación de Funcionalidad: Bundle Config Adapters (v2.1.0)

**Creado**: 2026-07-13

## Contexto y problema

El `_monai_workflow_factory` actual aplica los mismos overrides a todos los bundles:

```python
overrides = {
    "datalist":                   [image_path],   # no todos usan esta clave
    "output_dir":                 output_dir,      # sí universal
    "device":                     device,          # sí universal
    "checkpointloader#map_location": device,       # no todos lo tienen
}
```

Esto causa tres fallos en producción (diagnosticados en v2.0.0):

| Bundle | Causa del fallo |
|--------|----------------|
| `lung_nodule_ct_detection` | Requiere `torchvision` (dependencia ausente) + datalist diferente (`data_list_file_path`) |
| `brats_mri_segmentation` | Espera `data_list_file_path` + `dataset_dir`, no `datalist=[imagen]` |
| `breast_density_classification` | No tiene `checkpointloader` → KeyError al aplicar el override |

**Solución**: cada bundle necesita un **adaptador de configuración** (`BundleConfigAdapter`)
que sepa exactamente qué overrides inyectar para ese bundle específico, incluyendo cómo
representar la imagen de entrada en el formato que ese bundle espera.

---

## Escenarios de Usuario y Pruebas *(obligatorio)*

### Historia de Usuario 1 - Protocolo BundleConfigAdapter (Prioridad: P1)

El sistema introduce un protocolo `BundleConfigAdapter` que encapsula los overrides
específicos por bundle. El `_monai_workflow_factory` consulta un catálogo de adaptadores
en lugar de hardcodear overrides universales. Los adaptadores son inyectables para testing.

**Por qué esta prioridad**: Es la pieza fundacional. Sin ella los demás bundles siguen fallando.

**Prueba Independiente**: Testeable con un adaptador stub y un workflow factory mock.
No requiere MONAI ni downloads.

**Escenarios de Aceptación**:

1. **Escenario**: Factory usa el adaptador correcto para el bundle
   - **Dado** un catálogo con dos adaptadores distintos y un workflow factory spy
   - **Cuando** se ejecuta el engine con `analysis_type=ct_spleen_segmentation`
   - **Entonces** se llama al adaptador de spleen para construir los overrides, no al de otro bundle

2. **Escenario**: Bundle sin adaptador → InferenceError descriptivo
   - **Dado** un catálogo de adaptadores vacío
   - **Cuando** se intenta ejecutar cualquier bundle
   - **Entonces** se lanza `InferenceError` con mensaje que indica qué bundle no tiene adaptador

3. **Escenario**: Adaptador sin `checkpointloader` no lo inyecta
   - **Dado** un adaptador que no incluye `checkpointloader#map_location` en sus overrides
   - **Cuando** se construye el workflow
   - **Entonces** el override `checkpointloader#map_location` no aparece en los kwargs del factory

---

### Historia de Usuario 2 - Adaptador para spleen_ct_segmentation (Prioridad: P1)

`SpleenSegmentationAdapter` replica el comportamiento actual (que ya funciona), usando
`datalist` como lista de rutas y `checkpointloader#map_location`. Valida que el refactor
no rompe el bundle que ya funciona.

**Por qué esta prioridad**: Garantiza que el bundle en producción siga funcionando tras el refactor.

**Prueba Independiente**: El smoke test de integración existente (`test_inference_smoke.py`)
pasa sin cambios.

**Escenarios de Aceptación**:

1. **Escenario**: Spleen sigue funcionando tras el refactor
   - **Dado** el adaptador de spleen y la imagen fixture `tiny_volume.nii.gz`
   - **Cuando** se ejecuta el engine con `CT_SPLEEN_SEGMENTATION`
   - **Entonces** el smoke test de integración pasa como en v2.0.0

---

### Historia de Usuario 3 - Adaptador para brats_mri_segmentation (Prioridad: P1)

`BratsMriAdapter` construye un `datalist.json` temporal en formato Decathlon con la
imagen de entrada como entrada de testing, y pasa `data_list_file_path` + `dataset_dir`
en los overrides. La imagen de entrada (CT abdominal) producirá una segmentación vacía
o errónea en producción real, pero el pipeline completa sin error.

**Por qué esta prioridad**: Es uno de los 3 bundles rotos. Los datos de test usarán la
fixture `tiny_volume.nii.gz` que es CT, no MRI — suficiente para verificar que el
pipeline no lanza excepciones.

**Prueba Independiente**: Testeable con un `output_dir` temporal y un workflow mock.

**Escenarios de Aceptación**:

1. **Escenario**: Adapter crea datalist.json temporal con la imagen de entrada
   - **Dado** `image_path=/tmp/scan.nii.gz` y `bundle_path=/cache/brats_mri_segmentation`
   - **Cuando** se llama a `BratsMriAdapter().build_overrides(image_path, output_dir, bundle_path, device)`
   - **Entonces** los overrides incluyen `data_list_file_path` apuntando a un JSON temporal
     con `{"testing": [{"image": "/tmp/scan.nii.gz"}]}` y `dataset_dir` como directorio padre

2. **Escenario**: Pipeline BraTS completa sin KeyError
   - **Dado** el adaptador y un workflow factory mockeado
   - **Cuando** se ejecuta el engine con `MRI_BRAIN_TUMOR_SEGMENTATION`
   - **Entonces** no se lanza ninguna excepción de KeyError o datalist

---

### Historia de Usuario 4 - Adaptador para breast_density_classification (Prioridad: P1)

`BreastDensityAdapter` omite el override `checkpointloader#map_location` (que no existe
en este bundle) e inyecta la imagen usando el mecanismo propio del bundle
(`data` con `filename` que apunta a un JSON temporal de samples).

**Por qué esta prioridad**: Tercer bundle roto. La resolución del KeyError es simple pero requiere
el protocolo adaptador.

**Escenarios de Aceptación**:

1. **Escenario**: Adapter no inyecta checkpointloader
   - **Dado** `BreastDensityAdapter`
   - **Cuando** se llama a `build_overrides(...)`
   - **Entonces** el dict de overrides NO contiene la clave `checkpointloader#map_location`

2. **Escenario**: Pipeline breast density completa sin KeyError
   - **Dado** el adaptador y un workflow factory mockeado
   - **Cuando** se ejecuta el engine con `XR_BREAST_DENSITY_CLASSIFICATION`
   - **Entonces** no se lanza `KeyError`

---

### Historia de Usuario 5 - Adaptador para lung_nodule_ct_detection + torchvision (Prioridad: P2)

`LungNoduleAdapter` construye los overrides para el bundle de detección de nódulos
(`data_list_file_path` con JSON temporal en formato LUNA16). Adicionalmente se añade
`torchvision` a las dependencias del proyecto.

**Por qué esta prioridad**: Requiere añadir una dependencia nueva (`torchvision`), lo que
tiene más riesgo que los adaptadores de configuración puros. Se trata después de validar
que el mecanismo de adaptadores funciona con los otros tres.

**Escenarios de Aceptación**:

1. **Escenario**: `torchvision` importable tras `uv sync`
   - **Dado** `torchvision` añadido a `pyproject.toml`
   - **Cuando** se ejecuta `uv sync && python -c "import torchvision"`
   - **Entonces** no hay `ModuleNotFoundError`

2. **Escenario**: Adapter construye datalist LUNA16 temporal
   - **Dado** `image_path=/tmp/scan.nii.gz`
   - **Cuando** se llama a `LungNoduleAdapter().build_overrides(...)`
   - **Entonces** los overrides incluyen `data_list_file_path` apuntando a un JSON con
     el formato esperado por el bundle LUNA16

---

### Casos Límite

- ¿Qué ocurre si el directorio temporal del datalist no puede crearse? → `InferenceError`.
- ¿El JSON temporal se limpia tras la inferencia? → No en v2.1.0; vive en `output_dir`
  que es temporal por proceso.
- ¿Qué pasa si un bundle cambia su schema de config en una nueva versión? → El adaptador
  para esa versión falla en producción; el operador actualiza el adaptador. El error es
  `InferenceError` claro, no un crash opaco.
- ¿Los adaptadores son stateless? → Sí, no guardan estado entre llamadas.
- ¿Qué imagen se usa para breast density (que espera PNG/JPG, no NIfTI)? → El pipeline
  puede fallar en el pre-proceso con un error de formato; el adaptador no valida el tipo
  de imagen — esa validación la haría el bundle. En v2.1.0 se verifica que no hay KeyError;
  el error de tipo de imagen se maneja como `InferenceError` normal.

---

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El sistema DEBE definir un protocolo `BundleConfigAdapter` con método
  `build_overrides(image_path, output_dir, bundle_path, device) -> dict[str, Any]`
  en `infrastructure/bundle_adapters.py`.
- **RF-002**: El `_monai_workflow_factory` DEBE recibir un `adapter_catalog:
  dict[AnalysisType, BundleConfigAdapter]` inyectable (DIP) en lugar de hardcodear overrides.
- **RF-003**: El sistema DEBE implementar `SpleenSegmentationAdapter` que reproduce el
  comportamiento actual: `datalist=[image_path]`, `checkpointloader#map_location=device`.
- **RF-004**: El sistema DEBE implementar `BratsMriAdapter` que genera un `datalist.json`
  temporal en formato Decathlon y pasa `data_list_file_path` + `dataset_dir`.
- **RF-005**: El sistema DEBE implementar `BreastDensityAdapter` que omite
  `checkpointloader#map_location` y provee la imagen mediante el mecanismo del bundle.
- **RF-006**: El sistema DEBE implementar `LungNoduleAdapter` que genera un JSON temporal
  en formato LUNA16 y lo pasa como `data_list_file_path`.
- **RF-007**: `torchvision` DEBE añadirse a `pyproject.toml` como dependencia de runtime.
- **RF-008**: El catálogo de adaptadores por defecto `_ADAPTER_CATALOG` DEBE cubrir todos
  los `AnalysisType` declarados — verificado por un test que itera el enum.
- **RF-009**: Los overrides `output_dir` y `device` DEBEN seguir siendo universales
  (aplicados por el factory antes de los overrides del adaptador, con posibilidad de
  que el adaptador los sobreescriba si es necesario).
- **RF-010**: La cobertura DEBE mantenerse ≥ 85%.

### Entidades Clave

- **`BundleConfigAdapter`** (`infrastructure/bundle_adapters.py`): protocolo que
  construye el dict de overrides específico para un bundle. Stateless.
- **`_ADAPTER_CATALOG`** (`inference_engine.py`): mapa `AnalysisType → BundleConfigAdapter`,
  paralelo al `_EXTRACTOR_CATALOG` existente.

---

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: `uv run pytest` pasa al 100% incluyendo nuevos tests de adaptadores.
- **CE-002**: `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` reporta ≥ 85%.
- **CE-003**: `uv run mypy src` pasa sin errores.
- **CE-004**: `uv run ruff format --check . && uv run ruff check .` sin hallazgos.
- **CE-005**: `uv run python tmp/send_requests.py` + `uv run python tmp/wait_results.py`
  muestra los 4 resultados con `status=succeeded` (o `failed` solo por tipo de imagen
  incorrecto, no por KeyError/ImportError).
- **CE-006**: El smoke test de integración existente (`test_inference_smoke.py`) sigue
  pasando con `CT_SPLEEN_SEGMENTATION`.
- **CE-007**: Añadir un quinto bundle en el futuro requiere solo: (1) `AnalysisType`,
  (2) `_BUNDLE_CATALOG`, (3) `_EXTRACTOR_CATALOG`, (4) `_ADAPTER_CATALOG` — 4 líneas
  en 3 archivos.
