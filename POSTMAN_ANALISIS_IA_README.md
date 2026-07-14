# Colección Postman - Módulo Análisis IA

Esta colección permite probar los **3 endpoints** del módulo de análisis con IA integrado con el microservicio MONAI, más un endpoint de **Login** para obtener el token JWT.

## 📋 Archivos

- `Sinapsis_Analisis_IA.postman_collection.json` - Colección completa de Postman

## 🚀 Configuración Inicial

### 1. Importar la Colección

1. Abre Postman
2. Click en **Import** (Ctrl+O / Cmd+O)
3. Selecciona el archivo `Sinapsis_Analisis_IA.postman_collection.json`
4. Click en **Import**

### 2. Configurar Variables de Entorno

La colección utiliza las siguientes variables que debes configurar. Los valores por defecto ya apuntan a los **puertos expuestos por docker-compose.yml** en la raíz del proyecto:

| Variable | Valor por defecto | Descripción |
|----------|-------|-------------|
| `base_url` | `http://localhost:8080` | URL del backend (puerto `8080:8080` mapeado en `docker-compose.yml`) |
| `medico_email` | `medico@sinapsis.com` | Email del médico para el endpoint de Login |
| `medico_password` | `changeme` | Contraseña del médico para el endpoint de Login |
| `token` | Auto-poblada | Token JWT (se guarda automáticamente al ejecutar el request de Login) |
| `examen_id` | `<uuid>` | ID del examen para solicitar análisis (debes completarlo manualmente) |
| `sugerencia_ia_id` | Auto-poblada | ID generado por el endpoint "Solicitar Análisis IA" |
| `request_id` | Auto-poblada | ID de solicitud RabbitMQ |

**Otros puertos relevantes de docker-compose.yml (por si necesitas depurar):**

| Servicio | Puerto host | Uso |
|----------|-------------|-----|
| `backend` | `8080` | API REST (usado como `base_url`) |
| `postgres` | `5432` | Base de datos PostgreSQL |
| `rabbitmq` | `5672` | Cola AMQP (comunicación backend ↔ microservicio IA) |
| `rabbitmq` | `15672` | Management UI: http://localhost:15672 (guest/guest) |
| `frontend` | `3000` | Aplicación Next.js |

#### Opción A: Variables de Entorno en Postman

1. Click en el ícono de ojos (Environment selector) en la esquina superior derecha
2. Click en **Create Environment**
3. Nombra el entorno `Sinapsis-Dev`
4. Agrega las variables anteriores
5. Click en **Save**
6. Selecciona el entorno de la lista

#### Opción B: Variables de Colección

Si prefieres, puedes editar directamente en la colección:

1. Selecciona la colección "Sinapsis - Análisis IA"
2. Click en la pestaña **Variables**
3. Completa los valores (excepto `sugerencia_ia_id` y `request_id` que se auto-populan)

## 🔐 Obtener Token JWT (Login)

La colección incluye un request **"0. Login (Obtener Token)"** que se ejecuta antes que los demás. Al correrlo en Postman, su script de test guarda el token automáticamente en la variable `{{token}}`, así que **no necesitas copiarlo manualmente**.

### Endpoint de Login

**POST** `/api/v1/auth/login`

**Request Body:**
```json
{
  "email": "medico@sinapsis.com",
  "contrasena": "tu_password"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "usuario": {
    "id": "uuid",
    "nombre_usuario": "...",
    "apellidos": "...",
    "email": "...",
    "tipo_usuario": "medico",
    "especialidad": "...",
    "entidad": "..."
  }
}
```

### Curl equivalente (contra el backend en Docker, puerto 8080)

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "medico@sinapsis.com",
    "contrasena": "tu_password"
  }'
```

**Extraer solo el token con `jq` (útil para exportar como variable de shell):**
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "medico@sinapsis.com", "contrasena": "tu_password"}' \
  | jq -r '.token')

echo "Token: $TOKEN"
```

**Errores posibles:**
- `400` - Body inválido (falta `email` o `contrasena`, o email mal formado)
- `401` - Email o contraseña incorrectos

## 📝 Flujo de Pruebas

### Flujo Completo (Recomendado)

```
0️⃣  Login (POST)
    ↓
    🔑 Guarda token automáticamente
    ↓
1️⃣  Solicitar Análisis IA (POST)
    ↓
    ✅ Genera sugerencia_ia_id automáticamente
    ↓
2️⃣  Consultar Estado/Resultado (GET) - REPETIR HASTA COMPLETAR
    ↓
    ⏳ Muestra estado: "enviado" → "procesando" → "completado"
    ↓
3️⃣  Revisar Análisis IA (PATCH)
    ↓
    ✅ Marca como "revisada" o "rechazada"
```

### Detalles de Cada Endpoint

#### 1️⃣ **POST** `/api/v1/examenes/:id/analisis-ia`

**Solicita análisis IA del microservicio MONAI**

