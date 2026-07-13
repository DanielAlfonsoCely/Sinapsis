# Especificación de Funcionalidad: Endurecimiento para Producción (v1.0.0)

**Creado**: 2026-07-13

## Escenarios de Usuario y Pruebas *(obligatorio)*

### Historia de Usuario 1 - Política de reintentos acotada y dead-letter explícito (Prioridad: P1)

El consumidor aplica una política de reintentos configurable para errores **transitorios**
(e.g. `BundleResolutionError`, `ImageAccessError`): reintenta el procesamiento hasta
`RABBITMQ_MAX_RETRIES` veces con backoff, y tras agotar los intentos envía el mensaje a
un dead-letter exchange declarado explícitamente. Los errores no recuperables
(`InvalidMessageError`, `UnknownAnalysisTypeError`) siguen yendo a dead-letter
inmediatamente sin reintentar.

**Por qué esta prioridad**: Es el requisito de resiliencia más crítico para producción.
Sin él, un error transitorio (broker momentáneamente lento, imagen tardando en estar
disponible) mata mensajes válidos. Además el dead-letter exchange debe ser explícito para
que el operador pueda inspeccionar y reintentar mensajes fallidos manualmente.

**Prueba Independiente**: Puede testearse con canal `pika` mockeado y un servicio doblez
que lanza `BundleResolutionError` N veces. No requiere RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: Error transitorio con reintentos exitosos
   - **Dado** un servicio que lanza `BundleResolutionError` las primeras 2 veces y luego
     retorna éxito, con `MAX_RETRIES=3`
   - **Cuando** el consumidor procesa el mensaje
   - **Entonces** el servicio es invocado 3 veces, el resultado se publica y el canal recibe `basic_ack`

2. **Escenario**: Reintentos agotados → dead-letter
   - **Dado** un servicio que siempre lanza `BundleResolutionError` y `MAX_RETRIES=3`
   - **Cuando** el consumidor procesa el mensaje
   - **Entonces** el servicio es invocado exactamente `MAX_RETRIES` veces y el canal recibe
     `basic_nack(requeue=False)` (dead-letter)

3. **Escenario**: Error no recuperable → dead-letter inmediato (sin reintentos)
   - **Dado** un payload con `analysis_type` desconocido y `MAX_RETRIES=3`
   - **Cuando** el consumidor procesa el mensaje
   - **Entonces** el servicio NO es invocado, `basic_nack(requeue=False)` se emite
     al primer intento (0 reintentos)

4. **Escenario**: Dead-letter exchange declarado explícitamente al arrancar
   - **Dado** configuración con `RABBITMQ_DLX_NAME=sinapsis.ai.dlx`
   - **Cuando** el composition root inicializa la topología AMQP
   - **Entonces** el exchange dead-letter y la cola de requests quedan vinculados
     con `x-dead-letter-exchange`

---

### Historia de Usuario 2 - Healthcheck de liveness y readiness (Prioridad: P1)

El servicio expone un mecanismo de healthcheck que permite al orquestador (Docker/k8s)
detectar si el worker está vivo (`liveness`) y si está listo para procesar mensajes
(`readiness`). El healthcheck verifica la conexión a RabbitMQ y la disponibilidad del
directorio de caché de bundles.

**Por qué esta prioridad**: Sin healthcheck el orquestador no puede reiniciar el worker
cuando pierde la conexión al broker o queda colgado. Es bloqueante para despliegue en
producción.

**Prueba Independiente**: Puede testearse con conexión `pika` mockeada y sistema de
archivos real (tmp_path). No requiere RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: Healthcheck liveness pasa con conexión abierta y caché accesible
   - **Dado** una conexión `pika` abierta y un `BUNDLE_CACHE_DIR` existente
   - **Cuando** se llama a `healthcheck.is_alive()`
   - **Entonces** retorna `True`

2. **Escenario**: Liveness falla con conexión cerrada
   - **Dado** una conexión `pika` cerrada (`.is_open = False`)
   - **Cuando** se llama a `healthcheck.is_alive()`
   - **Entonces** retorna `False`

3. **Escenario**: Readiness falla si el directorio de caché no existe
   - **Dado** `BUNDLE_CACHE_DIR` apuntando a un directorio inexistente
   - **Cuando** se llama a `healthcheck.is_ready()`
   - **Entonces** retorna `False`

