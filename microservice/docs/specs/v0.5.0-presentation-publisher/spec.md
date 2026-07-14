# Especificación de Funcionalidad: Presentation + Publicación (v0.5.0)

**Creado**: 2026-07-13

## Escenarios de Usuario y Pruebas *(obligatorio)*

### Historia de Usuario 1 - Consumidor RabbitMQ con ack/nack (Prioridad: P1)

El worker arranca, conecta a RabbitMQ con `prefetch=1`, recibe un mensaje de solicitud de
análisis, lo valida, lo delega al `AnalysisService` y emite el acuse correspondiente:
`ack` si la inferencia termina (exitosa o fallida con resultado publicado), `nack` sin
requeue si el mensaje es irrecuperable (malformado o tipo desconocido).

**Por qué esta prioridad**: Es el núcleo del microservicio. Sin consumidor no hay flujo.
Además completa la arquitectura por capas: cablea Presentation → Application → Infra.

**Prueba Independiente**: Puede testearse completamente con canal `pika` mockeado,
un `AnalysisService` doblez y mensajes JSON de fixture. No requiere RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: Mensaje válido produce resultado y recibe ack
   - **Dado** un canal `pika` mockeado y un `AnalysisService` que devuelve un `AnalysisResult(status=succeeded)`
   - **Cuando** el consumidor recibe un mensaje JSON válido de `sample_request.json`
   - **Entonces** el servicio es invocado, el resultado es publicado y el canal recibe `basic_ack`

2. **Escenario**: Mensaje inválido recibe nack sin requeue (dead-letter)
   - **Dado** un canal mockeado y un mensaje JSON con `image_uri` ausente
   - **Cuando** el consumidor intenta procesar el mensaje
   - **Entonces** se lanza `InvalidMessageError`, el canal recibe `basic_nack(requeue=False)` y **no** se invoca el servicio

3. **Escenario**: `analysis_type` desconocido recibe nack sin requeue
   - **Dado** un canal mockeado y un mensaje con `analysis_type` no permitido
   - **Cuando** el consumidor procesa el mensaje
   - **Entonces** se lanza `UnknownAnalysisTypeError` y el canal recibe `basic_nack(requeue=False)`

4. **Escenario**: Error de inferencia publica resultado fallido y recibe ack
   - **Dado** un `AnalysisService` que lanza `InferenceError`
   - **Cuando** el consumidor procesa un mensaje válido
   - **Entonces** el servicio construye un `AnalysisResult(status=failed)`, lo publica y el canal recibe `basic_ack`

---

### Historia de Usuario 2 - Publisher de resultados persistente (Prioridad: P1)

La capa de infraestructura publica el `AnalysisResult` al exchange de resultados de
RabbitMQ con `delivery_mode=2` (persistente), de modo que los resultados sobrevivan
reinicios del broker antes de ser consumidos por el backend Go.

**Por qué esta prioridad**: Sin publicador no hay salida del sistema. Va junto con el
consumidor (P1) porque ambos son bloqueantes para que el servicio sea funcional.

**Prueba Independiente**: Puede testearse con canal `pika` mockeado verificando que se
llama `basic_publish` con los parámetros correctos (exchange, routing key, persistence,
body serializado). No requiere RabbitMQ real.

**Escenarios de Aceptación**:

1. **Escenario**: Publicación persistente exitosa
   - **Dado** un canal `pika` mockeado y un `AnalysisResult(status=succeeded)` válido
   - **Cuando** se llama a `publisher.publish(result)`
   - **Entonces** el canal recibe `basic_publish` con `delivery_mode=2`, el exchange correcto,
     la routing key correcta y el body es un JSON válido que representa el resultado

2. **Escenario**: Publicación de resultado fallido
   - **Dado** un `AnalysisResult(status=failed, error={...})`
   - **Cuando** se llama a `publisher.publish(result)`
   - **Entonces** el JSON publicado incluye el bloque `error` y `status=failed`

---

### Historia de Usuario 3 - Composition root y arranque limpio (Prioridad: P2)

`main.py` actúa como composition root: carga `Settings`, inicializa logging, construye
todos los componentes de infraestructura, los inyecta en `AnalysisService`, monta el
`Consumer` y arranca el bucle de consumo. Maneja SIGTERM/SIGINT con apagado limpio
(cierra conexión RabbitMQ antes de salir).

**Por qué esta prioridad**: Es la pieza de integración que hace arrancar el servicio real,
pero no bloquea los tests de HU1/HU2 ya que esas capas se testean con dobles inyectados.

**Prueba Independiente**: Se puede verificar que el servicio arranca limpiamente
(`uv run python -m sinapsis_ai.main`) y que el `docker-compose.yml` levanta RabbitMQ.
El test de integración `test_rabbitmq_round_trip` valida el flujo completo.

**Escenarios de Aceptación**:

1. **Escenario**: Arranque exitoso con configuración válida
   - **Dado** variables de entorno válidas (`RABBITMQ_URL`, `ALLOWED_ANALYSIS_TYPES`, etc.)
   - **Cuando** se ejecuta `python -m sinapsis_ai.main`
   - **Entonces** el proceso arranca, conecta a RabbitMQ, declara prefetch=1 y espera mensajes

2. **Escenario**: Fallo rápido con configuración inválida
   - **Dado** `RABBITMQ_URL` ausente en el entorno
   - **Cuando** se ejecuta `python -m sinapsis_ai.main`
   - **Entonces** el proceso sale con error de validación de `Settings` antes de conectar

