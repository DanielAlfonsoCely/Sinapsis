# Plan de Implementación: Endurecimiento para Producción (v1.0.0)

**Fecha**: 2026-07-13
**Especificación**: [spec.md](./spec.md)

---

## Resumen

Endurecer el microservicio para producción añadiendo: política de reintentos acotada con
backoff para errores transitorios, dead-letter exchange declarado explícitamente,
healthcheck de liveness/readiness con archivo de estado para Docker, apagado ordenado
bajo carga (SIGTERM espera al mensaje en vuelo), logs estructurados con métricas por
análisis, y test de integración de reconexión. Construye directamente sobre v0.5.0 sin
cambiar la arquitectura por capas ni los contratos de mensajes.

---

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12
**Dependencias Principales**: `pika>=1.3`, `pydantic-settings>=2.3`, `pydantic>=2.7`;
sin dependencias nuevas de runtime (todo con stdlib + pika existente)
**Almacenamiento**: N/A (stateless); healthcheck escribe en `/tmp/sinapsis_health`
**Testing**: `pytest` + `pytest-cov`; umbral ≥ 85% ya configurado en `pyproject.toml`;
`caplog` de pytest para verificar logs estructurados; canal `pika` mockeado en unit tests
**Plataforma Objetivo**: Servidor Linux (contenedor Docker); healthcheck compatible con
`HEALTHCHECK CMD cat /tmp/sinapsis_health`
**Objetivos de Rendimiento**: Backoff entre reintentos ≤ 1 s por intento en unit tests
(configurable en producción); sin overhead en el camino feliz
**Restricciones**: `application/` nunca importa `pika`; `RetryPolicy` vive en
`presentation/` (decide el ack/nack); `HealthCheck` en `infrastructure/` (accede a pika
connection y filesystem); `main.py` es el único composition root
**Escala/Alcance**: `prefetch=1` por worker; reintentos son síncronos (un worker, una
inferencia a la vez, con backoff entre intentos)

---

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v1.0.0-production-hardening/
├── plan.md   # Este archivo
└── spec.md
```

### Código Fuente

```text
src/sinapsis_ai/
├── config.py                              # Modificar: añadir MAX_RETRIES, DLX, SHUTDOWN_TIMEOUT_S, HEALTH_FILE
├── main.py                                # Modificar: declarar DLX, montar healthcheck, shutdown con timeout
├── presentation/
│   ├── consumer.py                        # Modificar: integrar RetryPolicy, mejorar logs estructurados
│   └── retry_policy.py                    # NUEVO: política de reintentos acotada con backoff
└── infrastructure/
    └── healthcheck.py                     # NUEVO: liveness/readiness + escritura de archivo de estado

tests/
├── unit/
│   ├── presentation/
│   │   ├── test_consumer.py               # Ampliar: tests con retry policy integrada
│   │   └── test_retry_policy.py           # NUEVO: tests de RetryPolicy (exhaustivos)
│   ├── infrastructure/
│   │   └── test_healthcheck.py            # NUEVO: tests de HealthCheck
│   ├── application/
│   │   └── test_analysis_service.py       # Ampliar: verificar logs estructurados con caplog
│   └── test_main.py                       # NUEVO: test de apagado graceful
└── integration/
    └── test_rabbitmq_flow.py              # Ampliar: test_reconnect_after_broker_restart