**Precondiciones:**
- Debes estar autenticado (`token` válido)
- El examen debe existir y pertenecer al médico
- El examen debe tener imagen asociada
- La consulta vinculada debe tener pre-diagnóstico registrado (RF-12/RN-007)

**Tipos de análisis soportados:**
```json
{
  "analysis_type": "ct_spleen_segmentation"
}
```

Opciones:
- `ct_spleen_segmentation` - Segmentación de bazo
- `ct_lung_nodule_detection` - Detección de nódulos pulmonares
- `mri_brain_tumor_segmentation` - Segmentación de tumor cerebral
- `xr_breast_density_classification` - Clasificación de densidad mamaria

**Response (202 Aceptado):**
```json
{
  "id": "uuid-de-sugerencia",
  "request_id": "uuid-de-solicitud-rabbitmq",
  "estado": "enviado"
}
```

### Curl equivalente

```bash
curl -X POST http://localhost:8080/api/v1/examenes/<EXAMEN_ID>/analisis-ia \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "analysis_type": "ct_spleen_segmentation"
  }'
```

**Posibles Errores:**
- `400` - Invalid analysis_type
- `401` - Unauthorized (token inválido)
- `403` - Usuario no es médico
- `404` - Examen no existe
- `422` - Examen sin imagen o sin pre-diagnóstico

---

#### 2️⃣ **GET** `/api/v1/sugerencias-ia/:id`

**Consulta estado y resultados del análisis IA**

**Comportamiento según RF-12/RN-007:**

Si hay pre-diagnóstico registrado:
```json
{
  "id": "uuid",
  "estado_procesamiento": "completado",
  "modelo_ia_utilizado": "ct_spleen_segmentation",
  "pre_diagnostico_registrado": true,
  "confianza_prediccion": 0.95,
  "descripcion_hallazgo": "Bazo de tamaño normal con...",
  "diagnostico_sugerido": "Sin hallazgos significativos",
  "metricas": { "dice_coefficient": 0.92 },
  "fecha_analisis": "2024-01-15T10:30:00Z",
  "estado_revision": "pendiente"
}
```

Si NO hay pre-diagnóstico registrado (campos clínicos ocultos):
```json
{
  "id": "uuid",
  "estado_procesamiento": "completado",
  "modelo_ia_utilizado": "ct_spleen_segmentation",
  "pre_diagnostico_registrado": false,
  "confianza_prediccion": null,
  "descripcion_hallazgo": null,
  "diagnostico_sugerido": null,
  "estado_revision": "pendiente"
}
```

**Posibles estados de procesamiento:**
- `enviado` - En cola RabbitMQ, esperando procesamiento
- `procesando` - Microservicio ejecutando análisis
- `completado` - Análisis exitoso
- `fallido` - Error durante procesamiento

**Polling Strategy:**
```
Intervalo sugerido: 2-5 segundos
Timeout máximo: 5-10 minutos (depende del modelo)
```

### Curl equivalente

```bash
curl -X GET http://localhost:8080/api/v1/sugerencias-ia/<SUGERENCIA_ID> \
  -H "Authorization: Bearer $TOKEN"
```

**Polling con curl (repetir cada pocos segundos):**
```bash
watch -n 3 "curl -s http://localhost:8080/api/v1/sugerencias-ia/<SUGERENCIA_ID> \
  -H 'Authorization: Bearer $TOKEN' | jq '.estado_procesamiento'"
```

---

#### 3️⃣ **PATCH** `/api/v1/sugerencias-ia/:id/revision`

**Marca análisis como revisado/rechazado**

**Precondiciones:**
- El análisis debe estar `completado`
- Solo el médico solicitante puede revisar

**Request:**
```json
{
  "estado_revision": "revisada",
  "observaciones_medico": "Resultados consistentes con evaluación clínica"
}
```

**Estados válidos:**
- `revisada` - Análisis aprobado
- `rechazada` - Análisis rechazado (puede requerir re-análisis)

**Response (200 OK):**
```json
{
  "id": "uuid",
  "estado_revision": "revisada"
}
```

### Curl equivalente

```bash
curl -X PATCH http://localhost:8080/api/v1/sugerencias-ia/<SUGERENCIA_ID>/revision \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "estado_revision": "revisada",
    "observaciones_medico": "Resultados consistentes con evaluacion clinica"
  }'
```

---

## 🧪 Casos de Prueba

### Caso 1: Flujo Exitoso Completo (con curl y jq)

Script completo usando `curl` + `jq` para encadenar los 4 endpoints en la terminal (asume backend en Docker, puerto `8080`):

