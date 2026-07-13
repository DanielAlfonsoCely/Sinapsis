# Especificación de Funcionalidad: Scaffolding v0.1.0

**Creado**: 2026-07-05

## Escenarios de Usuario y Pruebas *(obligatorio)*

### Historia de Usuario 1 - Proyecto instalable y verificable (Prioridad: P1)

Un desarrollador que acaba de clonar el repositorio puede instalar el entorno con un solo
comando (`uv sync`), ejecutar la suite de verificación completa (format → lint → types →
tests) y obtener todos los checks en verde, confirmando que el scaffolding es correcto y
reproducible.

**Por qué esta prioridad**: Es el prerequisito de todo lo demás. Sin un entorno instalable,
reproducible y con tooling funcionando, no se puede desarrollar ninguna versión posterior.

**Prueba Independiente**: Ejecutar en un entorno limpio:
```
uv sync
uv run ruff format --check .
uv run ruff check .
uv run mypy src
uv run pytest
```
Todos los comandos deben terminar con código de salida 0.

**Escenarios de Aceptación**:

1. **Escenario**: Instalación limpia del entorno
   - **Dado** un entorno sin dependencias instaladas (solo `uv` disponible)
   - **Cuando** se ejecuta `uv sync`
   - **Entonces** todas las dependencias se instalan sin errores y el paquete `sinapsis_ai`
     es importable con `uv run python -c "import sinapsis_ai"`

2. **Escenario**: Formato de código sin errores
   - **Dado** el entorno instalado
   - **Cuando** se ejecuta `uv run ruff format --check .`
   - **Entonces** termina con código 0 (sin archivos mal formateados)

3. **Escenario**: Lint sin errores
   - **Dado** el entorno instalado
   - **Cuando** se ejecuta `uv run ruff check .`
   - **Entonces** termina con código 0 (sin violaciones de lint)

4. **Escenario**: Type check sin errores
   - **Dado** el entorno instalado
   - **Cuando** se ejecuta `uv run mypy src`
   - **Entonces** termina con código 0 (sin errores de tipos)

5. **Escenario**: Tests pasan
   - **Dado** el entorno instalado
   - **Cuando** se ejecuta `uv run pytest`
   - **Entonces** todos los tests unitarios pasan; los tests de integración se omiten por
     defecto (marker `integration`)

---

### Historia de Usuario 2 - Configuración cargada desde entorno (Prioridad: P2)

Un operador puede configurar el servicio exclusivamente a través de variables de entorno
(sin editar código), y el proceso falla de forma rápida y clara si faltan variables
requeridas.

**Por qué esta prioridad**: La configuración tipada y con fail-fast es la base de seguridad
del servicio (no hay credenciales hardcodeadas) y permite el despliegue en múltiples
entornos. Es la segunda pieza más crítica del scaffolding.

**Prueba Independiente**: Los tests en `tests/unit/test_config.py` cubren los tres
escenarios de aceptación sin necesidad de RabbitMQ real ni MONAI, usando solo variables de
entorno inyectadas en el test.

**Escenarios de Aceptación**:

1. **Escenario**: Carga correcta desde variables de entorno
   - **Dado** variables de entorno con `RABBITMQ_URL`, `ALLOWED_ANALYSIS_TYPES` y
     `IMAGE_STORAGE_BACKEND` definidas
   - **Cuando** se instancia `Settings()`
   - **Entonces** los valores se reflejan correctamente en el objeto de configuración

2. **Escenario**: Fallo rápido si falta variable requerida
   - **Dado** variables de entorno sin `RABBITMQ_URL`
   - **Cuando** se instancia `Settings()`
   - **Entonces** se lanza un error de validación de pydantic (fail-fast antes de arrancar)

3. **Escenario**: Valores por defecto correctos
   - **Dado** solo las variables requeridas definidas (sin `RABBITMQ_PREFETCH` ni
     `MODEL_DEVICE`)
   - **Cuando** se instancia `Settings()`
   - **Entonces** `rabbitmq_prefetch == 1` y `model_device == "cpu"`

---

### Historia de Usuario 3 - Estructura de capas creada (Prioridad: P3)

Un desarrollador que inicia la versión v0.2.0 encuentra los paquetes de las cuatro capas
(`presentation/`, `application/`, `domain/`, `infrastructure/`) ya creados con sus
`__init__.py`, listos para recibir módulos sin conflictos de importación ni estructura
incorrecta.

**Por qué esta prioridad**: Aunque necesaria, esta historia no aporta lógica verificable
por sí misma más allá de la existencia de directorios e `__init__.py`. Su valor es de
andamiaje para las versiones siguientes.

