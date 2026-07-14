# Plan de Implementación: Scaffolding v0.1.0

**Fecha**: 2026-07-05
**Especificación**: [spec.md](./spec.md)

## Resumen

Establecer el scaffolding completo del microservicio `sinapsis-ai`: gestión de proyecto con
`uv`, tooling (ruff, mypy, pytest), configuración tipada con pydantic-settings, estructura
de capas y CI. Sin lógica de inferencia. Al finalizar, la secuencia
`uv sync → ruff format --check → ruff check → mypy src → pytest` termina con código 0.

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12 (`.python-version`)
**Dependencias Principales**: `pydantic-settings`, `pydantic`; dev: `ruff`, `mypy`, `pytest`, `pytest-cov`
**Almacenamiento**: N/A (scaffolding; solo sistema de archivos para caché de bundles)
**Testing**: `pytest` + `pytest-cov`; umbral objetivo ≥ 85% (no aplicable en v0.1.0, solo config)
**Plataforma Objetivo**: servidor Linux (contenedor)
**Objetivos de Rendimiento**: N/A (scaffolding)
**Restricciones**: gestor `uv` exclusivamente (sin pip/poetry/venv manual); sin credenciales hardcodeadas
**Escala/Alcance**: microservicio único stateless; v0.1.0 = andamiaje sin lógica de negocio

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v0.1.0-scaffolding/
├── plan.md   # Este archivo
└── spec.md
```

### Código Fuente

```text
src/
└── sinapsis_ai/
    ├── __init__.py
    ├── main.py                    # esqueleto: arranca y sale limpio
    ├── config.py                  # Settings (pydantic-settings)
    ├── presentation/
    │   └── __init__.py
    ├── application/
    │   └── __init__.py
    ├── domain/
    │   └── __init__.py
    └── infrastructure/
        ├── __init__.py
        └── logging.py             # setup de logging estructurado

tests/
├── __init__.py
├── conftest.py                    # fixture de Settings de prueba
└── unit/
    └── test_config.py             # 3 tests de Settings

.github/
└── workflows/
    └── ci.yml                     # format → lint → types → pytest

