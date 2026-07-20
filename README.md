# Sinapsis

**Sistema Inteligente de Análisis de Patrones de Salud Integrados**

Plataforma unificada de historia clínica electrónica para el sector salud colombiano. Centraliza el registro de pacientes, la programación de citas, las fórmulas médicas y el historial de consultas en un solo entorno, e integra un microservicio de análisis de imágenes médicas con IA (MONAI).

Proyecto final — Software Engineering II, Universidad Nacional de Colombia (Sede Bogotá), 2026.

**Equipo Error 418**
- Bermúdez Guaqueta Tomás Alejandro
- Cely Infante Daniel Alfonso
- Gámez Ariza Juan Sebastián
- Herrera Novoa David Alejandro
- Diosa Benavides Adrian Alberto

Docente: Liliana Marcela Olarte Mesa

---

## Tabla de contenido

- [Descripción del sistema](#descripción-del-sistema)
- [Alcance del proyecto](#alcance-del-proyecto)
- [Arquitectura](#arquitectura)
- [Stack tecnológico](#stack-tecnológico)
- [Patrones de diseño](#patrones-de-diseño)
- [Estructura del repositorio](#estructura-del-repositorio)
- [Requisitos previos](#requisitos-previos)
- [Variables de entorno](#variables-de-entorno)
- [Cómo ejecutar el proyecto](#cómo-ejecutar-el-proyecto)
- [API](#api)
- [Testing](#testing)
- [Notas importantes](#notas-importantes)

---

## Descripción del sistema

Sinapsis está construido alrededor de dos roles principales con interfaz dedicada:

- **Médico (`medico`)** — dashboard clínico para gestionar pacientes, registrar consultas, emitir fórmulas, autorizar remisiones a especialistas y ver su agenda diaria.
- **Administrador de plataforma (`admin_plataforma`)** — gestiona entidades de salud, usuarios de la plataforma y la bitácora de auditoría.

Existen dos roles adicionales a nivel de datos, sin interfaz dedicada en esta versión:

- **Paciente (`paciente`)** — creado automáticamente al registrar un paciente nuevo; pensado para un futuro portal donde el paciente vea sus propias citas e historia clínica.
- **Administrador de entidad (`admin_entidad`)** — reservado para gestión administrativa a nivel de entidad de salud.

El sistema también integra un **microservicio de análisis de imágenes médicas con MONAI** (*Medical Open Network for AI*, framework sobre PyTorch). Opera como un consumidor *stateless* de RabbitMQ: recibe solicitudes de análisis publicadas por el backend en Go, ejecuta inferencia con *bundles* pre-entrenados del MONAI Model Zoo (segmentación de bazo en TC, segmentación de tumores cerebrales en RM, clasificación de densidad mamaria en rayos X, entre otros) y publica el resultado de forma asíncrona.

El microservicio **no toma decisiones clínicas**: la autorización, la validación de pre-diagnóstico y la auditoría ocurren en el backend antes de que la solicitud llegue a la cola. Si el servicio de IA no está disponible, las solicitudes se acumulan en RabbitMQ y el resto de la plataforma sigue funcionando con normalidad.

## Alcance del proyecto

**Incluido:**
- Autenticación con control de acceso basado en roles (JWT)
- Registro de pacientes con creación atómica de usuario, perfil e historia clínica
- Gestión de historia clínica (consultas, fórmulas médicas, remisiones, anexos)
- Agendamiento de citas entre pacientes y médicos
- Lógica de triage — médicos con especialidad de triage pueden ver todos los pacientes de su entidad
- Gestión de entidades de salud (crear, listar) vía panel de administración
- Bitácora de auditoría inmutable (tabla `bitacora_auditoria` con protección a nivel de trigger)
- Vista de gestión de usuarios de la plataforma
- Microservicio de imágenes médicas con MONAI (worker de inferencia *stateless*)

## Arquitectura

```
Frontend (SSR / BFF)
        │
        ▼
   Backend (Go) ───► Cola RabbitMQ ───► Microservicio IA (MONAI)
        │
        ▼
 PostgreSQL + Auditoría
```

El proyecto se compone de **tres subsistemas independientes**: un backend en Go, un frontend en Next.js y un microservicio de inferencia en Python. El backend publica solicitudes de análisis de imagen en RabbitMQ y el microservicio MONAI las consume y responde de forma asíncrona, sin bloquear la petición HTTP original.

## Stack tecnológico

| Componente | Tecnología | Motivo |
|---|---|---|
| Backend — lenguaje | Go 1.26 | Concurrencia nativa (goroutines), tipado estático y compilación a binario único |
| Backend — framework HTTP | Gin | Router minimalista sobre `net/http`, middlewares encadenables |
| Backend — acceso a datos | pgx/v5 (pgxpool) | Driver nativo de PostgreSQL con control fino sobre UUID, JSONB, enums |
| Backend — autenticación | golang-jwt v5 + bcrypt | Sesiones sin estado, hashing de contraseñas con costo configurable |
| Backend — mensajería | RabbitMQ (amqp091-go) | Desacopla el backend del microservicio de IA |
| Backend — generación de PDF | maroto/v2, pdfcpu | Exportación de historia clínica (Resolución 1995 de 1999) |
| Base de datos | PostgreSQL | UUID nativo, JSONB, ENUM; garantías ACID |
| Frontend — framework | Next.js 16 (App Router) + React 19 | Rutas agrupadas por rol, Server Components |
| Frontend — lenguaje | TypeScript | Tipado estático al consumir la API |
| Frontend — estilos | Tailwind CSS 4 | CSS utilitario + clsx/tailwind-merge |
| Frontend — componentes base | Radix UI + lucide-react | Primitivas accesibles + iconografía |
| Microservicio IA — lenguaje | Python 3.12 | Ecosistema de deep learning para imágenes médicas |
| Microservicio IA — inferencia | MONAI + PyTorch + torchvision | Framework especializado en imágenes médicas (NIfTI vía nibabel) |
| Microservicio IA — mensajería | pika | Cliente RabbitMQ en Python |
| Microservicio IA — validación | Pydantic v2 + pydantic-settings | Tipado de mensajes de la cola y configuración |

## Patrones de diseño

| Patrón | Dónde se usa | Propósito |
|---|---|---|
| Repository | `backend/db/repositories/` | Aísla el acceso a PostgreSQL por dominio |
| Arquitectura en capas | `handlers → services → repositories → db` | Separa HTTP, reglas de negocio y persistencia |
| Observer | `audit/observer.go`, `audit/db_observer.go` | Notifica eventos auditables sin acoplar quién los genera de quién los persiste |
| Middleware (chain of responsibility) | `middleware/auth.go` | Valida JWT y rol antes de llegar al handler |
| Producer–Consumer / Pub-Sub | `backend/queue/` ↔ `microservice/presentation/consumer.go` | Desacopla el tiempo de respuesta HTTP de la inferencia del modelo |
| Arquitectura hexagonal (Ports & Adapters) | `microservice/src/sinapsis_ai/` | El dominio de inferencia no depende de RabbitMQ ni infraestructura concreta |
| Strategy | `infrastructure/bundle_adapters.py` (`BundleConfigAdapter`) | Cada tipo de análisis (bazo, pulmón, cerebro, mama) tiene su propio adaptador |
| Inyección de dependencias | `application/analysis_service.py` (`AnalysisService`) | Las dependencias reales solo se conectan en `main.py` (composition root) |
| Component-based architecture | `frontend/src/components/ui/`, `frontend/src/app/` | UI a partir de componentes reutilizables; App Router organizado por rol |

## Estructura del repositorio

```
Sinapsis/
├── backend/
│   ├── audit/            # Observer: Publisher + DBAuditObserver
│   ├── config/            # Carga de configuración (.env)
│   ├── db/
│   │   ├── repositories/   # Patrón Repository por dominio
│   │   ├── db.go
│   │   └── query_builder.go
│   ├── handlers/          # Controladores HTTP (Gin)
│   ├── middleware/         # RequireAuth, RequireRole (JWT)
│   ├── models/             # Structs de dominio
│   ├── queue/               # Productor/consumidor RabbitMQ
│   ├── routes/               # Registro de rutas por rol/prefijo
│   ├── services/              # Reglas de negocio
│   ├── Dockerfile
│   ├── go.mod / go.sum
│   └── main.go               # Entry point: wiring y arranque
├── frontend/
│   ├── public/
│   └── src/
│       ├── app/
│       │   ├── (auth)/         # login, mfa
│       │   ├── (dashboard)/    # agenda, análisis-ia, auditoría, consulta, pacientes...
│       │   ├── admin/           # dashboard_admin, entidades/[id], usuarios
│       │   └── paciente/         # vistas del rol paciente
│       ├── components/
│       │   ├── layout/
│       │   └── ui/
│       └── lib/
├── microservice/
│   ├── src/sinapsis_ai/
│   │   ├── domain/          # Modelos y errores de inferencia (sin dependencias externas)
│   │   ├── application/     # Casos de uso y contratos (ports.py)
│   │   ├── infrastructure/  # Adaptadores concretos: RabbitMQ, registro de modelos, MONAI
│   │   └── presentation/    # Consumidor de cola, esquemas de mensajes, política de reintentos
│   ├── docs/specs/          # Especificaciones versionadas
│   ├── tests/
│   └── pyproject.toml
├── Documentos/               # Diagramas ER y de clases en alta resolución
├── docker-compose.yml
├── schema.sql
└── init.sql
```

## Requisitos previos

- Docker Engine 24+ y Docker Compose v2 (probado con Docker 29.6 / Compose 5.3)
- Go 1.26+ (solo para desarrollo local del backend o `go test` fuera de contenedor)
- Node.js 22+ y npm (solo para desarrollo local del frontend fuera de contenedor)
- No se requiere instalación local de PostgreSQL ni RabbitMQ — ambos corren como contenedores del `docker-compose.yml` raíz

## Variables de entorno

El backend lee su configuración desde `backend/.env` (no versionado). Al levantar el stack completo con `docker-compose.yml`, estos valores se inyectan directamente como variables de entorno del contenedor.

| Variable | Descripción |
|---|---|
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Conexión a PostgreSQL |
| `JWT_SECRET` | Secreto de firma de tokens — debe cambiarse en cualquier despliegue real |
| `SERVER_PORT` | Puerto del servidor HTTP en Go (por defecto 8080) |
| `UPLOADS_DIR` | Ruta del sistema de archivos para los anexos |
| `RABBITMQ_URL`, `RABBITMQ_REQUEST_QUEUE`, `RABBITMQ_RESULT_EXCHANGE`, `RABBITMQ_RESULT_ROUTING_KEY`, `RABBITMQ_RESULT_QUEUE` | Conexión y topología del flujo asíncrono con MONAI |

El microservicio tiene su propio `microservice/.env.example` con `RABBITMQ_*`, `BUNDLE_CACHE_DIR`, `BUNDLE_SOURCE`, `ALLOWED_ANALYSIS_TYPES`, `MODEL_DEVICE` e `IMAGE_STORAGE_BACKEND` — usado solo al ejecutar ese servicio de forma independiente; en el `docker-compose.yml` raíz estas variables van directamente en línea.

> ⚠️ `JWT_SECRET` y las credenciales de Postgres/RabbitMQ en `docker-compose.yml` son valores de desarrollo (`changeme_in_production`, `sinapsis123`, `guest/guest`) y deben reemplazarse antes de cualquier despliegue real.

## Cómo ejecutar el proyecto

```bash
docker compose up --build -d
```

**Notas:**
- El esquema y los datos semilla (`schema.sql`, `init.sql`) solo se cargan automáticamente en la primera creación del volumen `pgdata`. Si el esquema cambia después, hay que eliminar el volumen (`docker compose down -v`) y reconstruir.
- Volúmenes con nombre que persisten entre reinicios: `pgdata` (base de datos), `uploads` (anexos clínicos, compartido entre backend y microservicio de IA) y `bundle_cache` (pesos de MONAI descargados).
- El primer arranque del servicio `sinapsis-ai` es lento porque descarga los *bundles* de MONAI habilitados antes de empezar a consumir la cola. Mientras tanto, las solicitudes de análisis se acumulan en RabbitMQ sin afectar al resto de la plataforma.
- Tras cambios en el código: `docker compose up --build` (no hay *hot reload*, son builds multi-stage tipo producción).

## API

Todas las rutas están prefijadas con `/api/v1`. La autenticación usa **JWT firmado con HS256**, emitido en el login, con expiración de 24 horas y claims `user_id`, `email` y `tipo_usuario`.

```
Authorization: Bearer <token>
```

Endpoints principales:

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| POST | `/auth/login` | — | Autenticación, emite JWT |
| POST | `/auth/register` | — | Registro de una nueva cuenta |
| GET | `/pacientes` | doctor | Lista pacientes visibles para el médico autenticado |
| POST | `/pacientes` | doctor | Registra un paciente (usuario + perfil + historia clínica en una transacción) |
| GET | `/pacientes/:id/historia-clinica/pdf` | any | Exporta la historia clínica como PDF |
| POST | `/consultas` | doctor | Crea un registro de consulta |
| POST | `/citas` | patient | Agenda una nueva cita |
| POST | `/examenes/:id/analisis-ia` | any | Solicita análisis de IA sobre una imagen (async vía RabbitMQ) |
| GET | `/sugerencias-ia/:id` | any | Consulta el resultado del análisis de IA |
| GET/POST/PUT/DELETE | `/admin/usuarios` | platform_admin | Gestión de usuarios de la plataforma |
| GET | `/admin/auditoria` | platform_admin | Lista la bitácora de auditoría |
| GET | `/health` | — | *Liveness probe* — `{"status": "ok"}` |

La colección completa de Postman con ejemplos de request/response está incluida en el repositorio.

## Testing

Las pruebas son pruebas unitarias sin dependencia real de base de datos: cada handler se instancia con una conexión `nil *sql.DB`, así que solo se ejercitan los caminos de código que resuelven antes de tocar la base de datos (middleware de autenticación, validación de request, chequeos tempranos de autorización). Las reglas de negocio que dependen de datos persistidos quedan como trabajo futuro de pruebas de integración contra una instancia real de Postgres.

```bash
cd backend/
go test ./handlers/ -v
```

## Notas importantes

- El diseño se basa en el estándar **HL7 FHIR** y en historias de usuario previas del sector salud en Colombia.
- La auditoría se registra en cada acceso, sea exitoso o fallido.
- El acceso al sistema es por rol; un médico también puede tener una cuenta de paciente independiente.
- Diagramas ER y de clases en mayor resolución disponibles en `Sinapsis/Documentos/DiagramaDeClases.jpg` y `Sinapsis/Documentos/ModeloRelacional.jpg`.