4. **Escenario**: Healthcheck escribe resultado en archivo para Docker HEALTHCHECK
   - **Dado** un healthcheck configurado con un archivo de estado
   - **Cuando** el worker está vivo y listo
   - **Entonces** el archivo de estado existe con contenido `ok` (compatible con
     `CMD cat /tmp/health`)

---

### Historia de Usuario 3 - Apagado ordenado bajo carga (Prioridad: P2)

Al recibir SIGTERM, el worker termina de procesar el mensaje **actualmente en vuelo**
antes de cerrar la conexión, garantizando que ningún mensaje quede en estado indefinido.
Si no hay mensaje en vuelo, el apagado es inmediato.

**Por qué esta prioridad**: En v0.5.0 el SIGTERM interrumpe inmediatamente el bucle de
consumo; si hay una inferencia en curso, el ack/nack nunca se emite y RabbitMQ reencola
el mensaje (o espera hasta el timeout). Para producción se necesita terminar limpiamente.

**Prueba Independiente**: Testeable con `main.py` mockeado y un servicio que tarda
deliberadamente. No requiere RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: SIGTERM con mensaje en vuelo → completa antes de cerrar
   - **Dado** el worker procesando un mensaje (inferencia simulada con sleep)
   - **Cuando** llega SIGTERM
   - **Entonces** el mensaje completa su procesamiento (ack emitido) y luego se cierra

2. **Escenario**: SIGTERM sin mensaje en vuelo → cierre inmediato
   - **Dado** el worker esperando mensajes (idle)
   - **Cuando** llega SIGTERM
   - **Entonces** el bucle se detiene y la conexión se cierra limpiamente

---

### Historia de Usuario 4 - Métricas en logs estructurados (Prioridad: P2)

Cada mensaje procesado emite una entrada de log estructurado que incluye: `request_id`,
`correlation_id`, `analysis_type`, `model_name`, `model_version`, `status`,
`duration_ms`, `artifacts_count` y (si aplica) `error_code`. Esto permite monitorización
y alertas sin instrumentación adicional.

**Por qué esta prioridad**: El CHANGELOG especifica "métricas básicas en logs
estructurados" como requisito de producción. Permite al equipo de operaciones detectar
degradación de rendimiento y tasa de errores sin añadir dependencias externas.

**Prueba Independiente**: Testeable capturando el output de logging con `caplog` de
pytest. No requiere RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: Log de éxito incluye todas las métricas requeridas
   - **Dado** un análisis completado con éxito
   - **Cuando** el service loguea el resultado
   - **Entonces** el log incluye `request_id`, `duration_ms`, `status=succeeded`,
     `artifacts_count` y `model_name`

2. **Escenario**: Log de fallo incluye código de error
   - **Dado** un análisis fallido por `InferenceError`
   - **Cuando** el service loguea el resultado
   - **Entonces** el log incluye `status=failed` y `error_code=INFERENCE_ERROR`

3. **Escenario**: Log de reintento incluye intento actual y máximo
   - **Dado** un error transitorio que activa reintentos
   - **Cuando** el consumer reintenta
   - **Entonces** el log incluye `attempt=N`, `max_retries=M` y el tipo de error

---

### Historia de Usuario 5 - Reconexión tras caída del broker (integración) (Prioridad: P3)

El test de integración valida que el worker detecta la desconexión del broker y reconecta
automáticamente (o falla limpiamente con log del error), sin quedar en estado zombie.

**Por qué esta prioridad**: Valida la robustez end-to-end pero requiere infraestructura
real (Docker). Las HU anteriores cubren el comportamiento funcional; esta es la validación
operacional.

**Prueba Independiente**: Requiere Docker. Marcado con `@pytest.mark.integration`.

**Escenarios de Aceptación**:

1. **Escenario**: Reconexión automática tras restart del broker
   - **Dado** el worker conectado a RabbitMQ en Docker
   - **Cuando** el contenedor RabbitMQ se reinicia
   - **Entonces** el worker detecta la desconexión, loguea el error y termina limpiamente
     (el orquestador lo reinicia) — o reconecta si se implementa reconexión activa

---

### Casos Límite

- ¿Qué ocurre si `MAX_RETRIES=0`? → El error transitorio va directamente a dead-letter
  sin ningún reintento (comportamiento de v0.5.0).
- ¿Qué ocurre si el DLX no existe al arrancar? → `main.py` lo declara explícitamente;
  nunca depende de que ya exista.
