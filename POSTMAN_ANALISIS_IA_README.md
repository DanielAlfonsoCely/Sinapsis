# Colección Postman - Módulo Análisis IA

Esta colección permite probar los **3 endpoints** del módulo de análisis con IA integrado con el microservicio MONAI.

## 📋 Archivos

- `Sinapsis_Analisis_IA.postman_collection.json` - Colección completa de Postman

## 🚀 Configuración Inicial

### 1. Importar la Colección

1. Abre Postman
2. Click en **Import** (Ctrl+O / Cmd+O)
3. Selecciona el archivo `Sinapsis_Analisis_IA.postman_collection.json`
4. Click en **Import**

### 2. Configurar Variables de Entorno

La colección utiliza las siguientes variables que debes configurar:

| Variable | Valor | Descripción |
|----------|-------|-------------|
| `base_url` | `http://localhost:8080` | URL del backend (ajusta si es diferente) |
| `token` | `<tu_token_jwt>` | Token JWT obtenido al autenticarse |
| `examen_id` | `<uuid>` | ID del examen para solicitar análisis |
| `sugerencia_ia_id` | Auto-poblada | ID generado por el primer endpoint |
| `request_id` | Auto-poblada | ID de solicitud RabbitMQ |

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

## 🔐 Obtener Token JWT

Antes de ejecutar los endpoints, necesitas autenticarte:

1. Usa tu endpoint de login (ej: `POST /api/v1/login`)
2. Copia el token JWT de la respuesta
3. Pégalo en la variable `token` de tu entorno/colección

## 📝 Flujo de Pruebas

### Flujo Completo (Recomendado)

```
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

---

## 🧪 Casos de Prueba

### Caso 1: Flujo Exitoso Completo

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

- **Backend Models:** `backend/models/analisis_ia.go`
- **Handler:** `backend/handlers/analisis_ia_handler.go`
- **Routes:** `backend/routes/routes.go`
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
