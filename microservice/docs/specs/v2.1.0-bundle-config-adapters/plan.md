# Plan de Implementación: Bundle Config Adapters (v2.1.0)

**Fecha**: 2026-07-13
**Especificación**: [spec.md](./spec.md)

---

## Resumen

Introducir el protocolo `BundleConfigAdapter` en `infrastructure/bundle_adapters.py`
con cuatro implementaciones concretas (una por bundle), y refactorizar
`_monai_workflow_factory` para que consulte un `_ADAPTER_CATALOG` inyectable en lugar
de aplicar overrides universales. Añadir `torchvision` como dependencia de runtime.

---

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12
**Dependencias nuevas**: `torchvision` (runtime, requerida por `lung_nodule_ct_detection`)
**Testing**: `pytest` + `pytest-cov`; umbral ≥ 85%; adaptadores testeados con
`tmp_path` y mocks de workflow factory; sin MONAI real en unit tests
**Restricciones**: `application/` no cambia; `domain/` no cambia; `bundle_adapters.py`
solo en `infrastructure/`; el catálogo por defecto `_ADAPTER_CATALOG` vive en
`inference_engine.py`

### Formatos de datalist por bundle (investigados en diagnóstico)

| Bundle | Override de imagen | Formato JSON |
|--------|-------------------|--------------|
| `spleen_ct_segmentation` | `datalist=[image_path]` | lista directa |
| `brats_mri_segmentation` | `data_list_file_path=<tmp.json>` + `dataset_dir=<dir>` | `{"testing": [{"image": "..."}]}` |
| `lung_nodule_ct_detection` | `data_list_file_path=<tmp.json>` + `dataset_dir=<dir>` | `{"validation": [{"image": "..."}]}` (formato Decathlon) |
| `breast_density_classification` | `data={"_target_": ..., "filename": "<tmp.json>"}` | `{"Test": [{"image": "...", "label": 0}]}` |

---

## Estructura del Proyecto

### Documentación

```text
docs/specs/v2.1.0-bundle-config-adapters/
├── plan.md   # Este archivo
└── spec.md
```

### Código Fuente

```text
src/sinapsis_ai/
└── infrastructure/
    ├── bundle_adapters.py          # NUEVO: BundleConfigAdapter protocol + 4 impl
    └── inference_engine.py         # Modificar: _ADAPTER_CATALOG + factory refactorizado

tests/
└── unit/
    └── infrastructure/
        ├── test_bundle_adapters.py # NUEVO: tests de los 4 adaptadores
        └── test_inference_engine.py # Ampliar: tests del factory con catálogo de adaptadores

pyproject.toml                      # Añadir torchvision>=0.17
```

---

## Fase 1: Dependencia torchvision

**Propósito**: Resolver el `ModuleNotFoundError` de `lung_nodule_ct_detection` antes
de implementar los adaptadores. Bloquea HU5.

- [ ] T001 Modificar `pyproject.toml`: añadir `torchvision>=0.17` a `dependencies`
- [ ] T002 Ejecutar `uv sync` para instalar y verificar `import torchvision`

**Punto de Control**: `uv run python -c "import torchvision; print(torchvision.__version__)"` pasa

---

## Fase 2: Fundacional — Protocolo BundleConfigAdapter (HU1)

**Propósito**: Definir el protocolo y refactorizar el factory. Todas las HU de adaptadores
concretos dependen de esta fase.

**CRITICO**: Ningún adaptador concreto puede implementarse hasta que el protocolo exista
y el factory lo consuma.

### Pruebas fundacionales

- [ ] T003 [P] [HU1] Escribir `tests/unit/infrastructure/test_bundle_adapters.py`
  con estructura de fixtures y helpers (sin tests de adaptadores concretos aún):
  - Helper `_make_engine_with_adapter(adapter, workflow_spy)` para reutilizar en HU2-5
  - `test_engine_uses_adapter_overrides_not_hardcoded_defaults` — verifica que los
    kwargs del factory vienen del adaptador, no de overrides hardcodeados

- [ ] T004 [P] [HU1] Ampliar `tests/unit/infrastructure/test_inference_engine.py`:
  - `test_adapter_catalog_covers_all_analysis_types` — verifica que `_ADAPTER_CATALOG`
    cubre todos los `AnalysisType` (análogo al test de extractores)
  - `test_missing_adapter_raises_inference_error` — catálogo vacío → `InferenceError`
    con mensaje descriptivo

