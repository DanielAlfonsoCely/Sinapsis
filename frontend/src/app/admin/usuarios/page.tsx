"use client"

import { useState, useEffect, useRef, useCallback } from "react"
import {
  Users,
  Building2,
  ShieldCheck,
  UserX,
  Filter,
  Download,
  Pencil,
  Trash2,
  Eye,
  ChevronLeft,
  ChevronRight,
} from "lucide-react"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { StatCard } from "@/components/ui/stat-card"

interface AdminUsuarioItem {
  id: string
  nombre_usuario: string
  apellidos: string
  email: string
  tipo_usuario: string
  estado: boolean
  entidad_nombre: string | null
  fecha_actualizacion: string
}

interface ListUsuariosResponse {
  usuarios: AdminUsuarioItem[]
  total: number
  total_activos: number
  total_inactivos: number
  limit: number
  offset: number
}

const LIMIT = 20

export default function UsuariosPage() {
  const [usuarios, setUsuarios] = useState<AdminUsuarioItem[]>([])
  const [total, setTotal] = useState(0)
  const [totalActivos, setTotalActivos] = useState(0)
  const [totalInactivos, setTotalInactivos] = useState(0)
  const [totalEntidades, setTotalEntidades] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [rolFilter, setRolFilter] = useState("")
  const [currentPage, setCurrentPage] = useState(1)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  async function fetchUsuarios(q: string, rol: string, page: number) {
    const token = document.cookie.split("; ").find(c => c.startsWith("token="))?.split("=")[1]
    const params = new URLSearchParams({
      limit: String(LIMIT),
      offset: String((page - 1) * LIMIT),
      ...(q   && { q }),
      ...(rol  && { rol }),
    })
    const res = await fetch(`http://localhost:8080/api/v1/admin/usuarios?${params}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.status === 403) { window.location.href = "/login"; throw new Error("forbidden") }
    if (!res.ok) throw new Error(`Error ${res.status}`)
    return res.json() as Promise<ListUsuariosResponse>
  }

  const loadData = useCallback(async (q: string, rol: string, page: number) => {
    setLoading(true)
    setError(null)
    try {
      const token = document.cookie.split("; ").find(c => c.startsWith("token="))?.split("=")[1]
      const [data, entidadesRes] = await Promise.all([
        fetchUsuarios(q, rol, page),
        fetch("http://localhost:8080/api/v1/entidades", {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ])
      setUsuarios(data.usuarios)
      setTotal(data.total)
      setTotalActivos(data.total_activos)
      setTotalInactivos(data.total_inactivos)
      if (entidadesRes.ok) {
        const entData = await entidadesRes.json() as { total: number }
        setTotalEntidades(entData.total)
      }
    } catch (err) {
      if (err instanceof Error && err.message !== "forbidden") {
        setError("No se pudo cargar la lista de usuarios.")
      }
    } finally {
      setLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    void loadData(searchQuery, rolFilter, currentPage)
  }, [currentPage, rolFilter, loadData])

  async function handleRolChange(userId: string, nuevoRol: string, index: number) {
    const rolAnterior = usuarios[index].tipo_usuario
    // 1. Optimistic update
    setUsuarios(prev => prev.map((u, i) => i === index ? { ...u, tipo_usuario: nuevoRol } : u))
    try {
      const token = document.cookie.split("; ").find(c => c.startsWith("token="))?.split("=")[1]
      const res = await fetch(`http://localhost:8080/api/v1/admin/usuarios/${userId}/rol`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ tipo_usuario: nuevoRol }),
      })
      if (!res.ok) throw new Error(`Error ${res.status}`)
      // 2. Sincronizar fecha_actualizacion desde el servidor
      const data = await res.json() as { fecha_actualizacion: string }
      setUsuarios(prev => prev.map((u, i) => i === index ? { ...u, fecha_actualizacion: data.fecha_actualizacion } : u))
    } catch {
      // 3. Rollback
      setUsuarios(prev => prev.map((u, i) => i === index ? { ...u, tipo_usuario: rolAnterior } : u))
      setError("No se pudo cambiar el rol. Intenta de nuevo.")
    }
  }

  function handleSearchChange(value: string) {
    setSearchQuery(value)
    setCurrentPage(1)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => void loadData(value, rolFilter, 1), 400)
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-start justify-between">
        <div>
          <h2 className="font-display text-3xl font-semibold text-ink">
            Gestión Global de Usuarios
          </h2>
          <p className="mt-1 text-sm text-slate">
            Administra todos los usuarios registrados en la plataforma SINAPSIS
          </p>
        </div>
        <button className="flex items-center gap-2 rounded bg-navy px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-navy-800">
          <Users className="size-4" />
          Crear Usuario
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
        <StatCard
          label="Total Usuarios"
          value={total.toLocaleString("es-CO")}
          icon={<Users className="size-6" />}
        />
        <StatCard
          label="Entidades Registradas"
          value={totalEntidades !== null ? totalEntidades.toLocaleString("es-CO") : "–"}
          icon={<Building2 className="size-6" />}
        />
        <StatCard
          label="Usuarios Activos"
          value={totalActivos.toLocaleString("es-CO")}
          icon={<ShieldCheck className="size-6" />}
        />
        <StatCard
          label="Cuentas Inactivas"
          value={totalInactivos.toLocaleString("es-CO")}
          valueClassName="text-danger"
          icon={<UserX className="size-6 text-danger" />}
        />
      </div>

      {/* Filtros */}
      <Card className="p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2 text-sm text-slate">
            <Filter className="size-4" />
            <span className="font-medium">Filtrar por:</span>
          </div>
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => handleSearchChange(e.target.value)}
            placeholder="Buscar usuario..."
            className="rounded border border-line bg-field px-3 py-2 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
          />
          <select
            value={rolFilter}
            onChange={(e) => { setRolFilter(e.target.value); setCurrentPage(1) }}
            className="rounded border border-line bg-field px-3 py-2 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
          >
            <option value="">Todos los roles</option>
            <option value="medico">Médico</option>
            <option value="paciente">Paciente</option>
            <option value="admin_entidad">Admin Entidad</option>
            <option value="admin_plataforma">Admin Plataforma</option>
          </select>
          <div className="ml-auto flex gap-2">
            <button className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <Download className="size-4" />
            </button>
          </div>
        </div>
      </Card>

      {/* Banner de error */}
      {error && (
        <div className="rounded border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
          {error}
        </div>
      )}

      {/* Tabla */}
      <Card className="overflow-hidden">
        {loading && (
          <div className="px-6 py-4 text-center text-sm text-muted">Cargando...</div>
        )}
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">Usuario</th>
              <th className="px-6 py-4 font-normal">Correo</th>
              <th className="px-6 py-4 font-normal">Rol</th>
              <th className="px-6 py-4 font-normal">Entidad</th>
              <th className="px-6 py-4 font-normal">Estado</th>
              <th className="px-6 py-4 font-normal text-center">Acciones</th>
            </tr>
          </thead>
          <tbody>
            {usuarios.map((u, index) => (
              <tr key={u.id} className="border-t border-line hover:bg-field/30">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <span className="flex size-8 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-xs font-medium text-teal">
                      {u.nombre_usuario.charAt(0).toUpperCase()}
                      {u.apellidos.charAt(0).toUpperCase()}
                    </span>
                    <div>
                      <p className="font-medium text-navy-800">
                        {u.nombre_usuario} {u.apellidos}
                      </p>
                      <p className="text-xs text-muted">{u.email}</p>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-slate">{u.email}</td>
                <td className="px-6 py-4">
                  <select
                    value={u.tipo_usuario}
                    onChange={(e) => void handleRolChange(u.id, e.target.value, index)}
                    className="rounded border border-line bg-field px-2 py-1 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                  >
                    <option value="medico">Médico</option>
                    <option value="paciente">Paciente</option>
                    <option value="admin_entidad">Admin Entidad</option>
                    <option value="admin_plataforma">Admin Plataforma</option>
                  </select>
                </td>
                <td className="px-6 py-4 text-slate">{u.entidad_nombre ?? "—"}</td>
                <td className="px-6 py-4">
                  <Badge tone={u.estado ? "success" : "neutral"}>
                    {u.estado ? "Activo" : "Inactivo"}
                  </Badge>
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-center justify-center gap-2">
                    <button className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-field hover:text-ink">
                      <Eye className="size-4" />
                    </button>
                    <button className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-field hover:text-teal">
                      <Pencil className="size-4" />
                    </button>
                    <button className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-danger/10 hover:text-danger">
                      <Trash2 className="size-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {!loading && usuarios.length === 0 && (
              <tr>
                <td
                  colSpan={6}
                  className="px-6 py-10 text-center text-sm text-muted"
                >
                  No hay usuarios para mostrar.
                </td>
              </tr>
            )}
          </tbody>
        </table>

        {/* Paginación */}
        {(() => {
          const totalPages = Math.ceil(total / LIMIT)
          const start = total === 0 ? 0 : (currentPage - 1) * LIMIT + 1
          const end = Math.min(currentPage * LIMIT, total)

          // Ventana de hasta 5 páginas centrada en currentPage
          const windowSize = 5
          const halfWindow = Math.floor(windowSize / 2)
          let pageStart = Math.max(1, currentPage - halfWindow)
          const pageEnd = Math.min(totalPages, pageStart + windowSize - 1)
          pageStart = Math.max(1, pageEnd - windowSize + 1)
          const pageNumbers: number[] = []
          for (let i = pageStart; i <= pageEnd; i++) pageNumbers.push(i)

          return (
            <div className="flex items-center justify-between border-t border-line px-6 py-4">
              <p className="text-sm text-muted">
                {total === 0
                  ? "No hay usuarios para mostrar"
                  : `Mostrando ${start.toLocaleString("es-CO")}–${end.toLocaleString("es-CO")} de ${total.toLocaleString("es-CO")} usuarios`}
              </p>
              <div className="flex items-center gap-1">
                <button
                  disabled={currentPage === 1}
                  onClick={() => setCurrentPage((p) => p - 1)}
                  className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <ChevronLeft className="size-4" />
                </button>
                {pageNumbers.map((p) => (
                  <button
                    key={p}
                    onClick={() => setCurrentPage(p)}
                    className={`flex size-8 items-center justify-center rounded border text-sm transition-colors ${
                      p === currentPage
                        ? "border-teal bg-teal text-white"
                        : "border-line text-slate hover:bg-field"
                    }`}
                  >
                    {p}
                  </button>
                ))}
                <button
                  disabled={currentPage >= totalPages}
                  onClick={() => setCurrentPage((p) => p + 1)}
                  className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <ChevronRight className="size-4" />
                </button>
              </div>
            </div>
          )
        })()}
      </Card>

      {/* Footer de seguridad */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card className="p-6">
          <div className="flex items-start gap-4">
            <ShieldCheck className="mt-0.5 size-5 shrink-0 text-teal" />
            <div>
              <h4 className="font-display text-base font-semibold text-ink">
                Política de Contraseñas
              </h4>
              <p className="mt-1 text-sm text-slate">
                Todas las cuentas requieren contraseñas de mínimo 12 caracteres con
                autenticación de dos factores habilitada.
              </p>
              <button className="mt-3 text-sm text-teal hover:text-teal-700">
                Revisar política de acceso →
              </button>
            </div>
          </div>
        </Card>
        <Card className="p-6">
          <div className="flex items-start gap-4">
            <Users className="mt-0.5 size-5 shrink-0 text-teal" />
            <div>
              <h4 className="font-display text-base font-semibold text-ink">
                Control de Roles
              </h4>
              <p className="mt-1 text-sm text-slate">
                La asignación de roles se rige por el principio de mínimo
                privilegio. Revisa regularmente los permisos de administrador.
              </p>
              <button className="mt-3 text-sm text-teal hover:text-teal-700">
                Gestionar roles →
              </button>
            </div>
          </div>
        </Card>
      </div>
    </div>
  )
}
