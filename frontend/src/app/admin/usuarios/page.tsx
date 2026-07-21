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
  X,
} from "lucide-react"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { StatCard } from "@/components/ui/stat-card"
import { exportToCSV } from "./api"

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

interface EntidadItem {
  id: string
  nombre_entidad: string
}

const LIMIT = 20

function getToken() {
  return document.cookie.split("; ").find(c => c.startsWith("token="))?.split("=")[1]
}

// ---------------------------------------------------------------------------
// Formulario de creación de usuario — campos base + condicionales por rol
// ---------------------------------------------------------------------------
interface CrearUsuarioForm {
  nombre_usuario: string
  apellidos: string
  email: string
  contrasena: string
  tipo_usuario: string
  numero_documento: string
  especialidad: string
  numero_colegiado: string
  experiencia_anios: string
  entidad_id: string
  tipo_documento: string
  fecha_nacimiento: string
  sexo: string
  telefono: string
}

const CREAR_USUARIO_INICIAL: CrearUsuarioForm = {
  nombre_usuario: "",
  apellidos: "",
  email: "",
  contrasena: "",
  tipo_usuario: "paciente",
  numero_documento: "",
  especialidad: "",
  numero_colegiado: "",
  experiencia_anios: "",
  entidad_id: "",
  tipo_documento: "CC",
  fecha_nacimiento: "",
  sexo: "",
  telefono: "",
}

interface EditarUsuarioForm {
  nombre_usuario: string
  apellidos: string
  email: string
  estado: boolean
}

