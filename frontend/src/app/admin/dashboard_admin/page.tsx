"use client"

import { useEffect, useState } from "react"
import { Users, Building2, ShieldAlert, Activity } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { StatCard } from "@/components/ui/stat-card"
import {
  fetchAuditoriaCritical,
  fetchEntidadesAdmin,
  fetchUsuariosTotales,
  AuditoriaCriticalResponse,
  AdminEntidadListResponse,
  UsuariosTotalesResponse,
  Gravedad,
} from "./api"

const NODE_STATUS = [
  { name: "Central General IPS", lat: "0.04ms", sync: "Activa", color: "bg-success" },
  { name: "Norte Diagnostic Lab", lat: "1.2ms", sync: "Ocupada", color: "bg-warning" },
  { name: "Pacific Pediatric Clinic", lat: "--", sync: "OFFLINE", color: "bg-danger" },
  { name: "MedLink Pharmacy Dist.", lat: "0.12ms", sync: "Activa", color: "bg-success" },
]

const BAR_HEIGHTS = [252, 315, 189, 378, 273, 336, 231, 168, 294, 357]

const SEVERITY_STYLES: Record<Gravedad, "danger" | "warning" | "neutral"> = {
  CRITICAL: "danger",
  HIGH: "warning",
  WARNING: "warning",
  INFORMATIVE: "neutral",
}

