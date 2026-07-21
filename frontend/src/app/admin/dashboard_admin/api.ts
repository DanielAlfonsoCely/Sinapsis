const BASE_URL = "http://localhost:8080/api/v1"

export type Gravedad = "CRITICAL" | "HIGH" | "WARNING" | "INFORMATIVE"

export interface AuditLogEntry {
  id: string
  usuario_id: string
  usuario_nombre: string
  usuario_email: string
  tipo_operacion: string
  tabla_afectada: string
  registro_id: string | null
  ip_origen: string | null
  detalles: string | null
  fecha_operacion: string
  gravedad: Gravedad
}

export interface AuditoriaCriticalResponse {
  registros: AuditLogEntry[]
  total: number
  limit: number
  offset: number
}

export interface AdminEntidadListItem {
  id: string
  nombre_entidad: string
  tipo_entidad: string
  nit: string
  ciudad: string
  estado: boolean
  fecha_creacion: string
}

export interface AdminEntidadListResponse {
  entidades: AdminEntidadListItem[]
  total: number
}

export interface UsuariosTotalesResponse {
  total: number
}

function getToken() {
  return document.cookie
    .split("; ")
    .find((c) => c.startsWith("token="))
    ?.split("=")[1]
}

function authHeaders() {
  return { Authorization: `Bearer ${getToken()}` }
}

function handleAuthAndErrors(res: Response, path: string) {
  if (res.status === 403) {
    window.location.href = "/login"
    throw new Error("forbidden")
  }
  if (!res.ok) throw new Error(`Error ${res.status} en ${path}`)
}

export async function fetchAuditoriaCritical(
  limit = 5,
  offset = 0
): Promise<AuditoriaCriticalResponse> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  })
  const path = `/admin/auditoria/critical?${params}`
  const res = await fetch(`${BASE_URL}${path}`, { headers: authHeaders() })
  handleAuthAndErrors(res, path)
  return res.json() as Promise<AuditoriaCriticalResponse>
}

export async function fetchEntidadesAdmin(
  q = ""
): Promise<AdminEntidadListResponse> {
  const params = new URLSearchParams({ ...(q && { q }) })
  const path = `/admin/entidades?${params}`
  const res = await fetch(`${BASE_URL}${path}`, { headers: authHeaders() })
  handleAuthAndErrors(res, path)
  return res.json() as Promise<AdminEntidadListResponse>
}

export async function fetchUsuariosTotales(): Promise<UsuariosTotalesResponse> {
  const path = `/admin/usuarios/totales`
  const res = await fetch(`${BASE_URL}${path}`, { headers: authHeaders() })
  handleAuthAndErrors(res, path)
  return res.json() as Promise<UsuariosTotalesResponse>
}