pyproject.toml                     # metadatos, deps, ruff, mypy, pytest markers
uv.lock                            # lockfile (generado por uv sync)
.python-version                    # 3.12
.env.example                       # plantilla de variables de entorno
.gitignore                         # excluye .env, .bundle_cache/, artefactos
```

**Decisión de Estructura**: Layout `src/` estándar con paquete importable `sinapsis_ai`.
Cuatro subpaquetes de capas vacíos (solo `__init__.py`) para dar andamiaje a v0.2.0+.
`logging.py` incluido en v0.1.0 porque es infraestructura transversal usada desde `main.py`.

---

## Fase 1: Configuración del proyecto (Infraestructura Compartida)

**Propósito**: Crear los archivos raíz de gestión del proyecto y tooling.

- [x] T001 [P] Crear `pyproject.toml` con: nombre `sinapsis-ai`, versión `0.1.0`, Python
  `>=3.12`, layout `src/`, dependencias (`pydantic`, `pydantic-settings`), dependencias dev
  (`ruff`, `mypy`, `pytest`, `pytest-cov`), sección `[tool.ruff]` (target `py312`, línea
  max 88), sección `[tool.mypy]` (strict, `explicit_package_bases = true`), sección
  `[tool.pytest.ini_options]` con markers `integration`.
- [x] T002 [P] Crear `.python-version` con contenido `3.12`.
- [x] T003 Crear `.env.example` con todas las variables documentadas en §8 de
  `PROJECT_ARCHITECTURE.md` (sin valores sensibles reales).
- [x] T004 Crear `.gitignore` que excluya: `.env`, `.bundle_cache/`, `__pycache__/`,
  `*.pyc`, `.mypy_cache/`, `.pytest_cache/`, `dist/`, `*.egg-info/`.
- [x] T005 Crear `.github/workflows/ci.yml` con job que ejecute en `ubuntu-latest`:
  instala `uv`, corre `uv sync`, luego `ruff format --check .`, `ruff check .`,
  `mypy src`, `pytest`.

**Punto de Control**: `pyproject.toml` y archivos de configuración presentes; `uv sync`
puede ejecutarse.

---

## Fase 2: Fundacional — Paquete e infraestructura base

**Propósito**: Crear el paquete `sinapsis_ai` importable con sus capas y módulos base.

**CRITICO**: Sin esta fase no hay paquete que testear ni importar.

- [x] T006 [P] Crear `src/sinapsis_ai/__init__.py` (vacío, con versión `__version__ =
  "0.1.0"`).
- [x] T007 [P] Crear los cuatro paquetes de capas vacíos:
  - `src/sinapsis_ai/presentation/__init__.py`
  - `src/sinapsis_ai/application/__init__.py`
  - `src/sinapsis_ai/domain/__init__.py`
  - `src/sinapsis_ai/infrastructure/__init__.py`
- [x] T008 [P] Crear `src/sinapsis_ai/infrastructure/logging.py`: función
  `configure_logging(level: str) -> None` que llama a `logging.basicConfig` con el nivel
  recibido y formato estructurado (incluye timestamp, level, nombre del logger, mensaje).
- [x] T009 [P] Crear `src/sinapsis_ai/main.py`: esqueleto que importa `Settings` y
  `configure_logging`, instancia settings, configura logging y sale limpio. Debe ser
  ejecutable con `python -m sinapsis_ai.main` (bloque `if __name__ == "__main__"`).

**Punto de Control**: `uv run python -c "import sinapsis_ai"` y
`uv run python -m sinapsis_ai.main` terminan sin error. ✓

---

## Fase 3: Historia de Usuario 2 — Configuración desde entorno (Prioridad: P2)

*(HU2 antes de HU1 por dependencia: los tests de HU1 necesitan que el paquete y config existan)*

**Objetivo**: `Settings` (pydantic-settings) carga configuración desde variables de
entorno con fail-fast para variables requeridas.

**Prueba Independiente**: `uv run pytest tests/unit/test_config.py` pasa sin RabbitMQ ni
MONAI.

### Pruebas para HU2

- [x] T010 [P] [HU2] Crear `tests/__init__.py` (vacío).
- [x] T011 [P] [HU2] Crear `tests/conftest.py` con fixture `test_settings` que construye
  `Settings` con variables de prueba (sin leer `.env` real), usando
  `model_config = SettingsConfigDict(env_file=None)` o equivalente.
- [x] T012 [P] [HU2] Crear `tests/unit/test_config.py` con los tres tests del CHANGELOG:
  - `test_settings_loads_from_env` — carga valores desde variables de entorno.
  - `test_settings_missing_required_raises` — falta `RABBITMQ_URL` → error de validación.
  - `test_settings_defaults` — `RABBITMQ_PREFETCH=1`, `MODEL_DEVICE=cpu` por defecto.

### Implementación para HU2

- [x] T013 [P] [HU2] Crear `src/sinapsis_ai/config.py` con clase `Settings` (pydantic
  `BaseSettings`) con todos los campos de §8 de `PROJECT_ARCHITECTURE.md`.
  Nota: `allowed_analysis_types` se recibe como CSV str y se convierte a lista mediante
  `@model_validator` (pydantic-settings v2 parsea listas como JSON; el validator evita
  esa restricción manteniendo la UX de CSV en `.env`).

**Punto de Control**: `uv run pytest tests/unit/test_config.py -v` → 3 tests passing. ✓

---

## Fase 4: Historia de Usuario 1 — Proyecto instalable y verificable (Prioridad: P1)

**Objetivo**: Secuencia completa de verificación (`uv sync → ruff format --check →
ruff check → mypy src → pytest`) termina con código 0.

**Prueba Independiente**: Ejecutar la secuencia completa en el entorno del proyecto.

### Verificación para HU1

- [x] T014 [P] [HU1] Ejecutar `uv sync` → OK (21 paquetes instalados).
- [x] T015 [P] [HU1] Ejecutar `uv run ruff format --check .` → código 0.
- [x] T016 [P] [HU1] Ejecutar `uv run ruff check .` → código 0.
- [x] T017 [P] [HU1] Ejecutar `uv run mypy src` → código 0, sin errores.
- [x] T018 [P] [HU1] Ejecutar `uv run pytest` → 3 passed; tests de integración omitidos.

**Punto de Control**: HU1 completamente verificada — secuencia completa en verde. ✓

---

## Fase 5: Historia de Usuario 3 — Estructura de capas importable (Prioridad: P3)

**Objetivo**: Los cuatro subpaquetes de capas son importables sin error.

**Prueba Independiente**: `python -c "from sinapsis_ai import presentation, application,
domain, infrastructure"` sin `ImportError`; `mypy src` sin errores de módulo faltante.