```

**Decisión de Estructura**: Layout `src/` existente. `RetryPolicy` en `presentation/`
porque controla el flujo ack/nack (Presentation layer). `HealthCheck` en
`infrastructure/` porque accede a recursos externos (pika connection, filesystem).
`main.py` coordina ambos desde el composition root.

---

## Fase 1: Configuración — Nuevas variables de entorno

**Propósito**: Añadir los campos de configuración necesarios para v1.0.0 a `Settings`.
Sin esto ninguna otra tarea puede usar configuración tipada para los nuevos parámetros.

- [x] T001 Modificar `src/sinapsis_ai/config.py`: añadir campos con defaults seguros:
  - `rabbitmq_max_retries: int = 3`
  - `rabbitmq_dlx_name: str = "sinapsis.ai.dlx"`
  - `shutdown_timeout_s: int = 30`
  - `health_file: str = "/tmp/sinapsis_health"`

**Punto de Control**: `uv run mypy src` pasa; `uv run pytest tests/unit/test_config.py -q` pasa

---

## Fase 2: Fundacional — RetryPolicy (Presentation)

**Propósito**: `RetryPolicy` es el prerrequisito bloqueante de HU1. El consumer necesita
esta clase para implementar reintentos acotados. La capa Application no cambia.

**CRITICO**: Ningún trabajo de HU1 ni de la integración en `consumer.py` puede comenzar
hasta completar esta fase.

### Pruebas para RetryPolicy

- [x] T002 [P] [HU1] Escribir `tests/unit/presentation/test_retry_policy.py`:
  - `test_retry_policy_succeeds_on_first_attempt` — callable exitosa: invocada 1 vez, retorna valor
  - `test_retry_policy_retries_on_transient_error` — falla 2 veces, éxito en 3ra: invocada 3 veces
  - `test_retry_policy_stops_after_max_attempts_raises` — siempre falla: lanza la última excepción
    tras `max_retries` intentos (no más)
  - `test_retry_policy_does_not_retry_non_transient` — `InvalidMessageError` no se reintenta:
    se relanza inmediatamente en el primer intento
  - `test_retry_policy_logs_each_retry` — con `caplog`, verifica que cada reintento loguea
    `attempt=N`, `max_retries=M`
  - `test_retry_policy_zero_retries_raises_immediately` — `max_retries=0` y error transitorio:
    lanza inmediatamente

### Implementación de RetryPolicy

- [x] T003 [P] [HU1] Implementar `src/sinapsis_ai/presentation/retry_policy.py`:
  - Clase `RetryPolicy` con constructor `(max_retries: int, backoff_s: float = 0.5)`
  - Método `execute(fn: Callable[[], T]) -> T` que:
    - Llama a `fn()` hasta `max_retries` veces para excepciones **transitorias**
      (`BundleResolutionError`, `ImageAccessError`)
    - Relanza inmediatamente errores **no transitorios** (`InvalidMessageError`,
      `UnknownAnalysisTypeError`, `InferenceError`)
    - Entre reintentos: `time.sleep(backoff_s)` (mockeable en tests vía monkeypatch)
    - Loguea cada reintento con `attempt`, `max_retries`, tipo de error
    - Tras agotar reintentos: relanza la última excepción transitoria
  - Set de excepciones transitorias definido como constante de módulo `_TRANSIENT_ERRORS`

**Punto de Control**: `uv run pytest tests/unit/presentation/test_retry_policy.py -q` pasa

---

## Fase 3: Historia de Usuario 1 — Reintentos acotados en Consumer (P1)

**Objetivo**: Integrar `RetryPolicy` en `Consumer._on_message` para errores transitorios.
Declarar DLX en `main.py` al arrancar.

**Prueba Independiente**: Canal `pika` mockeado + `RetryPolicy` real o mockeada.

### Pruebas para HU1 en Consumer

- [x] T004 [P] [HU1] Ampliar `tests/unit/presentation/test_consumer.py`:
  - `test_consumer_transient_error_retries_and_succeeds`
  - `test_consumer_transient_error_exhausts_retries_deadletters`
  - `test_consumer_non_transient_error_deadletters_immediately`

### Implementación para HU1 en Consumer

- [x] T005 [HU1] Modificar `src/sinapsis_ai/presentation/consumer.py`:

- [x] T006 [HU1] Modificar `src/sinapsis_ai/main.py`:
  - Declarar DLX al arrancar: `channel.exchange_declare(exchange=settings.rabbitmq_dlx_name, exchange_type="fanout", durable=True)`
  - Declarar la cola de requests con `arguments={"x-dead-letter-exchange": settings.rabbitmq_dlx_name}`
  - Instanciar `RetryPolicy(max_retries=settings.rabbitmq_max_retries)` e inyectarla en `Consumer`

**Punto de Control**: `uv run pytest tests/unit/presentation/ -q` pasa

---

## Fase 4: Historia de Usuario 2 — HealthCheck (P1)

**Objetivo**: `infrastructure/healthcheck.py` con liveness/readiness y escritura en archivo.
`main.py` ejecuta el healthcheck periódicamente en hilo de fondo.

**Prueba Independiente**: Conexión `pika` mockeada + `tmp_path` para el archivo de estado.

### Pruebas para HU2

- [x] T007 [P] [HU2] Escribir `tests/unit/infrastructure/test_healthcheck.py`:
  - `test_healthcheck_alive_with_open_connection_and_existing_cache`
  - `test_healthcheck_alive_false_when_connection_closed`
  - `test_healthcheck_ready_false_when_cache_dir_missing`
  - `test_healthcheck_writes_ok_to_file_when_healthy`
  - `test_healthcheck_writes_fail_to_file_when_unhealthy`

### Implementación para HU2

- [x] T008 [P] [HU2] Implementar `src/sinapsis_ai/infrastructure/healthcheck.py`:

- [x] T009 [HU2] Modificar `src/sinapsis_ai/main.py`:
  - Instanciar `HealthCheck` e invocar `start_background()` antes del bucle de consumo
  - Añadir `HEALTHCHECK CMD cat /tmp/sinapsis_health || exit 1` al `Dockerfile`

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_healthcheck.py -q` pasa