```bash
BASE_URL="http://localhost:8080"
EXAMEN_ID="<uuid-del-examen>"

# 0. Login - obtener token
TOKEN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "medico@sinapsis.com", "contrasena": "tu_password"}' \
  | jq -r '.token')
echo "Token obtenido: ${TOKEN:0:20}..."

# 1. Solicitar análisis
SUGERENCIA_ID=$(curl -s -X POST "$BASE_URL/api/v1/examenes/$EXAMEN_ID/analisis-ia" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"analysis_type": "ct_spleen_segmentation"}' \
  | jq -r '.id')
echo "Sugerencia IA solicitada. ID: $SUGERENCIA_ID"

# 2. Consultar estado (repetir hasta "completado")
curl -s "$BASE_URL/api/v1/sugerencias-ia/$SUGERENCIA_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# 3. Revisar análisis (una vez completado)
curl -s -X PATCH "$BASE_URL/api/v1/sugerencias-ia/$SUGERENCIA_ID/revision" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"estado_revision": "revisada", "observaciones_medico": "Aprobado"}' \
  | jq '.'
```

**Resultado esperado paso a paso:**

```bash
# 1. Solicitar análisis
POST /api/v1/examenes/examen-123/analisis-ia
Body: { "analysis_type": "ct_spleen_segmentation" }
# Response: 202 - id generado

# 2. Esperar procesamiento
GET /api/v1/sugerencias-ia/sugerencia-123
# Response: 200 - estado_procesamiento: "procesando"

# 3. Esperar a que complete
GET /api/v1/sugerencias-ia/sugerencia-123
# Response: 200 - estado_procesamiento: "completado"
#           - confianza_prediccion: 0.95
#           - diagnostico_sugerido: "..."

# 4. Revisar análisis
PATCH /api/v1/sugerencias-ia/sugerencia-123/revision
Body: { 
  "estado_revision": "revisada",
  "observaciones_medico": "Aprobado" 
}
# Response: 200 - estado_revision: "revisada"
```

### Caso 2: Sin Pre-diagnóstico (RF-12/RN-007)

```bash
# 1. Solicitar análisis (si falta pre-diagnóstico)
# Response: 422 - "Consulta sin pre-diagnóstico"

# Después de registrar pre-diagnóstico en consulta:

# 2. Solicitar nuevamente
POST /api/v1/examenes/examen-123/analisis-ia
# Response: 202 - OK

# 3. Consultar - NOTA: campos clínicos ocultos
GET /api/v1/sugerencias-ia/sugerencia-123
# Response: 200 - pre_diagnostico_registrado: false
#           - confianza_prediccion: null
#           - diagnostico_sugerido: null
```

### Caso 3: Análisis Rechazado

```bash
# 1-3. Pasos 1-3 iguales al Caso 1

# 4. Rechazar análisis
PATCH /api/v1/sugerencias-ia/sugerencia-123/revision
Body: { 
  "estado_revision": "rechazada",
  "observaciones_medico": "Requiere re-análisis con otra imagen" 
}
# Response: 200 - estado_revision: "rechazada"
```

## 🔍 Troubleshooting

### ❌ "401 Unauthorized"
- Verifica que el `token` sea válido
- Asegúrate de que el token no haya expirado
- Vuelve a autenticarte si es necesario

### ❌ "404 Not Found"
- Verifica que el `examen_id` existe
- El examen debe pertenecerle al médico autenticado
- Verifica que el `sugerencia_ia_id` sea correcto

### ❌ "422 Unprocessable Entity"
- El examen debe tener imagen asociada
- La consulta debe tener pre-diagnóstico registrado (RF-12/RN-007)
- El tipo de análisis debe ser uno de los 4 soportados

### ⏳ "Procesando por mucho tiempo"
- Los tiempos de procesamiento varían según:
  - Tipo de modelo (tamaño de imagen)
  - Carga del microservicio
  - Disponibilidad de GPU
- Típicamente: 30 segundos - 5 minutos

### ❌ "Estado: fallido"
- Revisa los logs del microservicio MONAI
- Verifica que la imagen sea válida para el modelo
- Comprueba que RabbitMQ esté disponible

## 📚 Referencias

- **Backend Models (Análisis IA):** `backend/models/analisis_ia.go`
- **Backend Models (Login):** `backend/models/usuario.go`
- **Handler (Análisis IA):** `backend/handlers/analisis_ia_handler.go`
- **Handler (Login):** `backend/handlers/auth.go`
- **Routes:** `backend/routes/routes.go`
- **Docker Compose:** `docker-compose.yml` (puertos de todos los servicios)
- **PROJECT_ARCHITECTURE:** Contrato exacto con microservicio MONAI (§9)
- **RF-12/RN-007:** Regla de negocio sobre pre-diagnóstico

## 💡 Tips

1. **Guarda requests exitosas** para referencia futura
2. **Usa Postman Tests** para validar responses automáticamente
3. **Monitorea RabbitMQ** si tienes acceso para debuggear
4. **Mantén logs** del backend para investigar errores
5. **Prueba cada tipo de análisis** para cobertura completa

---

**Última actualización:** 2026-07-14  
**Versión API:** v1  
**Estado:** Listo para pruebas
