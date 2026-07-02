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
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatCard } from "@/components/ui/stat-card";

const USUARIOS = [
  {
    initials: "MT",
    name: "Marcus Thorne",
    id: "USR-00124",
    email: "m.thorne@centralips.co",
    role: "Médico",
    entity: "Central General IPS",
    status: "Activo" as const,
  },
  {
    initials: "SC",
    name: "Sarah Chen",
    id: "USR-00231",
    email: "s.chen@nortelab.co",
    role: "Administrador Entidad",
    entity: "Norte Diagnostic Lab",
    status: "Activo" as const,
  },
  {
    initials: "JM",
    name: "James Miller",
    id: "USR-00318",
    email: "j.miller@pacificpediatric.co",
    role: "Médico",
    entity: "Pacific Pediatric Clinic",
    status: "Inactivo" as const,
  },
  {
    initials: "CO",
    name: "Clara Ospina",
    id: "USR-00445",
    email: "c.ospina@medlink.co",
    role: "Enfermera",
    entity: "MedLink Pharmacy Dist.",
    status: "Suspendido" as const,
  },
];

const STATUS_TONE = {
  Activo: "success" as const,
  Inactivo: "neutral" as const,
  Suspendido: "danger" as const,
};

export default function UsuariosPage() {
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
          value="1,248"
          icon={<Users className="size-6" />}
        />
        <StatCard
          label="Entidades Registradas"
          value="160"
          icon={<Building2 className="size-6" />}
        />
        <StatCard
          label="Usuarios Activos"
          value="1,102"
          icon={<ShieldCheck className="size-6" />}
        />
        <StatCard
          label="Cuentas Suspendidas"
          value="18"
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
          {["Todos los roles", "Todas las entidades", "Todos los estados"].map(
            (label) => (
              <select
                key={label}
                className="rounded border border-line bg-field px-3 py-2 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal"
                defaultValue=""
              >
                <option value="">{label}</option>
              </select>
            ),
          )}
          <div className="ml-auto flex gap-2">
            <button className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <Download className="size-4" />
            </button>
          </div>
        </div>
      </Card>

      {/* Tabla */}
      <Card className="overflow-hidden">
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
            {USUARIOS.map((u) => (
              <tr key={u.id} className="border-t border-line hover:bg-field/30">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <span className="flex size-8 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-xs font-medium text-teal">
                      {u.initials}
                    </span>
                    <div>
                      <p className="font-medium text-navy-800">{u.name}</p>
                      <p className="text-xs text-muted">{u.id}</p>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-slate">{u.email}</td>
                <td className="px-6 py-4">
                  <Badge tone="info">{u.role}</Badge>
                </td>
                <td className="px-6 py-4 text-slate">{u.entity}</td>
                <td className="px-6 py-4">
                  <Badge tone={STATUS_TONE[u.status]}>{u.status}</Badge>
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
          </tbody>
        </table>

        {/* Paginación */}
        <div className="flex items-center justify-between border-t border-line px-6 py-4">
          <p className="text-sm text-muted">
            Mostrando 1–4 de 1,248 usuarios
          </p>
          <div className="flex items-center gap-1">
            <button className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <ChevronLeft className="size-4" />
            </button>
            {[1, 2, 3].map((p) => (
              <button
                key={p}
                className={`flex size-8 items-center justify-center rounded border text-sm transition-colors ${
                  p === 1
                    ? "border-teal bg-teal text-white"
                    : "border-line text-slate hover:bg-field"
                }`}
              >
                {p}
              </button>
            ))}
            <span className="px-1 text-muted">…</span>
            <button className="flex size-8 items-center justify-center rounded border border-line text-sm text-slate transition-colors hover:bg-field">
              32
            </button>
            <button className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <ChevronRight className="size-4" />
            </button>
          </div>
        </div>
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
  );
}
