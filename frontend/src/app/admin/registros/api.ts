import{ API_URL } from "@/config/constants"
const BASE_URL = `${API_URL}/api/v1`

export type TipoOperacion =
  | "crear"
  | "actualizar"
  | "eliminar"
  | "consultar"
  | "exportar"
  | "cambiar_permisos"
  | "usar_ia"

// Forma en que llega del backend
export interface AuditLogEntry {
  id: string
  usuario_id: string
  usuario_nombre: string
  usuario_email: string
  tipo_operacion: TipoOperacion
  tabla_afectada: string
  registro_id: string | null
  ip_origen: string | null
  detalles: string | null
  fecha_operacion: string
  gravedad: string | null
}

export interface ListAuditoriaResponse {
  registros: AuditLogEntry[]
  total: number
  limit: number
  offset: number
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

export async function fetchAuditoria(
  limit: number,
  offset: number
): Promise<ListAuditoriaResponse> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  })
  const path = `/admin/auditoria?${params}`

  const res = await fetch(`${BASE_URL}${path}`, { headers: authHeaders() })

  if (res.status === 403) {
    window.location.href = "/login"
    throw new Error("forbidden")
  }
  if (!res.ok) throw new Error(`Error ${res.status} en ${path}`)

  return res.json() as Promise<ListAuditoriaResponse>
}

// frontend/src/app/admin/registros/api.ts
// ... (todo lo existente igual, solo agrega esto al final)

export async function exportToCSV(rows: Record<string, string>[], filename: string) {
  //Envía un registro de auditoría para la exportación
  
  await fetch(`${BASE_URL}/admin/auditoria`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(),
    },
    body: JSON.stringify({
      tabla_afectada: "auditoria",
      detalles: `Exportación de registros de auditoría a CSV`,
    }),
  }).catch((err) => console.error("Error al registrar auditoría:", err))

  if (rows.length === 0) return
  const headers = Object.keys(rows[0])
  const escape = (value: string) => {
    const needsQuotes = /[",\n]/.test(value)
    const escaped = value.replace(/"/g, '""')
    return needsQuotes ? `"${escaped}"` : escaped
  }

  const csvLines = [
    headers.join(","),
    ...rows.map((row) => headers.map((h) => escape(row[h] ?? "")).join(",")),
  ]

  const csvContent = "\uFEFF" + csvLines.join("\r\n") // BOM para acentos en Excel
  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" })
  const url = URL.createObjectURL(blob)

  const link = document.createElement("a")
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)

  URL.revokeObjectURL(url)
}