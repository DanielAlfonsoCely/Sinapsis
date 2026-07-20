"use client"

import { useEffect, useState } from "react"
import { useParams } from "next/navigation"
import Link from "next/link"
import { Building2, ChevronLeft, Pencil, X } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

interface PeriodoHistorico {
  anio: number
  cantidad_historias: number
}

interface UsuarioAsociado {
  id: string
  nombre_usuario: string
  apellidos: string
  tipo_usuario: string
}

interface EntidadDetalle {
  id: string
  nombre_entidad: string
  tipo_entidad: string
  nit: string
  ciudad: string | null
  direccion: string | null
  telefono: string | null
  estado: boolean
  fecha_creacion: string
  convenios_activos: number
  periodos_historicos: PeriodoHistorico[]
  usuarios_asociados: UsuarioAsociado[]
}

function BarChart({ data }: { data: { anio: number; cantidad_historias: number }[] }) {
  if (data.length === 0) return <p className="text-sm text-muted">Sin datos para graficar.</p>

  const maxVal = Math.max(...data.map((d) => d.cantidad_historias), 1)
  const chartH = 120
  const barW = 32
  const gap = 16
  const totalW = data.length * (barW + gap) - gap

  return (
    <svg width={totalW} height={chartH + 24} className="overflow-visible">
      {data.map((d, i) => {
        const barH = Math.max((d.cantidad_historias / maxVal) * chartH, 2)
        const x = i * (barW + gap)
        const y = chartH - barH
        return (
          <g key={d.anio}>
            <rect x={x} y={y} width={barW} height={barH} rx={4} className="fill-teal" />
            <text
              x={x + barW / 2}
              y={chartH + 14}
              textAnchor="middle"
              className="fill-muted text-[10px]"
              fontSize={10}
            >
              {d.anio}
            </text>
            <text
              x={x + barW / 2}
              y={y - 4}
              textAnchor="middle"
              className="fill-slate"
              fontSize={10}
            >
              {d.cantidad_historias}
            </text>
          </g>
        )
      })}
    </svg>
  )
}

const ROL_COLORS: Record<string, string> = {
  medico:           "#0d9488", // teal
  admin_entidad:    "#1e3a5f", // navy
  paciente:         "#64748b", // slate
  admin_plataforma: "#f59e0b", // warning/amber
}
const ROL_LABELS: Record<string, string> = {
  medico:           "Médico",
  admin_entidad:    "Admin Entidad",
  paciente:         "Paciente",
  admin_plataforma: "Admin Plataforma",
}