3. **Escenario**: Apagado limpio ante SIGTERM
   - **Dado** el consumidor en ejecución
   - **Cuando** recibe SIGTERM/SIGINT
   - **Entonces** cierra el canal y la conexión RabbitMQ antes de terminar

---

### Historia de Usuario 4 - Round-trip de integración (Prioridad: P3)

Un test de integración publica una solicitud en la cola real de RabbitMQ contra un
contenedor Docker, el consumidor la procesa con el `AnalysisService` (motor mockeado)
y el resultado aparece en el exchange de resultados.

**Por qué esta prioridad**: Valida la integración end-to-end pero requiere infraestructura
real. Las HU anteriores cubren la funcionalidad central; este es el sello de calidad.

**Prueba Independiente**: Requiere `docker compose up -d rabbitmq`. Se marca con
`@pytest.mark.integration` y se omite por defecto.

**Escenarios de Aceptación**:

1. **Escenario**: Round-trip feliz request → result
   - **Dado** RabbitMQ corriendo en Docker y el consumidor activo (con motor mockeado)
   - **Cuando** se publica un `AnalysisRequest` válido en la cola de solicitudes
   - **Entonces** el consumidor produce un `AnalysisResult` y lo publica en el exchange de resultados

---

### Casos Límite

- ¿Qué ocurre si RabbitMQ no está disponible al arrancar? → `main.py` deja propagar la
  excepción de conexión; el orquestador reinicia la réplica.
- ¿Qué ocurre si el body del mensaje no es JSON válido? → `InvalidMessageError` → nack sin requeue.
- ¿Qué ocurre si falla la serialización del resultado al publicar? → Se loguea el error y
  se relanza; el consumer decide si hace ack o nack según el tipo de excepción.
- ¿Qué ocurre si el canal se cierra en mitad del procesamiento? → La excepción de pika
  se propaga y el proceso reinicia (comportamiento esperado con `prefetch=1`).
- ¿El consumidor puede procesar dos mensajes en paralelo? → No. `prefetch=1` es intencional;
  se escala con réplicas.

---

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El consumidor DEBE conectar a RabbitMQ usando `RABBITMQ_URL` y declarar
  `prefetch=1` antes de empezar a consumir.
- **RF-002**: El consumidor DEBE validar el mensaje entrante usando `presentation/schemas.py`
  y lanzar `InvalidMessageError` si el payload no cumple el esquema.
- **RF-003**: El consumidor DEBE hacer `basic_nack(requeue=False)` para mensajes con
  `InvalidMessageError` o `UnknownAnalysisTypeError` (no recuperables → dead-letter).
- **RF-004**: El consumidor DEBE hacer `basic_ack` cuando la inferencia termina, ya sea
  con `status=succeeded` o `status=failed` (el resultado se publica antes del ack).
- **RF-005**: El publisher DEBE publicar el `AnalysisResult` con `delivery_mode=2`
  (persistente) al exchange `RABBITMQ_RESULT_EXCHANGE` con routing key
  `RABBITMQ_RESULT_ROUTING_KEY`.
- **RF-006**: El publisher DEBE serializar el `AnalysisResult` a JSON siguiendo el contrato
  del §9 de `PROJECT_ARCHITECTURE.md` antes de publicar.
- **RF-007**: `main.py` DEBE ser el único composition root: instancia `Settings`, logging,
  todos los componentes de infraestructura, los inyecta en `AnalysisService` y monta el
  `Consumer`.
- **RF-008**: `main.py` DEBE manejar SIGTERM/SIGINT cerrando limpiamente la conexión
  RabbitMQ.
- **RF-009**: El `Dockerfile` DEBE usar una imagen base compatible con MONAI/PyTorch y
  copiar el código fuente del layout `src/`.
- **RF-010**: `docker-compose.yml` DEBE incluir el servicio RabbitMQ con puertos expuestos
  para desarrollo local y tests de integración.

### Entidades Clave

- **`Consumer`** (`presentation/consumer.py`): componente stateless que recibe mensajes
  AMQP, valida vía schemas, delega al service y emite ack/nack. Depende de `AnalysisService`.
- **`Publisher`** (`infrastructure/publisher.py`): componente de infraestructura que
  serializa y publica `AnalysisResult` con garantías de persistencia. Recibe un canal pika
  inyectado.

---

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: `uv run pytest tests/unit/` pasa al 100% sin errores ni warnings de tipo.
- **CE-002**: `uv run mypy src` pasa sin errores con las nuevas implementaciones.
- **CE-003**: `uv run ruff format --check . && uv run ruff check .` pasan sin hallazgos.
- **CE-004**: `uv run python -m sinapsis_ai.main` arranca limpiamente con `.env` de ejemplo
  (con RabbitMQ disponible vía `docker compose up -d rabbitmq`).
- **CE-005**: El test de integración `test_rabbitmq_round_trip` pasa al ejecutar
  `uv run pytest -m integration` con RabbitMQ corriendo.
- **CE-006**: Un mensaje inválido jamás hace crash al worker; el proceso sigue consumiendo
  mensajes tras un nack.
- **CE-007**: Un mensaje con `InferenceError` produce un `AnalysisResult(status=failed)`
  publicado en el exchange (sin perder trazabilidad) y el worker sigue activo.