export default function UsuariosPage() {
  const [usuarios, setUsuarios] = useState<AdminUsuarioItem[]>([])
  const [total, setTotal] = useState(0)
  const [totalActivos, setTotalActivos] = useState(0)
  const [totalInactivos, setTotalInactivos] = useState(0)
  const [totalEntidades, setTotalEntidades] = useState<number | null>(null)
  const [entidades, setEntidades] = useState<EntidadItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [rolFilter, setRolFilter] = useState("")
  const [currentPage, setCurrentPage] = useState(1)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [exportLoading, setExportLoading] = useState(false)

  // --- Estado de modales ---
  const [showCrear, setShowCrear] = useState(false)
  const [crearForm, setCrearForm] = useState<CrearUsuarioForm>(CREAR_USUARIO_INICIAL)
  const [crearError, setCrearError] = useState<string | null>(null)
  const [crearLoading, setCrearLoading] = useState(false)

  const [showVer, setShowVer] = useState(false)
  const [usuarioVer, setUsuarioVer] = useState<AdminUsuarioItem | null>(null)

  const [showEditar, setShowEditar] = useState(false)
  const [usuarioEditar, setUsuarioEditar] = useState<AdminUsuarioItem | null>(null)
  const [editarForm, setEditarForm] = useState<EditarUsuarioForm>({
    nombre_usuario: "",
    apellidos: "",
    email: "",
    estado: true,
  })
  const [editarError, setEditarError] = useState<string | null>(null)
  const [editarLoading, setEditarLoading] = useState(false)

  const [showBorrar, setShowBorrar] = useState(false)
  const [usuarioBorrar, setUsuarioBorrar] = useState<AdminUsuarioItem | null>(null)
  const [borrarError, setBorrarError] = useState<string | null>(null)
  const [borrarLoading, setBorrarLoading] = useState(false)

  async function fetchUsuarios(q: string, rol: string, page: number) {
    const token = getToken()
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

  // Agregar en ./api.ts (o donde tengas fetchUsuarios) se puede usar para búsquedas variadas

async function fetchUsuariosSinLimite(q: string, rol: string): Promise<AdminUsuarioItem[]> {
  const token = getToken()
  const params = new URLSearchParams({
    limit: "1000000", // valor alto para traer todo lo que matchee el filtro
    offset: "0",
    ...(q && { q }),
    ...(rol && { rol }),
  })
  const res = await fetch(`http://localhost:8080/api/v1/admin/usuarios?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (res.status === 403) { window.location.href = "/login"; throw new Error("forbidden") }
  if (!res.ok) throw new Error(`Error ${res.status}`)
  const data = await res.json() as ListUsuariosResponse
  return data.usuarios
}

  const loadData = useCallback(async (q: string, rol: string, page: number) => {
    setLoading(true)
    setError(null)
    try {
      const token = getToken()
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
        const entData = await entidadesRes.json() as { total: number; entidades?: EntidadItem[] }
        setTotalEntidades(entData.total)
        if (entData.entidades) setEntidades(entData.entidades)
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
    setUsuarios(prev => prev.map((u, i) => i === index ? { ...u, tipo_usuario: nuevoRol } : u))
    try {
      const token = getToken()
      const res = await fetch(`http://localhost:8080/api/v1/admin/usuarios/${userId}/rol`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ tipo_usuario: nuevoRol }),
      })
      if (!res.ok) throw new Error(`Error ${res.status}`)
      const data = await res.json() as { fecha_actualizacion: string }
      setUsuarios(prev => prev.map((u, i) => i === index ? { ...u, fecha_actualizacion: data.fecha_actualizacion } : u))
    } catch {
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

  // -------------------------------------------------------------------
  // Crear Usuario — POST /api/v1/admin/usuarios
  // -------------------------------------------------------------------
  function openCrear() {
    setCrearForm(CREAR_USUARIO_INICIAL)
    setCrearError(null)
    setShowCrear(true)
  }

  async function handleCrearSubmit(e: React.FormEvent) {
    e.preventDefault()
    setCrearError(null)
    setCrearLoading(true)
    try {
      const token = getToken()
      const payload: Record<string, unknown> = {
        nombre_usuario: crearForm.nombre_usuario,
        apellidos: crearForm.apellidos,
        email: crearForm.email,
        contrasena: crearForm.contrasena,
        tipo_usuario: crearForm.tipo_usuario,
      }
      if (crearForm.tipo_usuario === "medico") {
        payload.numero_documento = crearForm.numero_documento
        payload.especialidad = crearForm.especialidad
        payload.numero_colegiado = crearForm.numero_colegiado
        payload.experiencia_anios = crearForm.experiencia_anios ? Number(crearForm.experiencia_anios) : null
        payload.entidad_id = crearForm.entidad_id
      } else if (crearForm.tipo_usuario === "paciente") {
        payload.numero_documento = crearForm.numero_documento
        payload.tipo_documento = crearForm.tipo_documento
        payload.fecha_nacimiento = crearForm.fecha_nacimiento
        payload.sexo = crearForm.sexo
        payload.telefono = crearForm.telefono
      } else if (crearForm.tipo_usuario === "admin_entidad") {
        payload.entidad_id = crearForm.entidad_id
      }

      const res = await fetch("http://localhost:8080/api/v1/admin/usuarios", {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => null) as { error?: string } | null
        throw new Error(body?.error ?? `Error ${res.status}`)
      }
      setShowCrear(false)
      setCurrentPage(1)
      void loadData(searchQuery, rolFilter, 1)
    } catch (err) {
      setCrearError(err instanceof Error ? err.message : "No se pudo crear el usuario.")
    } finally {
      setCrearLoading(false)
    }
  }

  // -------------------------------------------------------------------
  // Ver Usuario (solo lectura, sin llamada al backend)
  // -------------------------------------------------------------------
  function openVer(u: AdminUsuarioItem) {
    setUsuarioVer(u)
    setShowVer(true)
  }

  // -------------------------------------------------------------------
  // Editar Usuario — PUT /api/v1/admin/usuarios/:id
  // -------------------------------------------------------------------
  function openEditar(u: AdminUsuarioItem) {
    setUsuarioEditar(u)
    setEditarForm({
      nombre_usuario: u.nombre_usuario,
      apellidos: u.apellidos,
      email: u.email,
      estado: u.estado,
    })
    setEditarError(null)
    setShowEditar(true)
  }

  async function handleEditarSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!usuarioEditar) return
    setEditarError(null)
    setEditarLoading(true)
    try {
      const token = getToken()
      const res = await fetch(`http://localhost:8080/api/v1/admin/usuarios/${usuarioEditar.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          nombre_usuario: editarForm.nombre_usuario,
          apellidos: editarForm.apellidos,
          email: editarForm.email,
          estado: editarForm.estado,
        }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => null) as { error?: string } | null
        throw new Error(body?.error ?? `Error ${res.status}`)
      }
      setShowEditar(false)
      void loadData(searchQuery, rolFilter, currentPage)
    } catch (err) {
      setEditarError(err instanceof Error ? err.message : "No se pudo editar el usuario.")
    } finally {
      setEditarLoading(false)
    }
  }

  // -------------------------------------------------------------------
  // Eliminar Usuario — DELETE /api/v1/admin/usuarios/:id (soft delete)
  // -------------------------------------------------------------------
  function openBorrar(u: AdminUsuarioItem) {
    setUsuarioBorrar(u)
    setBorrarError(null)
    setShowBorrar(true)
  }

  async function handleBorrarConfirm() {
    if (!usuarioBorrar) return
    setBorrarError(null)
    setBorrarLoading(true)
    try {
      const token = getToken()
      const res = await fetch(`http://localhost:8080/api/v1/admin/usuarios/${usuarioBorrar.id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) {
        const body = await res.json().catch(() => null) as { error?: string } | null
        throw new Error(body?.error ?? `Error ${res.status}`)
      }
      setShowBorrar(false)
      void loadData(searchQuery, rolFilter, currentPage)
    } catch (err) {
      setBorrarError(err instanceof Error ? err.message : "No se pudo eliminar el usuario.")
    } finally {
      setBorrarLoading(false)
    }
  }

  // Handler dentro del componente UsuariosPage, junto a los demás handlers


  // Handler dentro del componente UsuariosPage, junto a los demás handlers