function DonutChart({ usuarios }: { usuarios: { tipo_usuario: string }[] }) {
  if (usuarios.length === 0) return <p className="text-sm text-muted">Sin usuarios para graficar.</p>

  // Contar por rol
  const counts: Record<string, number> = {}
  for (const u of usuarios) {
    counts[u.tipo_usuario] = (counts[u.tipo_usuario] ?? 0) + 1
  }
  const entries = Object.entries(counts)
  const total = usuarios.length

  // Calcular arcos SVG
  const cx = 60
  const cy = 60
  const R = 50
  const r = 28

  let startAngle = -Math.PI / 2
  const slices = entries.map(([rol, count]) => {
    const angle = (count / total) * 2 * Math.PI
    const endAngle = startAngle + angle
    const x1 = cx + R * Math.cos(startAngle)
    const y1 = cy + R * Math.sin(startAngle)
    const x2 = cx + R * Math.cos(endAngle)
    const y2 = cy + R * Math.sin(endAngle)
    const ix1 = cx + r * Math.cos(endAngle)
    const iy1 = cy + r * Math.sin(endAngle)
    const ix2 = cx + r * Math.cos(startAngle)
    const iy2 = cy + r * Math.sin(startAngle)
    const largeArc = angle > Math.PI ? 1 : 0
    const d = `M ${x1} ${y1} A ${R} ${R} 0 ${largeArc} 1 ${x2} ${y2} L ${ix1} ${iy1} A ${r} ${r} 0 ${largeArc} 0 ${ix2} ${iy2} Z`
    const slice = { rol, count, d, color: ROL_COLORS[rol] ?? "#94a3b8" }
    startAngle = endAngle
    return slice
  })

  return (
    <div className="flex items-center gap-6">
      <svg width={120} height={120}>
        {slices.map((s) => (
          <path key={s.rol} d={s.d} fill={s.color} />
        ))}
        <text x={cx} y={cy + 4} textAnchor="middle" fontSize={13} fontWeight="bold" fill="#1e3a5f">
          {total}
        </text>
      </svg>
      <ul className="flex flex-col gap-1.5">
        {slices.map((s) => (
          <li key={s.rol} className="flex items-center gap-2 text-xs text-slate">
            <span className="inline-block size-2.5 rounded-full" style={{ backgroundColor: s.color }} />
            {ROL_LABELS[s.rol] ?? s.rol}: <span className="font-medium text-ink">{s.count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

export default function EntidadDetallePage() {
  const params = useParams()
  const id = params?.id as string

  const [detalle, setDetalle] = useState<EntidadDetalle | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // --- Estado modal de edición ---
  const [showEditar, setShowEditar] = useState(false)
  const [editForm, setEditForm] = useState({
    nombre_entidad: "",
    tipo_entidad: "IPS",
    nit: "",
    ciudad: "",
    direccion: "",
    telefono: "",
    estado: true,
  })
  const [editError, setEditError] = useState<string | null>(null)
  const [editLoading, setEditLoading] = useState(false)

  function openEditar() {
    if (!detalle) return
    setEditForm({
      nombre_entidad: detalle.nombre_entidad,
      tipo_entidad: detalle.tipo_entidad,
      nit: detalle.nit,
      ciudad: detalle.ciudad ?? "",
      direccion: detalle.direccion ?? "",
      telefono: detalle.telefono ?? "",
      estado: detalle.estado,
    })
    setEditError(null)
    setShowEditar(true)
  }

  async function handleEditarSubmit(e: React.FormEvent) {
    e.preventDefault()
    setEditError(null)
    setEditLoading(true)
    try {
      const token = document.cookie
        .split("; ")
        .find((c) => c.startsWith("token="))
        ?.split("=")[1]
      const res = await fetch(`http://localhost:8080/api/v1/admin/entidades/${id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token ?? ""}`,
        },
        body: JSON.stringify({
          nombre_entidad: editForm.nombre_entidad,
          tipo_entidad: editForm.tipo_entidad,
          nit: editForm.nit,
          ciudad: editForm.ciudad || null,
          direccion: editForm.direccion || null,
          telefono: editForm.telefono || null,
          estado: editForm.estado,
        }),
      })
      const data = await res.json().catch(() => ({})) as { error?: string }
      if (!res.ok) {
        setEditError(data.error ?? "No se pudo actualizar la entidad.")
        return
      }
      // Actualizar detalle local con los nuevos valores
      setDetalle((prev) =>
        prev
          ? {
              ...prev,
              nombre_entidad: editForm.nombre_entidad,
              tipo_entidad: editForm.tipo_entidad,
              nit: editForm.nit,
              ciudad: editForm.ciudad || null,
              direccion: editForm.direccion || null,
              telefono: editForm.telefono || null,
              estado: editForm.estado,
            }
          : prev
      )
      setShowEditar(false)
    } catch {
      setEditError("Error de conexión con el servidor.")
    } finally {
      setEditLoading(false)
    }
  }

  useEffect(() => {
    async function fetchDetalle() {
      try {
        const token = document.cookie
          .split("; ")
          .find((c) => c.startsWith("token="))
          ?.split("=")[1]
        const res = await fetch(
          `http://localhost:8080/api/v1/admin/entidades/${id}`,
          { headers: { Authorization: `Bearer ${token ?? ""}` } }
        )
        if (res.status === 403) {
          window.location.href = "/login"
          return
        }
        if (res.status === 404) {
          setError("Entidad no encontrada.")
          return
        }
        if (!res.ok) throw new Error(`Error ${res.status}`)
        const data: EntidadDetalle = await res.json()
        setDetalle(data)
      } catch {
        setError("No se pudo cargar el detalle de la entidad.")
      } finally {
        setLoading(false)
      }
    }
    if (id) void fetchDetalle()
  }, [id])

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <div className="py-12 text-center text-sm text-muted">Cargando…</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <div className="rounded border border-danger bg-danger/10 p-4 text-sm text-danger">
          {error}
        </div>
        <Link
          href="/admin/entidades"
          className="flex w-fit items-center gap-1 text-sm text-teal hover:underline"
        >
          <ChevronLeft className="size-4" />
          Volver al listado
        </Link>
      </div>
    )
  }

  if (!detalle) return null

  return (
    <div className="flex flex-col gap-6">
      {/* Modal de edición */}
      {showEditar && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-lg flex-col gap-5 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">
                Editar entidad de salud
              </h3>
              <button
                type="button"
                onClick={() => setShowEditar(false)}
                className="text-muted hover:text-ink"
              >
                <X className="size-4" />
              </button>
            </div>

            <form onSubmit={handleEditarSubmit} className="flex flex-col gap-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="sm:col-span-2">
                  <label className="mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Nombre de la entidad
                  </label>
                  <input
                    required
                    maxLength={150}
                    value={editForm.nombre_entidad}
                    onChange={(e) => setEditForm({ ...editForm, nombre_entidad: e.target.value })}
                    className="h-10 w-full rounded border border-line bg-field px-3 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                  />
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Tipo de entidad
                  </label>
                  <select
                    value={editForm.tipo_entidad}
                    onChange={(e) => setEditForm({ ...editForm, tipo_entidad: e.target.value })}
                    className="h-10 w-full rounded border border-line bg-field px-3 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                  >
                    <option value="IPS">IPS</option>
                    <option value="EPS">EPS</option>
                    <option value="clinica">Clínica</option>
                    <option value="hospital">Hospital</option>
                    <option value="consultorio">Consultorio</option>
                  </select>
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    NIT
                  </label>
                  <input
                    required
                    maxLength={50}
                    value={editForm.nit}
                    onChange={(e) => setEditForm({ ...editForm, nit: e.target.value })}
                    className="h-10 w-full rounded border border-line bg-field px-3 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                  />
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Ciudad
                  </label>
                  <input
                    maxLength={100}
                    value={editForm.ciudad}
                    onChange={(e) => setEditForm({ ...editForm, ciudad: e.target.value })}
                    className="h-10 w-full rounded border border-line bg-field px-3 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                    placeholder="Bogotá"
                  />
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Teléfono
                  </label>
                  <input
                    maxLength={50}
                    value={editForm.telefono}
                    onChange={(e) => setEditForm({ ...editForm, telefono: e.target.value })}
                    className="h-10 w-full rounded border border-line bg-field px-3 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                    placeholder="6012000000"
                  />
                </div>

                <div className="sm:col-span-2">
                  <label className="mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Dirección
                  </label>
                  <input
                    maxLength={255}
                    value={editForm.direccion}
                    onChange={(e) => setEditForm({ ...editForm, direccion: e.target.value })}
                    className="h-10 w-full rounded border border-line bg-field px-3 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                    placeholder="Calle 119 # 7-75"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <input
                    id="estado-check"
                    type="checkbox"
                    checked={editForm.estado}
                    onChange={(e) => setEditForm({ ...editForm, estado: e.target.checked })}
                    className="size-4 accent-teal"
                  />
                  <label htmlFor="estado-check" className="text-sm text-slate">
                    Entidad activa
                  </label>
                </div>
              </div>

              {editError && <p className="text-sm text-danger">{editError}</p>}

              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowEditar(false)}
                  className="rounded border border-line px-4 py-2 text-sm text-slate transition-colors hover:bg-field"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={editLoading}
                  className="rounded bg-navy px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-navy-800 disabled:opacity-60"
                >
                  {editLoading ? "Guardando…" : "Guardar cambios"}
                </button>
              </div>
            </form>
          </Card>
        </div>
      )}

      {/* Header */}
      <div className="flex items-start justify-between border-b border-line pb-5">
        <div className="flex items-center gap-3">
          <Building2 className="size-5 text-teal" />
          <div>
            <div className="flex items-center gap-3">
              <h2 className="font-display text-2xl font-semibold text-ink">
                {detalle.nombre_entidad}
              </h2>
              <Badge tone={detalle.estado ? "success" : "danger"}>
                {detalle.estado ? "Activa" : "Inactiva"}
              </Badge>
              <Badge tone="neutral">{detalle.tipo_entidad}</Badge>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={openEditar}
            className="flex items-center gap-2 rounded border border-navy px-4 py-2 text-sm text-navy transition-colors hover:bg-navy hover:text-white"
          >
            <Pencil className="size-4" />
            Editar entidad
          </button>
          <Link
            href="/admin/entidades"
            className="flex items-center gap-1 text-sm text-teal hover:underline"
          >
            <ChevronLeft className="size-4" />
            Volver al listado
          </Link>
        </div>
      </div>

      {/* Datos generales + Convenios activos */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="col-span-2 p-6">
          <h3 className="mb-4 font-display text-base font-semibold text-ink">
            Datos generales
          </h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-[10px] uppercase tracking-[0.6px] text-muted">NIT</p>
              <p className="mt-0.5 text-sm text-navy-800">{detalle.nit}</p>
            </div>
            <div>
              <p className="text-[10px] uppercase tracking-[0.6px] text-muted">CIUDAD</p>
              <p className="mt-0.5 text-sm text-navy-800">{detalle.ciudad ?? "—"}</p>
            </div>
            <div>
              <p className="text-[10px] uppercase tracking-[0.6px] text-muted">DIRECCIÓN</p>
              <p className="mt-0.5 text-sm text-navy-800">{detalle.direccion ?? "—"}</p>
            </div>
            <div>
              <p className="text-[10px] uppercase tracking-[0.6px] text-muted">TELÉFONO</p>
              <p className="mt-0.5 text-sm text-navy-800">{detalle.telefono ?? "—"}</p>
            </div>
            <div>
              <p className="text-[10px] uppercase tracking-[0.6px] text-muted">FECHA DE REGISTRO</p>
              <p className="mt-0.5 text-sm text-navy-800">
                {new Date(detalle.fecha_creacion).toLocaleDateString("es-CO")}
              </p>
            </div>
          </div>
        </Card>

        <Card className="flex flex-col items-center justify-center p-6 text-center">
          <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
            CONVENIOS ACTIVOS
          </p>
          <p className="mt-2 font-display text-4xl font-bold text-teal">
            {detalle.convenios_activos}
          </p>
          <p className="mt-1 text-xs text-slate">historias clínicas activas</p>
        </Card>
      </div>

      {/* Períodos históricos */}
      <Card className="p-6">
        <h3 className="mb-4 font-display text-base font-semibold text-ink">
          Períodos históricos
        </h3>
        {detalle.periodos_historicos.length === 0 ? (
          <p className="text-sm text-muted">Sin períodos registrados.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-line text-left">
                <th className="pb-2 text-[10px] uppercase tracking-[0.6px] text-muted">Año</th>
                <th className="pb-2 text-[10px] uppercase tracking-[0.6px] text-muted">
                  Historias clínicas
                </th>
              </tr>
            </thead>
            <tbody>
              {detalle.periodos_historicos.map((p) => (
                <tr key={p.anio} className="border-b border-line last:border-0">
                  <td className="py-2 text-navy-800">{p.anio}</td>
                  <td className="py-2 text-navy-800">{p.cantidad_historias}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      {/* Usuarios asociados */}
      <Card className="p-6">
        <h3 className="mb-4 font-display text-base font-semibold text-ink">
          Usuarios asociados
        </h3>
        {detalle.usuarios_asociados.length === 0 ? (
          <p className="text-sm text-muted">Sin usuarios asociados.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-line text-left">
                <th className="pb-2 text-[10px] uppercase tracking-[0.6px] text-muted">Nombre</th>
                <th className="pb-2 text-[10px] uppercase tracking-[0.6px] text-muted">Apellidos</th>
                <th className="pb-2 text-[10px] uppercase tracking-[0.6px] text-muted">Rol</th>
              </tr>
            </thead>
            <tbody>
              {detalle.usuarios_asociados.map((u) => (
                <tr key={u.id} className="border-b border-line last:border-0">
                  <td className="py-2 text-navy-800">{u.nombre_usuario}</td>
                  <td className="py-2 text-navy-800">{u.apellidos}</td>
                  <td className="py-2 text-navy-800">{u.tipo_usuario}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      {/* Gráficas */}
      {(detalle.periodos_historicos.length > 0 || detalle.usuarios_asociados.length > 0) && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card className="p-6">
            <h3 className="mb-4 font-display text-base font-semibold text-ink">
              Actividad por año
            </h3>
            <div className="overflow-x-auto">
              <BarChart data={[...detalle.periodos_historicos].reverse()} />
            </div>
          </Card>

          <Card className="p-6">
            <h3 className="mb-4 font-display text-base font-semibold text-ink">
              Distribución de roles
            </h3>
            <DonutChart usuarios={detalle.usuarios_asociados} />
          </Card>
        </div>
      )}
    </div>
  )
}