- ¿El healthcheck puede bloquear al worker? → No; se ejecuta en un hilo separado o
  mediante escritura en archivo (sin bloquear el bucle de consumo).
- ¿El backoff entre reintentos usa sleep bloqueante? → Sí, dado `prefetch=1`; no hay
  otro mensaje en vuelo que se vea afectado.
- ¿Qué ocurre si la inferencia en vuelo tarda más de 30 s al recibir SIGTERM? → Se
  respeta un timeout configurable (`SHUTDOWN_TIMEOUT_S`); si se supera, se cierra igual
  y el message se reencola.
- ¿Las métricas de duración incluyen tiempo de descarga de bundle? → Solo si el bundle
  no estaba en caché; el `duration_ms` del `AnalysisResult` mide el flujo completo desde
  que `run_analysis` recibe la solicitud.

---

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El consumidor DEBE aplicar política de reintentos con backoff para errores
  clasificados como transitorios (`BundleResolutionError`, `ImageAccessError`) hasta
  `RABBITMQ_MAX_RETRIES` veces (configurable, default `3`).
- **RF-002**: Tras agotar los reintentos, el consumidor DEBE emitir
  `basic_nack(requeue=False)` enviando el mensaje al dead-letter exchange.
- **RF-003**: Los errores no recuperables (`InvalidMessageError`, `UnknownAnalysisTypeError`)
  DEBEN ir a dead-letter en el primer intento, sin ningún reintento.
- **RF-004**: El dead-letter exchange (`RABBITMQ_DLX_NAME`, default `sinapsis.ai.dlx`) DEBE
  declararse explícitamente al arrancar y la cola de solicitudes debe quedar vinculada con
  `x-dead-letter-exchange`.
- **RF-005**: El servicio DEBE exponer un healthcheck que verifique la conexión al broker
  y la accesibilidad del directorio de caché de bundles.
- **RF-006**: El healthcheck DEBE escribir su resultado en un archivo (`HEALTH_FILE`,
  default `/tmp/sinapsis_health`) para compatibilidad con `HEALTHCHECK CMD` de Docker.
- **RF-007**: Al recibir SIGTERM, el worker DEBE completar el mensaje actualmente en
  proceso (si lo hay) antes de cerrar la conexión AMQP.
- **RF-008**: El apagado graceful DEBE respetar un timeout configurable
  (`SHUTDOWN_TIMEOUT_S`, default `30`); si se supera, el proceso termina igualmente.
- **RF-009**: Cada mensaje procesado DEBE emitir un log estructurado con `request_id`,
  `correlation_id`, `analysis_type`, `model_name`, `model_version`, `status`,
  `duration_ms`, `artifacts_count` y `error_code` (si aplica).
- **RF-010**: La cobertura global de tests DEBE alcanzar o superar el 85% tras la
  verificación final (`uv run pytest --cov=sinapsis_ai --cov-report=term-missing`).

### Entidades Clave

- **`RetryPolicy`** (`presentation/retry_policy.py`): encapsula la lógica de reintentos
  acotados con backoff. Recibe una callable (el servicio) y la invoca con reintentos.
  Pertenece a Presentation porque decide qué hacer con el mensaje (ack/nack).
- **`HealthCheck`** (`infrastructure/healthcheck.py`): verifica la conexión al broker y el
  directorio de caché. Escribe el resultado en un archivo. Pertenece a Infrastructure
  porque accede a recursos externos (pika connection, filesystem).

---

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: `uv run pytest tests/unit/` pasa al 100% incluyendo los nuevos tests de
  política de reintentos (`test_retry_policy.py`) y apagado (`test_main.py`).
- **CE-002**: `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` reporta
  cobertura ≥ 85% (umbral ya configurado en `pyproject.toml`).
- **CE-003**: `uv run mypy src` pasa sin errores en los nuevos módulos.
- **CE-004**: `uv run ruff format --check . && uv run ruff check .` pasan sin hallazgos.
- **CE-005**: `docker inspect` del contenedor muestra el healthcheck en estado `healthy`
  tras `docker compose up`.
- **CE-006**: El log de un análisis exitoso contiene los campos `duration_ms`, `status`,
  `model_name` y `artifacts_count` verificados en `test_analysis_service.py` con `caplog`.
- **CE-007**: Un error transitorio que se recupera en el segundo intento no genera ningún
  nack, solo el ack final.
- **CE-008**: Un error transitorio que agota `MAX_RETRIES=3` genera exactamente 3
  invocaciones al servicio y 1 nack final.