### Implementación fundacional

- [ ] T005 [P] [HU1] Crear `src/sinapsis_ai/infrastructure/bundle_adapters.py`:
  ```python
  class BundleConfigAdapter(Protocol):
      def build_overrides(
          self,
          image_path: str,
          output_dir: str,
          bundle_path: str,
          device: str,
      ) -> dict[str, Any]: ...
  ```

- [ ] T006 [P] [HU1] Refactorizar `src/sinapsis_ai/infrastructure/inference_engine.py`:
  - Añadir `_ADAPTER_CATALOG: dict[AnalysisType, BundleConfigAdapter]` (inicialmente vacío
    hasta que se implanten los adaptadores en Fases 3-6)
  - Modificar `_monai_workflow_factory` para recibir `adapter: BundleConfigAdapter` y
    usar `adapter.build_overrides(image_path, output_dir, bundle_path, device)` como overrides
  - El engine busca el adaptador en `_ADAPTER_CATALOG` antes de llamar al factory;
    si no existe → `InferenceError`
  - `output_dir` y `device` siguen siendo construidos por el engine (no por el adaptador)
    pero el adaptador puede sobreescribirlos si necesita (merge: adapter_overrides toman precedencia)

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_inference_engine.py -q` pasa

---

## Fase 3: HU2 — SpleenSegmentationAdapter (P1)

**Objetivo**: Adaptador que replica el comportamiento actual de spleen. Valida el refactor.

### Tests HU2

- [ ] T007 [P] [HU2] En `test_bundle_adapters.py`:
  - `test_spleen_adapter_includes_datalist` — overrides contienen `datalist=[image_path]`
  - `test_spleen_adapter_includes_checkpoint_map_location` — contiene `checkpointloader#map_location`
  - `test_spleen_adapter_registered_in_catalog` — `CT_SPLEEN_SEGMENTATION` en `_ADAPTER_CATALOG`

### Implementación HU2

- [ ] T008 [HU2] En `bundle_adapters.py`: implementar `SpleenSegmentationAdapter`:
  ```python
  class SpleenSegmentationAdapter:
      def build_overrides(self, image_path, output_dir, bundle_path, device):
          return {
              "datalist": [image_path],
              "checkpointloader#map_location": device,
          }
  ```
- [ ] T009 [HU2] En `inference_engine.py`: añadir entrada en `_ADAPTER_CATALOG`:
  `AnalysisType.CT_SPLEEN_SEGMENTATION: SpleenSegmentationAdapter()`

**Punto de Control**: smoke test de integración existente pasa: `uv run pytest -m integration tests/integration/test_inference_smoke.py -q`

---

## Fase 4: HU3 — BratsMriAdapter (P1)

**Objetivo**: Crea `datalist.json` temporal en formato Decathlon, pasa `data_list_file_path`.

### Tests HU3

- [ ] T010 [P] [HU3] En `test_bundle_adapters.py`:
  - `test_brats_adapter_creates_decathlon_datalist_json` — overrides contienen
    `data_list_file_path` apuntando a un JSON con `{"testing": [{"image": image_path}]}`
  - `test_brats_adapter_sets_dataset_dir_to_image_parent` — `dataset_dir` es el directorio
    padre de `image_path`
  - `test_brats_adapter_does_not_include_checkpointloader` — no hay esa clave
  - `test_brats_adapter_registered_in_catalog`

### Implementación HU3

- [ ] T011 [HU3] En `bundle_adapters.py`: implementar `BratsMriAdapter`:
  - Escribe un JSON temporal en `output_dir` con `{"testing": [{"image": image_path}]}`
  - Overrides: `data_list_file_path=<json_path>`, `dataset_dir=<parent of image_path>`,
    `checkpointloader#map_location=device`
- [ ] T012 [HU3] En `inference_engine.py`: añadir entrada en `_ADAPTER_CATALOG`:
  `AnalysisType.MRI_BRAIN_TUMOR_SEGMENTATION: BratsMriAdapter()`

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_bundle_adapters.py -k brats -q` pasa

---

## Fase 5: HU4 — BreastDensityAdapter (P1)

**Objetivo**: Omite `checkpointloader#map_location`, provee imagen via JSON propio del bundle.

