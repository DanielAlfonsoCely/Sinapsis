# Plan de Implementación: Registry de Bundles MONAI (v0.3.0)

**Fecha**: 2026-07-05
**Especificación**: [spec.md](./spec.md)

## Resumen

Implementar `infrastructure/bundle_registry.py`: un componente de infraestructura que
mapea `AnalysisType → bundle MONAI`, descarga el bundle vía `monai.bundle` a
`BUNDLE_CACHE_DIR` (una sola vez, con caché reutilizable), y devuelve un `ModelRef` del
dominio con nombre, versión y ruta local. Los errores se traducen a la jerarquía del
dominio (`UnknownAnalysisTypeError`, `BundleResolutionError`).

**Enfoque técnico**: clase `BundleRegistry` con un catálogo estático
`AnalysisType → nombre de bundle` y una función de descarga **inyectable** (DIP), cuyo
default hace import perezoso de `monai.bundle.download`. Esto permite unit tests 100%
sin red y sin cargar `monai`/`torch` en memoria durante la suite. La detección de caché
se basa en la presencia de `configs/metadata.json` dentro del directorio del bundle
(criterio verificable frente a descargas parciales); la versión se lee de ese mismo
archivo.

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12 (gestor `uv`, layout `src/`)
**Dependencias Principales**: `monai` + `torch` (NUEVAS en `pyproject.toml` — la
descarga de bundles es `monai.bundle`; torch es dependencia transitiva requerida por
monai y parte del stack fijado en AGENTS.md). `pydantic`/`pydantic-settings` ya
presentes.
**Almacenamiento**: sistema de archivos local — `BUNDLE_CACHE_DIR` (default
`./.bundle_cache`, no versionado en git)
**Testing**: pytest + pytest-cov, cobertura ≥ 85% (`fail_under = 85` en pyproject);
descarga MONAI mockeada en unit tests (sin red)
**Plataforma Objetivo**: servidor Linux (worker RabbitMQ containerizado)
**Tipo de Proyecto**: proyecto único (microservicio, arquitectura por capas simple)
**Objetivos de Rendimiento**: resolución con cache hit sin I/O de red; una descarga por
bundle por volumen de caché
**Restricciones**: Infrastructure solo importa Domain (nunca Application/Presentation);
`monai` confinado a `infrastructure/`; sin PII en logs; sin credenciales hardcodeadas;
mypy `strict`
**Escala/Alcance**: 1 módulo nuevo de infraestructura + extensión retrocompatible de
`ModelRef` + 1 archivo de tests unitarios (~8 tests)

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v0.3.0-bundle-registry/
├── plan.md  # Este archivo
└── spec.md
```

### Código Fuente (raíz de `microservice/`)

```text
pyproject.toml                                # MODIFICAR: deps monai/torch + override mypy
src/
└── sinapsis_ai/
    ├── domain/
    │   └── models.py                         # MODIFICAR: ModelRef + local_path (retrocompat)
    └── infrastructure/
        └── bundle_registry.py                # NUEVO: BundleRegistry (catálogo + descarga + caché)

tests/
└── unit/
    ├── domain/
    │   └── test_models.py                    # MODIFICAR: cobertura de local_path
    ├── presentation/
    │   └── test_schemas.py                   # MODIFICAR: assert local_path NO se serializa
    └── infrastructure/
        ├── __init__.py                       # NUEVO: paquete espejo de la capa
        └── test_bundle_registry.py           # NUEVO: 4 escenarios CHANGELOG + casos límite
