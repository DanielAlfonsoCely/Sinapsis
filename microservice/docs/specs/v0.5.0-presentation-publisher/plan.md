# Plan de Implementación: Presentation + Publicación (v0.5.0)

**Fecha**: 2026-07-13
**Especificación**: [spec.md](./spec.md)

---

## Resumen

Implementar la capa de presentación completa del microservicio: el consumidor RabbitMQ
con política ack/nack/dead-letter (`presentation/consumer.py`), el publicador de
resultados persistente (`infrastructure/publisher.py`), el composition root completo
(`main.py`) y la infraestructura Docker. Esto completa el flujo end-to-end del servicio
y lo hace ejecutable como consumidor real de RabbitMQ.

---

## Contexto Técnico

**Lenguaje/Versión**: Python 3.12
**Dependencias Principales**: `pika` (AMQP), `pydantic` (DTOs), `pydantic-settings` (config),
`monai` + `torch` (inferencia, mockeados en unit tests)
**Almacenamiento**: N/A (stateless; resultados publicados a RabbitMQ)
**Testing**: `pytest` + `pytest-cov`; umbral objetivo ≥ 85% en `src/sinapsis_ai`;
canal `pika` mockeado en unit tests; `@pytest.mark.integration` omitidos por defecto
**Plataforma Objetivo**: Servidor Linux (contenedor Docker)
**Objetivos de Rendimiento**: `prefetch=1` por worker; escalar con réplicas, no con concurrencia interna
**Restricciones**: `application/` nunca importa `pika`; `main.py` es el único composition root;
no loguear PII; no hardcodear credenciales
**Escala/Alcance**: Una inferencia por worker por vez; coordinación vía RabbitMQ

---

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/v0.5.0-presentation-publisher/
├── plan.md   # Este archivo
└── spec.md
```

### Código Fuente

```text
src/sinapsis_ai/
├── main.py                          # Composition root (actualizar esqueleto v0.1.0)
├── presentation/
│   ├── consumer.py                  # NUEVO: consumidor pika prefetch=1, ack/nack
│   └── schemas.py                   # Ya existe (v0.2.0) — sin cambios
└── infrastructure/
    └── publisher.py                 # NUEVO: publicador persistente pika

tests/
├── unit/
│   ├── presentation/
│   │   ├── __init__.py              # Ya existe
│   │   └── test_consumer.py         # NUEVO: tests de consumer con pika mockeado
│   └── infrastructure/
│       └── test_publisher.py        # NUEVO: tests de publisher con pika mockeado
└── integration/
    └── test_rabbitmq_flow.py        # NUEVO: round-trip con RabbitMQ real