const handleExport = () => {
  const rows = usuarios.map((u) => ({
    "Nombre": u.nombre_usuario,
    "Apellidos": u.apellidos,
    "Email": u.email,
    "Rol": u.tipo_usuario,
    "Estado": u.estado ? "Activo" : "Inactivo",
    "Entidad": u.entidad_nombre ?? "",
    "Última actualización": new Date(u.fecha_actualizacion).toLocaleString("es-CO"),
  }));

  const fecha = new Date().toISOString().slice(0, 10);
  exportToCSV(rows, `usuarios_sinapsis_${fecha}.csv`);
};
/*
const handleExport = async () => {
  setExportLoading(true)
  setError(null)
  try {
    const todosLosUsuarios = await fetchUsuariosSinLimite(searchQuery, rolFilter)

    const rows = todosLosUsuarios.map((u) => ({
      "Nombre": u.nombre_usuario,
      "Apellidos": u.apellidos,
      "Email": u.email,
      "Rol": u.tipo_usuario,
      "Estado": u.estado ? "Activo" : "Inactivo",
      "Entidad": u.entidad_nombre ?? "",
      "Última actualización": new Date(u.fecha_actualizacion).toLocaleString("es-CO"),
    }))

    const fecha = new Date().toISOString().slice(0, 10)
    exportToCSV(rows, `usuarios_sinapsis_${fecha}.csv`)
  } catch (err) {
    if (err instanceof Error && err.message !== "forbidden") {
      setError("No se pudo exportar la lista de usuarios.")
    }
  } finally {
    setExportLoading(false)
  }
}
*/
  const inputClass =
    "w-full rounded border border-line bg-field px-3 py-2 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
  const labelClass = "mb-1 block text-xs font-medium uppercase tracking-[0.4px] text-muted"

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
        <button
          onClick={openCrear}
          className="flex items-center gap-2 rounded bg-navy px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-navy-800"
        >
          <Users className="size-4" />
          Crear Usuario
        </button>
        <button
        //Usa la primer consulta encontrada de usuario
          onClick={() => openEditar(usuarios[0])}
          className="flex items-center gap-2 rounded bg-navy px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-navy-800"
        >
          <Users className="size-4" />
          Editar Usuario
        </button>
        <button
          onClick={() => openBorrar(usuarios[0])}
          className="flex items-center gap-2 rounded bg-navy px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-navy-800"
        >
          <Users className="size-4" />
          Eliminar Usuario
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
            <button
              onClick={handleExport}
              className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field"
            >
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
                    <button
                      onClick={() => openVer(u)}
                      className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-field hover:text-ink"
                    >
                      <Eye className="size-4" />
                    </button>
                    <button
                      onClick={() => openEditar(u)}
                      className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-field hover:text-teal"
                    >
                      <Pencil className="size-4" />
                    </button>
                    <button
                      onClick={() => openBorrar(u)}
                      className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-danger/10 hover:text-danger"
                    >
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

      {/* ------------------------------------------------------------------ */}
      {/* Modal: Crear Usuario                                              */}
      {/* ------------------------------------------------------------------ */}
      {showCrear && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden">
            <div className="flex items-center justify-between border-b border-line px-6 py-4">
              <h3 className="font-display text-lg font-semibold text-ink">
                Crear Usuario
              </h3>
              <button
                onClick={() => setShowCrear(false)}
                className="text-muted hover:text-ink"
              >
                <X className="size-5" />
              </button>
            </div>

            <form onSubmit={handleCrearSubmit} className="flex flex-col gap-4 overflow-y-auto px-6 py-5">
              {crearError && (
                <div className="rounded border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                  {crearError}
                </div>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Nombre</label>
                  <input
                    required
                    type="text"
                    value={crearForm.nombre_usuario}
                    onChange={(e) => setCrearForm(f => ({ ...f, nombre_usuario: e.target.value }))}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Apellidos</label>
                  <input
                    required
                    type="text"
                    value={crearForm.apellidos}
                    onChange={(e) => setCrearForm(f => ({ ...f, apellidos: e.target.value }))}
                    className={inputClass}
                  />
                </div>
              </div>

              <div>
                <label className={labelClass}>Correo electrónico</label>
                <input
                  required
                  type="email"
                  value={crearForm.email}
                  onChange={(e) => setCrearForm(f => ({ ...f, email: e.target.value }))}
                  className={inputClass}
                />
              </div>

              <div>
                <label className={labelClass}>Contraseña</label>
                <input
                  required
                  type="password"
                  minLength={8}
                  value={crearForm.contrasena}
                  onChange={(e) => setCrearForm(f => ({ ...f, contrasena: e.target.value }))}
                  className={inputClass}
                  placeholder="Mínimo 8 caracteres"
                />
              </div>

              <div>
                <label className={labelClass}>Rol</label>
                <select
                  value={crearForm.tipo_usuario}
                  onChange={(e) => setCrearForm(f => ({ ...f, tipo_usuario: e.target.value }))}
                  className={inputClass}
                >
                  <option value="paciente">Paciente</option>
                  <option value="medico">Médico</option>
                  <option value="admin_entidad">Admin Entidad</option>
                  <option value="admin_plataforma">Admin Plataforma</option>
                </select>
              </div>

              {/* Campos condicionales: Médico */}
              {crearForm.tipo_usuario === "medico" && (
                <div className="flex flex-col gap-4 rounded border border-line bg-shell p-4">
                  <p className="text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Datos de médico
                  </p>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className={labelClass}>Número de documento</label>
                      <input
                        required
                        type="text"
                        value={crearForm.numero_documento}
                        onChange={(e) => setCrearForm(f => ({ ...f, numero_documento: e.target.value }))}
                        className={inputClass}
                      />
                    </div>
                    <div>
                      <label className={labelClass}>Número colegiado</label>
                      <input
                        required
                        type="text"
                        value={crearForm.numero_colegiado}
                        onChange={(e) => setCrearForm(f => ({ ...f, numero_colegiado: e.target.value }))}
                        className={inputClass}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className={labelClass}>Especialidad</label>
                      <input
                        required
                        type="text"
                        value={crearForm.especialidad}
                        onChange={(e) => setCrearForm(f => ({ ...f, especialidad: e.target.value }))}
                        className={inputClass}
                      />
                    </div>
                    <div>
                      <label className={labelClass}>Años de experiencia</label>
                      <input
                        type="number"
                        min={0}
                        value={crearForm.experiencia_anios}
                        onChange={(e) => setCrearForm(f => ({ ...f, experiencia_anios: e.target.value }))}
                        className={inputClass}
                      />
                    </div>
                  </div>
                  <div>
                    <label className={labelClass}>Entidad</label>
                    <select
                      required
                      value={crearForm.entidad_id}
                      onChange={(e) => setCrearForm(f => ({ ...f, entidad_id: e.target.value }))}
                      className={inputClass}
                    >
                      <option value="">Selecciona una entidad</option>
                      {entidades.map(ent => (
                        <option key={ent.id} value={ent.id}>{ent.nombre_entidad}</option>
                      ))}
                    </select>
                  </div>
                </div>
              )}

              {/* Campos condicionales: Paciente */}
              {crearForm.tipo_usuario === "paciente" && (
                <div className="flex flex-col gap-4 rounded border border-line bg-shell p-4">
                  <p className="text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Datos de paciente
                  </p>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className={labelClass}>Tipo de documento</label>
                      <select
                        value={crearForm.tipo_documento}
                        onChange={(e) => setCrearForm(f => ({ ...f, tipo_documento: e.target.value }))}
                        className={inputClass}
                      >
                        <option value="CC">Cédula de ciudadanía</option>
                        <option value="TI">Tarjeta de identidad</option>
                        <option value="CE">Cédula de extranjería</option>
                        <option value="RC">Registro civil</option>
                        <option value="PA">Pasaporte</option>
                      </select>
                    </div>
                    <div>
                      <label className={labelClass}>Número de documento</label>
                      <input
                        required
                        type="text"
                        value={crearForm.numero_documento}
                        onChange={(e) => setCrearForm(f => ({ ...f, numero_documento: e.target.value }))}
                        className={inputClass}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className={labelClass}>Fecha de nacimiento</label>
                      <input
                        required
                        type="date"
                        value={crearForm.fecha_nacimiento}
                        onChange={(e) => setCrearForm(f => ({ ...f, fecha_nacimiento: e.target.value }))}
                        className={inputClass}
                      />
                    </div>
                    <div>
                      <label className={labelClass}>Sexo</label>
                      <select
                        value={crearForm.sexo}
                        onChange={(e) => setCrearForm(f => ({ ...f, sexo: e.target.value }))}
                        className={inputClass}
                      >
                        <option value="">Seleccionar</option>
                        <option value="M">Masculino</option>
                        <option value="F">Femenino</option>
                        <option value="Otro">Otro</option>
                      </select>
                    </div>
                  </div>
                  <div>
                    <label className={labelClass}>Teléfono</label>
                    <input
                      type="text"
                      value={crearForm.telefono}
                      onChange={(e) => setCrearForm(f => ({ ...f, telefono: e.target.value }))}
                      className={inputClass}
                    />
                  </div>
                </div>
              )}

              {/* Campos condicionales: Admin Entidad */}
              {crearForm.tipo_usuario === "admin_entidad" && (
                <div className="flex flex-col gap-4 rounded border border-line bg-shell p-4">
                  <p className="text-xs font-medium uppercase tracking-[0.4px] text-muted">
                    Datos de administrador de entidad
                  </p>
                  <div>
                    <label className={labelClass}>Entidad</label>
                    <select
                      required
                      value={crearForm.entidad_id}
                      onChange={(e) => setCrearForm(f => ({ ...f, entidad_id: e.target.value }))}
                      className={inputClass}
                    >
                      <option value="">Selecciona una entidad</option>
                      {entidades.map(ent => (
                        <option key={ent.id} value={ent.id}>{ent.nombre_entidad}</option>
                      ))}
                    </select>
                  </div>
                </div>
              )}

              <div className="flex justify-end gap-3 border-t border-line pt-4">
                <button
                  type="button"
                  onClick={() => setShowCrear(false)}
                  className="rounded border border-line px-4 py-2 text-sm text-slate hover:bg-field"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={crearLoading}
                  className="rounded bg-navy px-4 py-2 text-sm font-medium text-white hover:bg-navy-800 disabled:opacity-50"
                >
                  {crearLoading ? "Creando..." : "Crear Usuario"}
                </button>
              </div>
            </form>
          </Card>
        </div>
      )}

      {/* ------------------------------------------------------------------ */}
      {/* Modal: Ver Usuario                                                */}
      {/* ------------------------------------------------------------------ */}
      {showVer && usuarioVer && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-md flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">
                Detalle del usuario
              </h3>
              <button onClick={() => setShowVer(false)} className="text-muted hover:text-ink">
                <X className="size-5" />
              </button>
            </div>
            <div className="flex flex-col gap-3 rounded border border-line bg-shell p-4 text-sm">
              <DetailRow label="Nombre completo" value={`${usuarioVer.nombre_usuario} ${usuarioVer.apellidos}`} />
              <DetailRow label="Correo" value={usuarioVer.email} />
              <DetailRow label="Rol" value={usuarioVer.tipo_usuario} />
              <DetailRow label="Entidad" value={usuarioVer.entidad_nombre ?? "—"} />
              <DetailRow label="Estado" value={usuarioVer.estado ? "Activo" : "Inactivo"} />
              <DetailRow label="Última actualización" value={new Date(usuarioVer.fecha_actualizacion).toLocaleString("es-CO")} />
            </div>
            <div className="flex justify-end">
              <button
                onClick={() => setShowVer(false)}
                className="rounded bg-navy px-4 py-2 text-sm text-white hover:bg-navy-800"
              >
                Cerrar
              </button>
            </div>
          </Card>
        </div>
      )}

      {/* ------------------------------------------------------------------ */}
      {/* Modal: Editar Usuario (info básica) — PUT al backend              */}
      {/* ------------------------------------------------------------------ */}
      {showEditar && usuarioEditar && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-md flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">
                Editar Usuario
              </h3>
              <button onClick={() => setShowEditar(false)} className="text-muted hover:text-ink">
                <X className="size-5" />
              </button>
            </div>

            <form onSubmit={handleEditarSubmit} className="flex flex-col gap-4">
              {editarError && (
                <div className="rounded border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                  {editarError}
                </div>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Nombre</label>
                  <input
                    required
                    type="text"
                    value={editarForm.nombre_usuario}
                    onChange={(e) => setEditarForm(f => ({ ...f, nombre_usuario: e.target.value }))}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Apellidos</label>
                  <input
                    required
                    type="text"
                    value={editarForm.apellidos}
                    onChange={(e) => setEditarForm(f => ({ ...f, apellidos: e.target.value }))}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label className={labelClass}>Correo electrónico</label>
                <input
                  required
                  type="email"
                  value={editarForm.email}
                  onChange={(e) => setEditarForm(f => ({ ...f, email: e.target.value }))}
                  className={inputClass}
                />
              </div>
              <div>
                <label className={labelClass}>Estado</label>
                <select
                  value={editarForm.estado ? "activo" : "inactivo"}
                  onChange={(e) => setEditarForm(f => ({ ...f, estado: e.target.value === "activo" }))}
                  className={inputClass}
                >
                  <option value="activo">Activo</option>
                  <option value="inactivo">Inactivo</option>
                </select>
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowEditar(false)}
                  className="rounded border border-line px-4 py-2 text-sm text-slate hover:bg-field"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={editarLoading}
                  className="rounded bg-navy px-4 py-2 text-sm font-medium text-white hover:bg-navy-800 disabled:opacity-50"
                >
                  {editarLoading ? "Guardando..." : "Guardar Cambios"}
                </button>
              </div>
            </form>
          </Card>
        </div>
      )}

      {/* ------------------------------------------------------------------ */}
      {/* Modal: Confirmar Borrado — DELETE al backend (soft delete)        */}
      {/* ------------------------------------------------------------------ */}
      {showBorrar && usuarioBorrar && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-sm flex-col gap-4 p-6">
            <div className="flex items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-danger/10 text-danger">
                <Trash2 className="size-5" />
              </div>
              <h3 className="font-display text-lg font-semibold text-ink">
                Eliminar Usuario
              </h3>
            </div>
            {borrarError && (
              <div className="rounded border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {borrarError}
              </div>
            )}
            <p className="text-sm text-slate">
              ¿Estás seguro que deseas eliminar a{" "}
              <span className="font-medium text-navy-800">
                {usuarioBorrar.nombre_usuario} {usuarioBorrar.apellidos}
              </span>
              ? El usuario quedará desactivado en la plataforma.
            </p>
            <div className="flex justify-end gap-3 pt-2">
              <button
                onClick={() => setShowBorrar(false)}
                className="rounded border border-line px-4 py-2 text-sm text-slate hover:bg-field"
              >
                Cancelar
              </button>
              <button
                onClick={handleBorrarConfirm}
                disabled={borrarLoading}
                className="rounded bg-danger px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
              >
                {borrarLoading ? "Eliminando..." : "Eliminar"}
              </button>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Helper de fila para el modal "Ver"
// ---------------------------------------------------------------------------
function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[10px] uppercase tracking-[0.6px] text-muted">{label}</p>
      <p className="mt-0.5 text-sm text-navy-800">{value}</p>
    </div>
  )
}