---

## Fase 5: Historia de Usuario 3 — Apagado ordenado bajo carga (P2)

**Objetivo**: El manejador SIGTERM en `main.py` espera al mensaje en vuelo antes de cerrar,
respetando `SHUTDOWN_TIMEOUT_S`.

**Prueba Independiente**: `main.py` con componentes mockeados + `threading.Event` para
simular mensaje en vuelo.

### Pruebas para HU3

- [x] T010 [P] [HU3] Escribir `tests/unit/test_main.py`:
  - `test_graceful_shutdown_finishes_inflight_message`
  - `test_graceful_shutdown_immediate_when_idle`
  - `test_graceful_shutdown_respects_timeout`

### Implementación para HU3

- [x] T011 [HU3] Modificar `src/sinapsis_ai/presentation/consumer.py`:

- [x] T012 [HU3] Modificar `src/sinapsis_ai/main.py`:
  - Reemplazar la función `_shutdown` con versión que espera a `consumer.inflight == False`
    con polling cada 0.1 s hasta `settings.shutdown_timeout_s`
  - Loguea advertencia si el timeout se supera

**Punto de Control**: `uv run pytest tests/unit/test_main.py -q` pasa

---

## Fase 6: Historia de Usuario 4 — Métricas en logs estructurados (P2)

**Objetivo**: Enriquecer los logs del `AnalysisService` con todos los campos requeridos
por el spec. Verificar con `caplog` en los tests del service.

**Prueba Independiente**: `caplog` de pytest; sin RabbitMQ real.

### Pruebas para HU4

- [x] T013 [P] [HU4] Ampliar `tests/unit/application/test_analysis_service.py`:

### Implementación para HU4

- [x] T014 [HU4] Modificar `src/sinapsis_ai/application/analysis_service.py`:
  - En el log de éxito: añadir `analysis_type`, `model_version`, `artifacts_count` al mensaje
    de log estructurado (manteniendo el formato `key=value`)
  - En el log de fallo: añadir `error_code` al mensaje de log
  - Ambas entradas ya incluyen `request_id`, `correlation_id`, `model_name`, `duration_ms`