export default function AdminDashboardPage() {
  const [auditoria, setAuditoria] = useState<AuditoriaCriticalResponse | null>(null)
  const [entidades, setEntidades] = useState<AdminEntidadListResponse | null>(null)
  const [usuarios, setUsuarios] = useState<UsuariosTotalesResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([
      fetchAuditoriaCritical(),
      fetchEntidadesAdmin(),
      fetchUsuariosTotales(),
    ])
      .then(([a, e, u]) => {
        setAuditoria(a)
        setEntidades(e)
        setUsuarios(u)
      })
      .catch((err) => setError(err.message))
  }, [])

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 rounded border border-line p-12 text-center">
        <ShieldAlert className="size-8 text-danger" />
        <p className="text-sm text-slate">
          No se pudo conectar con el backend: {error}
        </p>
      </div>
    )
  }

  if (!auditoria || !entidades || !usuarios) {
    return <p className="p-6 text-sm text-slate">Cargando dashboard...</p>
  }

  const entidadesActivas = entidades.entidades.filter((e) => e.estado).length
  const entidadesInactivas = entidades.entidades.length - entidadesActivas

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div>
        <h2 className="font-display text-3xl font-semibold text-ink">
          Master Dashboard
        </h2>
        <p className="mt-1 text-sm text-label">
          <span className="text-muted">Platform Admin</span>
          <span className="mx-2 text-muted">›</span>
          System Overview
        </p>
      </div>

      {/* Tarjetas resumen — datos reales */}
      <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
        <StatCard
          label="Usuarios Activos Totales"
          value={usuarios.total.toLocaleString("es-CO")}
          icon={<Users className="size-6" />}
        />
        <StatCard
          label="Entidades de Salud"
          value={String(entidades.total)}
          hint={`${entidadesActivas} activas · ${entidadesInactivas} inactivas`}
          icon={<Building2 className="size-6" />}
        />
        <StatCard
          label="Uptime del Sistema"
          value="99.98%"
          hint="Todos los clusters respondiendo"
          icon={<Activity className="size-6" />}
        />
        <StatCard
          label="Alertas de Seguridad"
          value={String(auditoria.total).padStart(2, "0")}
          hint="Requiere auditoría inmediata"
          valueClassName="text-danger"
          icon={<ShieldAlert className="size-6 text-danger" />}
        />
      </div>

      {/* Bento grid: métricas + nodos (mock, sin endpoint aún) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="col-span-2 p-6">
          <div className="mb-6 flex items-center justify-between">
            <div>
              <h3 className="font-display text-lg font-semibold text-ink">
                Métricas de Uso Global
              </h3>
              <p className="text-sm text-slate">
                Volumen de interacciones — últimos 30 días
              </p>
            </div>
            <span className="rounded border border-line bg-field px-3 py-1.5 text-xs text-slate">
              Últimos 30 días
            </span>
          </div>

          <div className="relative h-64">
            <div className="absolute inset-0 flex flex-col justify-between">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="border-t border-line/40" />
              ))}
            </div>
            <div className="absolute inset-x-0 bottom-0 flex items-end gap-1 px-4">
              {BAR_HEIGHTS.map((h, i) => (
                <div
                  key={i}
                  className={`flex-1 rounded-t transition-all ${
                    i === 3 ? "bg-navy" : "bg-navy/20 hover:bg-navy/40"
                  }`}
                  style={{ height: `${(h / 378) * 100}%` }}
                />
              ))}
            </div>
            <div className="absolute bottom-[100%] left-[37%] -translate-x-1/2 translate-y-2">
              <span className="rounded bg-navy px-2 py-0.5 text-[10px] text-white">
                Pico
              </span>
            </div>
          </div>
          <div className="mt-2 flex justify-between px-4 text-[10px] uppercase tracking-[0.6px] text-muted">
            <span>Oct 01</span>
            <span>Oct 10</span>
            <span>Oct 20</span>
            <span>Oct 30</span>
          </div>
        </Card>

        <Card className="p-6">
          <h3 className="mb-5 font-display text-lg font-semibold text-ink">
            Estado de Nodos
          </h3>
          <div className="flex flex-col gap-3">
            {NODE_STATUS.map((node) => (
              <div
                key={node.name}
                className="flex items-center gap-3 rounded border border-line p-3"
              >
                <div className="flex size-8 items-center justify-center rounded bg-navy">
                  <Building2 className="size-4 text-[#76aecc]" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="truncate text-sm font-medium text-navy-800">
                    {node.name}
                  </p>
                  <p className="text-xs text-muted">
                    Lat: {node.lat} · Sync: {node.sync}
                  </p>
                </div>
                <span
                  className={`size-2.5 shrink-0 rounded-full ${node.color} ring-4 ring-current/10`}
                />
              </div>
            ))}
          </div>
          <button className="mt-4 w-full rounded border border-teal py-2 text-sm text-teal transition-colors hover:bg-teal/5">
            Ver Todos los Nodos
          </button>
        </Card>
      </div>

      {/* Tabla de alertas críticas — datos reales */}
      <Card className="overflow-hidden">
        <div className="flex items-center justify-between border-b border-line p-6">
          <div className="flex items-center gap-2">
            <ShieldAlert className="size-5 text-danger" />
            <h3 className="font-display text-xl font-semibold text-ink">
              Alertas Críticas del Sistema
            </h3>
          </div>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">Timestamp</th>
              <th className="px-6 py-4 font-normal">Severidad</th>
              <th className="px-6 py-4 font-normal">Tipo de Evento</th>
              <th className="px-6 py-4 font-normal">IP de Origen</th>
              <th className="px-6 py-4 font-normal text-right">Usuario</th>
            </tr>
          </thead>
          <tbody>
            {auditoria.registros.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-sm text-muted">
                  No hay alertas críticas registradas.
                </td>
              </tr>
            ) : (
              auditoria.registros.map((entry) => (
                <tr key={entry.id} className="border-t border-line">
                  <td className="px-6 py-4 font-mono text-xs text-slate">
                    {new Date(entry.fecha_operacion).toLocaleString("es-CO")}
                  </td>
                  <td className="px-6 py-4">
                    <Badge tone={SEVERITY_STYLES[entry.gravedad]}>
                      {entry.gravedad}
                    </Badge>
                  </td>
                  <td className="px-6 py-4 text-navy-800">
                    {entry.tipo_operacion} — {entry.tabla_afectada}
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-slate">
                    {entry.ip_origen ?? "—"}
                  </td>
                  <td className="px-6 py-4 text-right text-navy-800">
                    {entry.usuario_nombre}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </Card>
    </div>
  )
}