### Verificación para HU3

- [x] T019 [HU3] Verificar importación de las cuatro capas → imprime `OK`.
- [x] T020 [HU3] Confirmar que `uv run mypy src` no reporta errores de módulo faltante
  para ninguna de las capas. ✓

**Punto de Control**: Estructura de capas verificada e importable. ✓

---

## Fase 6: Acabado y verificación final

**Propósito**: Verificación final completa y actualización de estado.

- [x] T021 Secuencia completa: `ruff format --check` ✓ · `ruff check` ✓ · `mypy src` ✓ · `pytest` ✓
- [x] T022 `uv run python -m sinapsis_ai.main` arranca y sale con código 0. ✓
- [x] T023 `.env.example` contiene todas las variables de §8 de `PROJECT_ARCHITECTURE.md`.
- [x] T024 `.gitignore` excluye `.env` y `.bundle_cache/`.

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Fase 1 (Configuración)**: Sin dependencias — puede comenzar de inmediato.
- **Fase 2 (Fundacional)**: Depende de Fase 1 — BLOQUEA todas las historias.
- **Fase 3 (HU2 — Config)**: Depende de Fase 2; implementar `Settings` antes de los tests.
- **Fase 4 (HU1 — Verificación)**: Depende de Fase 2 y Fase 3 (necesita paquete + config).
- **Fase 5 (HU3 — Capas)**: Depende de Fase 2 (capas ya creadas en T007).
- **Fase 6 (Acabado)**: Depende de Fases 3, 4 y 5.

### Dependencias entre Historias de Usuario

- **HU2 (P2)**: Solo depende de la Fase Fundacional — testeable con `pytest tests/unit/test_config.py`.
- **HU1 (P1)**: Depende de HU2 (necesita `Settings` para que mypy y pytest pasen).
- **HU3 (P3)**: Solo depende de la Fase Fundacional (capas creadas en T007).

### Dentro de Cada Fase

- Archivos de configuración del proyecto antes que código fuente.
- Paquete base antes que subpaquetes.
- Implementación antes que tests.
- Tests antes que verificación final.

## Notas

- [P] = tarea prioritaria / bloqueante dentro de la fase.
- [HU#] = mapeo a historia de usuario del spec.
- `uv.lock` se genera automáticamente con `uv sync`; no editarlo manualmente.
- El marker `integration` en `pyproject.toml` hace que `pytest` sin flags omita esos tests.
- `mypy` con `strict = true` requiere type annotations completas en todos los módulos.
- No commitear `.env`; solo `.env.example`.
- `allowed_analysis_types` en `Settings`: pydantic-settings v2 intenta parsear `list[str]`
  desde env como JSON. Se declaró como `str` + `@model_validator` para soportar CSV en
  `.env` (más ergonómico para operadores). La lista parseada queda en
  `allowed_analysis_types_list`.

## Ejecución Completada

**Fecha**: 2026-07-05
**Estado**: Todas las fases implementadas y verificadas.
