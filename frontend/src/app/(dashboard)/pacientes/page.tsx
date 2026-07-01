import Link from "next/link";
import {
  Search,
  UserPlus,
  ArrowUpRight,
  Phone,
  Calendar,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const PATIENTS = [
  {
    initials: "RM",
    name: "Ricardo Mendoza",
    id: "1.024.556.782",
    phone: "311 429 8801",
    lastVisit: "24 May 2026",
    nextAppt: "07 Jun 2026",
    status: "Activo",
    tone: "success" as const,
  },
  {
    initials: "SP",
    name: "Sofía Palacio",
    id: "52.441.908",
    phone: "315 772 3390",
    lastVisit: "22 May 2026",
    nextAppt: "10 Jun 2026",
    status: "En Tratamiento",
    tone: "info" as const,
  },
  {
    initials: "GC",
    name: "Gabriel Caicedo",
    id: "1.109.223.441",
    phone: "300 118 4452",
    lastVisit: "20 May 2026",
    nextAppt: "29 May 2026",
    status: "En Espera",
    tone: "warning" as const,
  },
  {
    initials: "LV",
    name: "Lorena Vargas",
    id: "43.981.204",
    phone: "320 654 9901",
    lastVisit: "18 May 2026",
    nextAppt: "—",
    status: "Alta",
    tone: "neutral" as const,
  },
  {
    initials: "AO",
    name: "Andrés Ospina",
    id: "79.448.115",
    phone: "318 234 7765",
    lastVisit: "16 May 2026",
    nextAppt: "02 Jun 2026",
    status: "Urgente",
    tone: "danger" as const,
  },
  {
    initials: "CM",
    name: "Carolina Mora",
    id: "1.015.339.820",
    phone: "305 881 2233",
    lastVisit: "14 May 2026",
    nextAppt: "05 Jun 2026",
    status: "Activo",
    tone: "success" as const,
  },
];

const FILTERS = ["Todos", "Activo", "En Tratamiento", "Alta", "Urgente"];

export default function PacientesPage() {
  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink">
            Pacientes
          </h2>
          <p className="text-sm text-slate">
            {PATIENTS.length} pacientes registrados
          </p>
        </div>
        <Button size="sm">
          <UserPlus className="size-4" />
          Nuevo Paciente
        </Button>
      </div>

      {/* Barra de búsqueda y filtros */}
      <Card className="flex flex-col gap-4 p-5">
        <Input
          icon={<Search className="size-4" />}
          placeholder="Buscar por nombre o número de documento…"
        />
        <div className="flex gap-2 flex-wrap">
          {FILTERS.map((f, i) => (
            <button
              key={f}
              className={`rounded-full border px-3.5 py-1 text-xs font-medium transition-colors ${
                i === 0
                  ? "border-teal bg-teal text-white"
                  : "border-line bg-white text-slate hover:bg-field"
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </Card>

      {/* Tabla */}
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">Paciente</th>
              <th className="px-6 py-4 font-normal">
                <span className="flex items-center gap-1.5">
                  <Phone className="size-3" />
                  Contacto
                </span>
              </th>
              <th className="px-6 py-4 font-normal">
                <span className="flex items-center gap-1.5">
                  <Calendar className="size-3" />
                  Última consulta
                </span>
              </th>
              <th className="px-6 py-4 font-normal">Próxima cita</th>
              <th className="px-6 py-4 font-normal">Estado</th>
              <th className="px-6 py-4 font-normal text-right">Acción</th>
            </tr>
          </thead>
          <tbody>
            {PATIENTS.map((p) => (
              <tr key={p.id} className="border-t border-line hover:bg-shell">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-xs font-medium text-teal">
                      {p.initials}
                    </span>
                    <div>
                      <p className="font-medium text-navy-800">{p.name}</p>
                      <p className="font-mono text-xs text-muted">{p.id}</p>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-slate">{p.phone}</td>
                <td className="px-6 py-4 text-slate">{p.lastVisit}</td>
                <td className="px-6 py-4 text-slate">{p.nextAppt}</td>
                <td className="px-6 py-4">
                  <Badge tone={p.tone}>{p.status}</Badge>
                </td>
                <td className="px-6 py-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <Link
                      href="/consulta"
                      className="rounded border border-line px-2.5 py-1 text-xs text-slate hover:bg-field"
                    >
                      Consultar
                    </Link>
                    <Link
                      href="/historia-clinica"
                      className="inline-flex text-teal hover:text-teal-700"
                    >
                      <ArrowUpRight className="size-[18px]" />
                    </Link>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="flex items-center justify-between border-t border-line px-6 py-4 text-xs text-muted">
          <span>Mostrando 1–6 de 6 pacientes</span>
          <div className="flex gap-1">
            <button className="rounded border border-line px-3 py-1 hover:bg-field disabled:opacity-40" disabled>
              Anterior
            </button>
            <button className="rounded border border-line px-3 py-1 hover:bg-field disabled:opacity-40" disabled>
              Siguiente
            </button>
          </div>
        </div>
      </Card>
    </div>
  );
}