```

**Decisión de Estructura**: se sigue el árbol planificado en PROJECT_ARCHITECTURE.md §3:
el módulo vive en `src/sinapsis_ai/infrastructure/` y sus tests espejan la capa en
`tests/unit/infrastructure/` (convención AGENTS.md).

**Decisiones técnicas**:

1. **Deps `monai`/`torch` se agregan en esta versión** (no en v0.4.0): el CHANGELOG
   v0.3.0 exige descarga "vía `monai.bundle`" y el stack está fijado en AGENTS.md. Se
   añade override de mypy (`ignore_missing_imports` para `monai.*`) porque monai no
   distribuye stubs completos compatibles con `strict`.
2. **Función de descarga inyectable** (`download_fn`): el constructor de
   `BundleRegistry` acepta un callable con la firma de descarga; el default importa
   `monai.bundle.download` de forma perezosa (dentro de la función). Los unit tests
   inyectan un fake → sin red, sin cargar torch, cumpliendo CE-003.
3. **Criterio de caché**: bundle cacheado ⇔ existe
   `<BUNDLE_CACHE_DIR>/<bundle_name>/configs/metadata.json` (layout estándar de bundle
   MONAI). Un directorio parcial sin metadata se considera no cacheado → re-descarga
   (caso límite del spec).
4. **`ModelRef.local_path`**: se agrega como tercer campo con default `""` para
   retrocompatibilidad (CE-005); `presentation/schemas.py` no requiere cambios porque
   serializa `name`/`version` explícitamente — se agrega test que lo garantice (RF-004).

---

## Fase 1: Configuración (Infraestructura Compartida)

**Propósito**: Dependencias y estructura de paquetes de test para la capa Infrastructure

- [x] T001 Agregar `monai>=1.3` y `torch>=2.2` a `dependencies` en `pyproject.toml`,
      agregar override `[[tool.mypy.overrides]]` con `module = "monai.*"` e
      `ignore_missing_imports = true`, y ejecutar `uv sync`
- [x] T002 Crear paquete de tests espejo: `tests/unit/infrastructure/__init__.py`

---

## Fase 2: Fundacional (Prerequisitos Bloqueantes)

**Propósito**: Extender `ModelRef` con la ruta local en caché (RF-004) — todas las
historias devuelven o consumen este `ModelRef` extendido

**⚠️ CRÍTICO**: Ningún trabajo de historia de usuario puede comenzar hasta que esta fase
esté completa

- [x] T003 Extender `ModelRef` en `src/sinapsis_ai/domain/models.py` con campo
      `local_path: str = ""` (frozen dataclass, docstring actualizado, sin deps externas)
- [x] T004 Actualizar `tests/unit/domain/test_models.py`: `ModelRef` con y sin
      `local_path` (retrocompatibilidad de construcción posicional/por defecto)
- [x] T005 Agregar test en `tests/unit/presentation/test_schemas.py` que verifique que
      la serialización del resultado incluye solo `name` y `version` del modelo (nunca
      `local_path`) aunque el `ModelRef` lo tenga poblado

**Punto de Control**: Fundación lista — `uv run pytest tests/unit/` verde; la
implementación del registry puede comenzar

---

## Fase 3: Historia de Usuario 1 - Resolver un tipo de análisis a un bundle listo para usar (Prioridad: P1)

**Objetivo**: `BundleRegistry.resolve(analysis_type)` descarga el bundle (si no está
cacheado) y devuelve `ModelRef(name, version, local_path)`; fallos de descarga →
`BundleResolutionError`

**Prueba Independiente**: invocar `resolve()` con `CT_SPLEEN_SEGMENTATION` y un
`download_fn` fake que materializa el layout del bundle → `ModelRef` correcto; con un
fake que lanza → `BundleResolutionError`

### Implementación para Historia de Usuario 1

- [x] T006 [P] [HU1] Crear `src/sinapsis_ai/infrastructure/bundle_registry.py` con:
      catálogo `_BUNDLE_CATALOG: dict[AnalysisType, str]`
      (`CT_SPLEEN_SEGMENTATION → "spleen_ct_segmentation"`), clase `BundleRegistry`
      construida con `cache_dir`, `source` y `download_fn` opcional (default = import
      perezoso de `monai.bundle.download`), y método
      `resolve(analysis_type: AnalysisType) -> ModelRef` (RF-001, RF-002, RF-003, RF-010)
- [x] T007 [HU1] Implementar en `resolve()`: creación de `cache_dir` si no existe
      (RF-008), invocación de `download_fn` con nombre/dir/fuente, lectura de versión
      desde `configs/metadata.json` con fallback a `""` (RF-009), y construcción del
      `ModelRef` (depende de T006)
- [x] T008 [HU1] Implementar traducción de errores: cualquier excepción del
      `download_fn` o de la carga de metadatos → `raise BundleResolutionError(...) from
      exc` (RF-006); logging operativo sin PII: tipo de análisis, bundle, cache
      hit/miss (RF-012) (depende de T007)

### Pruebas para Historia de Usuario 1

- [x] T009 [P] [HU1] Crear `tests/unit/infrastructure/test_bundle_registry.py` con
      fixture de `download_fn` fake (materializa
      `<cache>/<bundle>/configs/metadata.json` con versión) y tests:
      `test_resolve_known_type_returns_modelref` (nombre, versión y ruta local
      esperados, descarga invocada con args correctos),
      `test_download_failure_raises_bundle_resolution_error` (causa encadenada),
      `test_resolve_reads_version_from_metadata` y
      `test_resolve_missing_metadata_version_falls_back_to_empty` (depende de T008)

**Punto de Control**: HU1 funcional — verificación completa AGENTS.md sobre lo
implementado; `resolve()` de tipo conocido testeable de forma independiente

---

## Fase 4: Historia de Usuario 2 - Rechazar tipos de análisis desconocidos (Prioridad: P2)

**Objetivo**: un `AnalysisType` sin mapeo en el catálogo lanza
`UnknownAnalysisTypeError` sin intentar descargas (fail-safe)

**Prueba Independiente**: invocar `resolve()` con un tipo sin entrada en el catálogo →
`UnknownAnalysisTypeError` y `download_fn` nunca invocado

### Implementación para Historia de Usuario 2

- [x] T010 [HU2] Implementar guard en `resolve()`: lookup del catálogo con manejo
      explícito de ausencia → `UnknownAnalysisTypeError` con mensaje que incluya el tipo
      solicitado; el guard se evalúa ANTES de tocar caché o descarga (RF-005; nunca
      `KeyError` crudo) (depende de T006)

### Pruebas para Historia de Usuario 2

- [x] T011 [P] [HU2] Agregar a `tests/unit/infrastructure/test_bundle_registry.py`:
      `test_resolve_unknown_type_raises` (usa un `AnalysisType` de prueba sin mapeo o
      catálogo inyectado/parcheado vacío; asserts: excepción correcta + `download_fn`
      con cero invocaciones) (depende de T010)

**Punto de Control**: HU1 y HU2 funcionan de forma independiente

---

## Fase 5: Historia de Usuario 3 - Reutilizar bundles cacheados (Prioridad: P3)

**Objetivo**: si `<cache>/<bundle>/configs/metadata.json` existe, `resolve()` no
re-descarga; funciona también con caché poblada por un proceso anterior

**Prueba Independiente**: dos `resolve()` consecutivos → una sola descarga; registry
nuevo sobre caché pre-poblada → cero descargas

### Implementación para Historia de Usuario 3

- [x] T012 [HU3] Implementar detección de caché en `resolve()`: si el criterio de
      caché se cumple, saltar `download_fn` y construir el `ModelRef` desde el bundle
      existente (RF-007); si el directorio existe pero sin `metadata.json` (descarga
      parcial), tratar como no cacheado y re-descargar (caso límite del spec)
      (depende de T007)

### Pruebas para Historia de Usuario 3

- [x] T013 [P] [HU3] Agregar a `tests/unit/infrastructure/test_bundle_registry.py`:
      `test_bundle_cached_not_redownloaded` (segunda resolución → `download_fn` una sola
      vez), `test_preexisting_cache_skips_download` (caché materializada manualmente →
      cero descargas) y `test_partial_cache_triggers_redownload` (directorio sin
      metadata → descarga) (depende de T012)

**Punto de Control**: las tres historias funcionan de forma independiente; los 4
escenarios del CHANGELOG v0.3.0 pasan

---

## Fase 6: Acabado y Preocupaciones Transversales

**Propósito**: verificación integral, cobertura y documentación

- [x] T014 Ejecutar suite completa de verificación en orden AGENTS.md:
      `uv run ruff format --check .` → `uv run ruff check .` → `uv run mypy src` →
      `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` (cobertura global
      ≥ 85%, CE-002/CE-004); corregir cualquier hallazgo
- [x] T015 Verificar CE-003: la suite unitaria corre sin red (ningún import efectivo de
      `monai` durante los tests — el import perezoso solo se ejercita en producción)
- [x] T016 Actualizar `docs/CHANGELOG.md`: marcar v0.3.0 según la convención de estado
      del proyecto una vez verificada

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Configuración (Fase 1)**: sin dependencias — puede comenzar de inmediato
- **Fundacional (Fase 2)**: depende de Fase 1 — BLOQUEA todas las historias (todas
  devuelven el `ModelRef` extendido)
- **Historias de Usuario (Fases 3–5)**: dependen de la Fase 2
  - HU2 y HU3 modifican `resolve()` (mismo archivo) → ejecución **secuencial**
    P1 → P2 → P3 para evitar conflictos
- **Acabado (Fase 6)**: depende de que las tres historias estén completas

### Dependencias entre Historias de Usuario

- **HU1 (P1)**: solo depende de Fundacional — crea el módulo y el flujo de descarga
- **HU2 (P2)**: testeable de forma independiente (guard previo a descarga); comparte el
  módulo creado en HU1 (T006)
- **HU3 (P3)**: testeable de forma independiente (solo requiere el flujo de descarga de
  HU1 para el escenario "segunda resolución")

### Dentro de Cada Historia

- Modelos antes que servicios: `ModelRef` (Fase 2) antes que `BundleRegistry`
- Implementación antes que pruebas: T006–T008 → T009; T010 → T011; T012 → T013
- Historia completa (con tests verdes) antes de pasar a la siguiente prioridad

## Notas

- [P] = tarea prioritaria dentro de la historia; [HU#] = trazabilidad a la historia
- Cada historia es completable y testeable de forma independiente
- Verificación por punto de control con los comandos exactos de AGENTS.md, en orden
- `uv sync` tras T001 descargará wheels grandes (torch); es esperado y se cachea en uv
- No se versiona nada bajo `.bundle_cache/` (ya ignorado por `.gitignore` de v0.1.0)
- Commit sugerido por fase o grupo lógico; los commits los decide el desarrollador

---

## Ejecución Completada

**Fecha**: 2026-07-05
**Estado**: Todas las fases implementadas y verificadas

- Tests: 50 passed (unit, sin red, descarga MONAI mockeada)
- Cobertura global: 88.29% (≥ 85%); `bundle_registry.py`: 96%
- Verificación AGENTS.md: ruff format ✓ · ruff check ✓ · mypy src ✓ · pytest ✓

**Desviaciones del plan**:
- T010 (guard tipo desconocido) y T012 (detección de caché) se implementaron junto
  con T006–T008 en una única versión cohesiva de `resolve()` (mismo archivo, evita
  reescrituras); los tests sí se agregaron por historia en sus fases respectivas.
- El import del downloader default usa `monai.bundle.scripts.download` (módulo real)
  en lugar de `monai.bundle.download` porque mypy strict rechaza el re-export
  implícito de monai.
- La lectura de versión es totalmente defensiva (metadata ausente, corrupta o con
  version no-string → `""`); `BundleResolutionError` queda reservado a fallos de
  descarga o descargas que no materializan el bundle.