# Infraestructura Docker (raíz del microservicio):
docker-compose.yml                   # NUEVO: servicio RabbitMQ local
Dockerfile                           # NUEVO: imagen del servicio
```

**Decisión de Estructura**: Layout `src/` existente. Se crean exactamente los archivos
listados en CHANGELOG.md v0.5.0. No se modifican capas Domain ni Application (DIP).

---

## Fase 1: Infraestructura Docker y preparación

**Propósito**: Asegurar que `docker-compose.yml` y `Dockerfile` existen para que los
tests de integración puedan levantar RabbitMQ. Sin esto el round-trip de integración
no puede ejecutarse.

- [x] T001 Crear `docker-compose.yml` con servicio `rabbitmq` (imagen `rabbitmq:3-management`,
  puertos 5672 y 15672, healthcheck) y servicio `sinapsis-ai` comentado/condicional
- [x] T002 Crear `Dockerfile` con imagen base Python 3.12 slim, instalación vía `uv`,
  layout `src/` y entrypoint `python -m sinapsis_ai.main`

---

## Fase 2: Fundacional — Publisher (Infrastructure)

**Propósito**: El publisher es prerequisito para el consumer y para el composition root.
`AnalysisService` ya tiene la firma `ResultPublisher.publish()` definida en los ports
(v0.4.0); esta fase provee la implementación concreta de infraestructura.

**CRITICO**: El consumer (HU1) inyecta el service, que requiere el publisher. El publisher
debe existir antes de poder construir el consumer.

- [x] T003 [P] [HU2] Escribir tests unitarios en `tests/unit/infrastructure/test_publisher.py`:
  - `test_publisher_publishes_persistent_to_result_exchange` — verifica `basic_publish` con
    `delivery_mode=2`, exchange y routing key correctos, body JSON válido
  - `test_publisher_publishes_failed_result` — verifica que el bloque `error` aparece en el JSON
- [x] T004 [P] [HU2] Implementar `src/sinapsis_ai/infrastructure/publisher.py`:
  - Clase `RabbitMQPublisher` que implementa el protocolo `ResultPublisher`
  - Constructor recibe canal `pika` y `Settings` (exchange, routing key)
  - `publish()` serializa con `serialise_result()` de `presentation/schemas.py`, llama a
    `basic_publish` con `pika.BasicProperties(delivery_mode=2)`
  - Loguea `request_id`, `study_id`, `status` (sin PII)

**Punto de Control**: `uv run pytest tests/unit/infrastructure/test_publisher.py -q` pasa

---

## Fase 3: Historia de Usuario 1 — Consumidor RabbitMQ con ack/nack (P1)

**Objetivo**: `presentation/consumer.py` con política completa ack/nack/dead-letter,
delegando al `AnalysisService` inyectado.

**Prueba Independiente**: Canal `pika` mockeado + `AnalysisService` doblez; sin RabbitMQ real.

### Pruebas para HU1

- [x] T005 [P] [HU1] Escribir tests unitarios en `tests/unit/presentation/test_consumer.py`:
  - `test_consumer_valid_message_acks_and_publishes_result` — service doblez retorna
    `AnalysisResult(status=succeeded)`; verificar `basic_ack` llamado, service invocado
  - `test_consumer_invalid_message_deadletters` — payload sin `image_uri`; verificar
    `basic_nack(requeue=False)`, service NO invocado
  - `test_consumer_unknown_analysis_type_deadletters` — `analysis_type` desconocido;
    verificar `basic_nack(requeue=False)`
  - `test_consumer_inference_error_publishes_failed_result_and_acks` — service doblez
    lanza `InferenceError`; verificar que el service construye result `failed` y el consumer
    hace `basic_ack` (la publicación ocurre dentro del service)

### Implementación para HU1

- [x] T006 [P] [HU1] Implementar `src/sinapsis_ai/presentation/consumer.py`:
  - Clase `Consumer` que recibe `AnalysisService` y canal `pika` inyectados
  - Método `start_consuming(queue, prefetch)` que declara prefetch y registra el callback
  - Callback `_on_message(ch, method, properties, body)`:
    - Llama a `parse_request(body)` de `schemas.py`
    - En `InvalidMessageError` o `UnknownAnalysisTypeError` → `basic_nack(requeue=False)`, log warning
    - Llama a `service.run_analysis(request)` — el service maneja `InferenceError` internamente
    - En cualquier otra excepción no esperada → `basic_nack(requeue=False)`, log error
    - En éxito → `basic_ack`
  - No importa `monai`, `torch` ni `pika.BlockingConnection` (solo el canal inyectado)

**Punto de Control**: `uv run pytest tests/unit/presentation/test_consumer.py -q` pasa

---

## Fase 4: Historia de Usuario 3 — Composition Root y arranque limpio (P2)

**Objetivo**: `main.py` completo que instancia todos los componentes, los inyecta y
arranca el bucle de consumo con apagado limpio ante SIGTERM/SIGINT.

**Prueba Independiente**: Se verifica con `uv run python -m sinapsis_ai.main` (requiere
`.env` válido y RabbitMQ), y con el test de integración (HU4).

### Implementación para HU3

- [x] T007 [P] [HU3] Reescribir `src/sinapsis_ai/main.py` como composition root completo:
  - Importa y construye `Settings`
  - Llama a `configure_logging(settings.log_level)`
  - Crea conexión `pika.BlockingConnection` con `pika.URLParameters(settings.rabbitmq_url)`
  - Instancia `BundleRegistry`, `LocalImageStore`, `MonaiInferenceEngine`,
    `RabbitMQPublisher` (canal de la conexión), `AnalysisService` con inyección
  - Instancia `Consumer(service=..., channel=...)`
  - Registra manejadores de SIGTERM/SIGINT que llaman a `connection.close()`
  - Llama a `consumer.start_consuming(settings.rabbitmq_request_queue, settings.rabbitmq_prefetch)`
  - `channel.start_consuming()` (bucle bloqueante)
  - Log de inicio con device, queue, prefetch

---

## Fase 5: Historia de Usuario 4 — Round-trip de integración (P3)

**Objetivo**: Test end-to-end que publica un request en RabbitMQ real y verifica que el
consumidor produce y publica un result.

**Prueba Independiente**: Requiere `docker compose up -d rabbitmq`. Marcado con
`@pytest.mark.integration`.

### Pruebas para HU4

- [x] T008 [HU4] Escribir `tests/integration/test_rabbitmq_flow.py`:
  - `test_rabbitmq_round_trip`: fixture levanta conexión pika real, publica request en
    la cola de solicitudes, arranca el consumer en un thread (con `AnalysisService` donde
    el engine está mockeado para no descargar bundles), espera a que el result aparezca
    en el exchange de resultados, verifica campos del JSON, limpia colas/exchanges al terminar
  - Marcado con `@pytest.mark.integration` (omitido por defecto en `pyproject.toml`)

**Punto de Control**: `uv run pytest -m integration tests/integration/test_rabbitmq_flow.py -q`
pasa con RabbitMQ corriendo

---

## Fase 6: Acabado y verificación final

**Propósito**: Verificación completa de la suite, lint y tipos.

- [x] T009 Ejecutar `uv run ruff format --check .` y corregir si hay hallazgos
- [x] T010 Ejecutar `uv run ruff check .` y corregir si hay hallazgos
- [x] T011 Ejecutar `uv run mypy src` y corregir errores de tipo
- [x] T012 Ejecutar `uv run pytest` (solo unit) y verificar que pasan al 100%
- [x] T013 Ejecutar `uv run pytest --cov=sinapsis_ai --cov-report=term-missing` y revisar cobertura
- [x] T014 Actualizar `CHANGELOG.md`: marcar `[v0.5.0]` como `COMPLETADO` con la fecha

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Fase 1 (Docker)**: Sin dependencias — puede ejecutarse inmediatamente
- **Fase 2 (Publisher)**: Sin dependencias de código anterior — solo usa `schemas.py` y
  `domain/models.py` que ya existen
- **Fase 3 (Consumer/HU1)**: Depende de Fase 2 (publisher debe existir para poder inyectarlo)
- **Fase 4 (Composition Root/HU3)**: Depende de Fase 2 y Fase 3 (todos los componentes deben existir)
- **Fase 5 (Integración/HU4)**: Depende de Fase 1, 3 y 4 (necesita Docker + consumer + composition root)
- **Fase 6 (Acabado)**: Depende de todas las fases anteriores

### Dependencias entre Historias de Usuario

- **HU1 (Consumer)**: Depende de que Publisher exista (Fase 2) para poder inyectarlo en el service
- **HU2 (Publisher)**: Solo depende de `schemas.py` y `domain/` ya existentes — testeable independientemente
- **HU3 (Composition Root)**: Depende de HU1 y HU2 — solo integra, no aporta lógica nueva
- **HU4 (Integración)**: Depende de HU1, HU2, HU3 y Docker (Fase 1)

### Dentro de Cada Historia

- Tests primero (para verificar contrato esperado)
- Implementación después
- Verificar lint/tipos al completar cada fase

---

## Ejecución Completada

**Fecha**: 2026-07-13
**Estado**: Todas las fases implementadas y verificadas

- Tests: 80 passing (unit), 2 deselected (integration)
- Cobertura: 86.13% (umbral: 85%)
- Lint: OK, sin errores
- Types: OK, sin errores

---

## Notas

- `[P]` = tarea prioritaria para la historia (tests antes que implementación)
- `[HU#]` = mapeo a historia de usuario del spec
- El `AnalysisService` (v0.4.0) ya maneja `InferenceError` internamente y publica el result
  `failed`; el consumer solo necesita hacer `basic_ack` en ese caso
- `pika.BlockingConnection` solo se instancia en `main.py`; el consumer y el publisher
  reciben el canal ya construido (facilita el testing con dobles)
- La integración usa motor mockeado para no requerir descarga de bundles MONAI
- Los cambios NO se commitean automáticamente