**Prueba Independiente**: Verificar que `mypy src` no reporta errores de módulo faltante y
que `python -c "from sinapsis_ai import presentation, application, domain, infrastructure"`
importa sin error.

**Escenarios de Aceptación**:

1. **Escenario**: Paquetes de capas importables
   - **Dado** el entorno instalado
   - **Cuando** se importan los cuatro subpaquetes de capas
   - **Entonces** no hay `ImportError` ni errores de mypy por módulos faltantes

---

### Casos Límite

- ¿Qué ocurre si `ALLOWED_ANALYSIS_TYPES` está vacío? → `Settings()` debe rechazarlo con
  error de validación (lista requerida y no vacía).
- ¿Qué ocurre si `MODEL_DEVICE` recibe un valor no soportado (ej. `"tpu"`)? → error de
  validación pydantic en el arranque.
- ¿Qué ocurre si se ejecutan los tests sin haber corrido `uv sync`? → error de importación
  claro, no un fallo críptico.
- Tests marcados `@pytest.mark.integration` deben omitirse al correr `pytest` sin el flag
  `-m integration`.

---

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El proyecto DEBE ser gestionado con `uv` (layout `src/`, `pyproject.toml`,
  `uv.lock`); sin `pip`, `poetry` ni `venv` manual.
- **RF-002**: El paquete importable DEBE llamarse `sinapsis_ai` (snake_case) y vivir en
  `src/sinapsis_ai/`.
- **RF-003**: El sistema DEBE incluir `ruff` (format + lint) y `mypy` configurados en
  `pyproject.toml` y ejecutables vía `uv run`.
- **RF-004**: El sistema DEBE incluir `pytest` y `pytest-cov` con el marker `integration`
  declarado; los tests de integración DEBEN omitirse por defecto.
- **RF-005**: `Settings` (pydantic-settings) DEBE leer configuración exclusivamente desde
  variables de entorno; sin valores sensibles hardcodeados.
- **RF-006**: Las variables `RABBITMQ_URL`, `ALLOWED_ANALYSIS_TYPES` e
  `IMAGE_STORAGE_BACKEND` DEBEN ser requeridas (sin default); el proceso DEBE fallar al
  arranque si faltan.
- **RF-007**: `RABBITMQ_PREFETCH` DEBE tener default `1`; `MODEL_DEVICE` DEBE tener default
  `"cpu"`.
- **RF-008**: El proyecto DEBE incluir `.env.example` con todas las variables documentadas y
  `.gitignore` que excluya `.env` y `.bundle_cache/`.
- **RF-009**: Los paquetes de las cuatro capas (`presentation/`, `application/`, `domain/`,
  `infrastructure/`) DEBEN existir con `__init__.py` vacíos bajo `src/sinapsis_ai/`.
- **RF-010**: `src/sinapsis_ai/infrastructure/logging.py` DEBE configurar logging
  estructurado básico (nivel desde `LOG_LEVEL`).
- **RF-011**: `src/sinapsis_ai/main.py` DEBE ser un esqueleto que arranque y salga limpio
  (sin lógica de inferencia); ejecutable con `uv run python -m sinapsis_ai.main`.
- **RF-012**: El CI (`.github/workflows/ci.yml`) DEBE ejecutar la secuencia de verificación
  completa: `ruff format --check` → `ruff check` → `mypy src` → `pytest`.

### Entidades Clave

- **`Settings`**: Objeto de configuración pydantic-settings. Campos: `rabbitmq_url`,
  `rabbitmq_request_queue`, `rabbitmq_result_exchange`, `rabbitmq_result_routing_key`,
  `rabbitmq_prefetch`, `bundle_cache_dir`, `bundle_source`, `allowed_analysis_types`,
  `model_device`, `image_storage_backend`, `log_level`.

---

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: `uv sync && uv run ruff format --check . && uv run ruff check . && uv run mypy src && uv run pytest` termina con código 0 en un entorno limpio.
- **CE-002**: `uv run pytest` ejecuta al menos los 3 tests de `tests/unit/test_config.py`
  y todos pasan.
- **CE-003**: `uv run pytest -m integration` no falla por ausencia de RabbitMQ cuando no
  hay tests de integración en v0.1.0 (0 tests collected es aceptable).
- **CE-004**: `uv run python -m sinapsis_ai.main` arranca y termina con código 0 (esqueleto
  limpio, sin error de importación).
- **CE-005**: `uv run mypy src` no reporta ningún error de tipo en los módulos del
  scaffolding.