### Tests HU5

- [ ] T013 [P] [HU4] En `test_bundle_adapters.py`:
  - `test_breast_density_adapter_no_checkpointloader` — overrides NO contienen `checkpointloader#map_location`
  - `test_breast_density_adapter_creates_sample_json` — overrides contienen `data` con
    `filename` apuntando a JSON con `{"Test": [{"image": image_path, "label": 0}]}`
  - `test_breast_density_adapter_registered_in_catalog`

### Implementación HU5

- [ ] T014 [HU4] En `bundle_adapters.py`: implementar `BreastDensityAdapter`:
  - Escribe JSON temporal con `{"Test": [{"image": image_path, "label": 0}]}`
  - Overrides: `data={"_target_": "scripts.createList.CreateImageLabelList", "filename": <json_path>}`
  - Sin `checkpointloader#map_location`
- [ ] T015 [HU4] En `inference_engine.py`: añadir entrada en `_ADAPTER_CATALOG`:
  `AnalysisType.XR_BREAST_DENSITY_CLASSIFICATION: BreastDensityAdapter()`

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_bundle_adapters.py -k breast -q` pasa

---

## Fase 6: HU5 — LungNoduleAdapter (P2)

**Objetivo**: Crea datalist en formato LUNA16 (`validation` key), pasa `data_list_file_path`.

### Tests HU5

- [ ] T016 [P] [HU5] En `test_bundle_adapters.py`:
  - `test_lung_nodule_adapter_creates_luna16_datalist` — JSON con
    `{"validation": [{"image": image_path}]}`
  - `test_lung_nodule_adapter_sets_dataset_dir`
  - `test_lung_nodule_adapter_registered_in_catalog`

### Implementación HU5

- [ ] T017 [HU5] En `bundle_adapters.py`: implementar `LungNoduleAdapter`:
  - Escribe JSON temporal con `{"validation": [{"image": image_path}]}`
  - Overrides: `data_list_file_path=<json_path>`, `dataset_dir=<parent of image_path>`,
    `checkpointloader#map_location=device`
- [ ] T018 [HU5] En `inference_engine.py`: añadir entrada en `_ADAPTER_CATALOG`:
  `AnalysisType.CT_LUNG_NODULE_DETECTION: LungNoduleAdapter()`

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_bundle_adapters.py -k lung -q` pasa

---

## Fase 7: Acabado y verificación final

- [ ] T019 Ejecutar `uv run ruff format --check .` y corregir
- [ ] T020 Ejecutar `uv run ruff check .` y corregir
- [ ] T021 Ejecutar `uv run mypy src` y corregir
- [ ] T022 Ejecutar `uv run python -m pytest` y verificar 100% passing
- [ ] T023 Ejecutar `uv run python -m pytest --cov=sinapsis_ai --cov-report=term-missing`
  y verificar cobertura ≥ 85%
- [ ] T024 Actualizar `docs/CHANGELOG.md`: añadir `[v2.1.0]` como `COMPLETADO`

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Fase 1 (torchvision)**: Sin dependencias
- **Fase 2 (Fundacional)**: Depende de Fase 1 — BLOQUEA Fases 3-6
- **Fases 3-6 (Adaptadores concretos)**: Dependen de Fase 2; secuenciales por prioridad
- **Fase 7 (Acabado)**: Depende de todas las anteriores

### Dependencias entre Historias

- **HU1**: Fundacional — bloquea todo
- **HU2, HU3, HU4**: Dependen de HU1; independientes entre sí
- **HU5**: Depende de HU1 + Fase 1 (torchvision)

---

## Notas

- El JSON temporal de datalist se escribe dentro de `output_dir` (que ya es temporal
  por proceso) — no hay archivos huérfanos.
- `output_dir` y `device` se siguen construyendo en el engine antes de llamar al adaptador;
  el adaptador puede incluirlos en sus overrides si necesita valores diferentes.
- Los adaptadores son **stateless** — pueden ser singletons en `_ADAPTER_CATALOG`.
- El test `test_adapter_catalog_covers_all_analysis_types` funciona como segundo canario
  (junto al de extractores) para detectar tipos sin adaptador.
- Los cambios NO se commitean automáticamente.