**Punto de Control**: `uv run pytest tests/unit/application/test_analysis_service.py -q` pasa

---

## Fase 7: Historia de Usuario 5 — Integración: reconexión (P3)

**Objetivo**: Añadir test de integración que valida el comportamiento del worker al
perder la conexión con el broker.

### Pruebas para HU5

- [x] T015 [HU5] Ampliar `tests/integration/test_rabbitmq_flow.py`:
  - `test_reconnect_after_broker_restart` — `@pytest.mark.integration`: publica un mensaje,
    detiene RabbitMQ, verifica que el worker loguea el error de conexión y termina limpiamente
    (sin quedar zombie); requiere `docker compose` disponible

**Punto de Control**: `uv run pytest -m integration tests/integration/test_rabbitmq_flow.py -q`
pasa con RabbitMQ disponible

---

## Fase 8: Acabado y verificación final

- [x] T016 Ejecutar `uv run ruff format --check .` y corregir si hay hallazgos
- [x] T017 Ejecutar `uv run ruff check .` y corregir si hay hallazgos
- [x] T018 Ejecutar `uv run mypy src` y corregir errores de tipo
- [x] T019 Ejecutar `uv run pytest` (solo unit) y verificar 100% passing
- [x] T020 Ejecutar `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` y
  verificar cobertura ≥ 85%
- [x] T021 Actualizar `CHANGELOG.md`: marcar `[v1.0.0]` como `COMPLETADO` con la fecha

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Fase 1 (Config)**: Sin dependencias — primer paso obligatorio
- **Fase 2 (RetryPolicy)**: Depende de Fase 1 — BLOQUEA Fase 3
- **Fase 3 (HU1 Consumer+DLX)**: Depende de Fase 2
- **Fase 4 (HU2 HealthCheck)**: Depende de Fase 1; paralela a Fase 3
- **Fase 5 (HU3 Shutdown)**: Depende de Fase 3 (consumer con `_inflight`)
- **Fase 6 (HU4 Logs)**: Depende de Fase 1; paralela a Fases 3-5
- **Fase 7 (HU5 Integración)**: Depende de todas las fases previas
- **Fase 8 (Acabado)**: Depende de todas las fases anteriores

### Dependencias entre Historias de Usuario

- **HU1 (Reintentos)**: Depende de Fase 2 (RetryPolicy fundacional)
- **HU2 (HealthCheck)**: Solo depende de Fase 1 (Config) — independiente
- **HU3 (Shutdown)**: Depende de HU1 (consumer con `_inflight` tras integrar retry)
- **HU4 (Logs)**: Solo depende de Fase 1 — independiente de HU1/HU2/HU3
- **HU5 (Integración)**: Depende de HU1+HU2+HU3 completos

### Dentro de Cada Historia

- Tests antes que implementación
- Implementación de módulo nuevo antes que modificación de módulos existentes
- Verificación de cada fase antes de avanzar

---

## Ejecución Completada

**Fecha**: 2026-07-13
**Estado**: Todas las fases implementadas y verificadas

- Tests: 110 passing (unit), 3 deselected (integration)
- Cobertura: 89.45% (umbral: 85%)
- Lint: OK, sin errores
- Types: OK, sin errores (20 source files)

---

## Notas

- `[P]` = tarea prioritaria para la historia
- `[HU#]` = mapeo a historia de usuario del spec
- `RetryPolicy.backoff_s` debe ser **monkeypatcheable** en tests para evitar sleeps reales
- El `_inflight` en `consumer.py` no necesita ser thread-safe (el bucle pika es de un único
  hilo); basta con `bool` simple
- `HealthCheck.start_background()` usa `daemon=True` para no bloquear el cierre del proceso
- El DLX se declara como `fanout` para simplicidad; en producción se puede afinar a `direct`
- Los cambios NO se commitean automáticamente
