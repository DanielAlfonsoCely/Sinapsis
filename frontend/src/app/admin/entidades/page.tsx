import {
  Building2,
  Users,
  Activity,
  BarChart2,
  Plus,
  Search,
  MapPin,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const ENTIDADES = [
  {
    initials: "CG",
    name: "Central General IPS",
    nit: "900.123.456-7",
    location: "Medellín, CO",
    admin: "Marc Jacobs",
    patients: "12,402",
    apiUsage: 75,
    status: "Activa" as const,
    plan: "Enterprise",
  },
  {
    initials: "ND",
    name: "Norte Diagnostic Lab",
    nit: "800.987.654-3",
    location: "Bogotá, CO",
    admin: "Sarah Connor",
    patients: "5,120",
    apiUsage: 42,
    status: "Activa" as const,
    plan: "Pro",
  },
  {
    initials: "PP",
    name: "Pacific Pediatric Clinic",
    nit: "700.456.789-1",
    location: "Cali, CO",
    admin: "Roberto Gomez",
    patients: "2,880",
    apiUsage: 31,
    status: "Offline" as const,
    plan: "Basic",
  },
];

const STATUS_TONE = {
  Activa: "success" as const,
  Offline: "danger" as const,
  Suspendida: "warning" as const,
};

export default function EntidadesPage() {
  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between border-b border-line pb-5">
        <div className="flex items-center gap-3">
          <Building2 className="size-5 text-teal" />
          <h2 className="font-display text-2xl font-semibold text-ink">
            Gestión de Entidades de Salud
          </h2>
        </div>
        <button className="flex items-center gap-2 rounded bg-navy px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-navy-800">
          <Plus className="size-4" />
          Registrar Entidad
        </button>
      </div>

      {/* Stats del sistema */}
      <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
        {[
          { label: "TOTAL ENTIDADES", value: "160", sub: "registradas" },
          { label: "USUARIOS ACTIVOS GLOBALES", value: "42,891", sub: "en línea" },
          { label: "SALUD DEL SISTEMA", value: "Óptima", sub: "todos los nodos" },
          { label: "CARGA PROMEDIO API", value: "68%", sub: "uso actual" },
        ].map(({ label, value, sub }) => (
          <Card key={label} className="p-5">
            <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
              {label}
            </p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">
              {value}
            </p>
            <p className="text-xs text-label">{sub}</p>
          </Card>
        ))}
      </div>

      {/* Buscador */}
      <Card className="p-4">
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted" />
            <input
              type="text"
              placeholder="Buscar por NIT, nombre de entidad o ciudad..."
              className="h-10 w-full rounded border border-line bg-field pl-9 pr-4 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
            />
          </div>
          <button className="flex items-center gap-2 rounded border border-navy px-4 py-2 text-sm text-navy transition-colors hover:bg-navy hover:text-white">
            <BarChart2 className="size-4" />
            Exportar Reporte
          </button>
          <div className="flex items-center gap-2 border-l border-line pl-3">
            <button className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <BarChart2 className="size-4" />
            </button>
            <button className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <Users className="size-4" />
            </button>
          </div>
        </div>
      </Card>

      {/* Cards de entidades */}
      <div className="flex flex-col gap-4">
        {ENTIDADES.map((ent) => (
          <Card key={ent.nit} className="p-6">
            <div className="flex items-start gap-6">
              {/* Ícono entidad */}
              <div className="flex size-20 shrink-0 items-center justify-center rounded border border-line bg-field">
                <span className="font-display text-xl font-bold text-teal">
                  {ent.initials}
                </span>
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="flex items-center gap-3">
                      <h3 className="font-display text-lg font-semibold text-ink">
                        {ent.name}
                      </h3>
                      <Badge tone={STATUS_TONE[ent.status]}>{ent.status}</Badge>
                      <Badge tone="neutral">{ent.plan}</Badge>
                    </div>
                    <div className="mt-1 flex items-center gap-1 text-sm text-slate">
                      <MapPin className="size-3" />
                      {ent.location}
                    </div>
                  </div>
                </div>

                {/* Métricas */}
                <div className="mt-4 grid grid-cols-4 gap-4 border-t border-line pt-4">
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
                      UBICACIÓN
                    </p>
                    <p className="mt-0.5 text-sm text-navy-800">{ent.location}</p>
                  </div>
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
                      ADMINISTRADOR
                    </p>
                    <p className="mt-0.5 text-sm text-navy-800">{ent.admin}</p>
                  </div>
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
                      PACIENTES ACTIVOS
                    </p>
                    <p className="mt-0.5 text-sm text-navy-800">{ent.patients}</p>
                  </div>
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
                      USO DE API
                    </p>
                    <div className="mt-1 flex items-center gap-2">
                      <span className="text-sm text-navy-800">{ent.apiUsage}%</span>
                      <div className="h-1.5 flex-1 rounded-full bg-field">
                        <div
                          className="h-full rounded-full bg-teal"
                          style={{ width: `${ent.apiUsage}%` }}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Paginación */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted">Mostrando 1–3 de 160 entidades</p>
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
          <button className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
            <ChevronRight className="size-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
