import Link from "next/link";
import {
  Stethoscope,
  UserCheck,
  Clock,
  CalendarDays,
  ArrowUpRight,
  ChevronRight,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { StatCard } from "@/components/ui/stat-card";

const PATIENTS = [
  {
    initials: "RM",
    name: "Ricardo Mendoza",
    id: "1.024.556.782",
    time: "10:20 AM",
    status: "Atendido",
    tone: "success" as const,
  },
  {
    initials: "SP",
    name: "Sofía Palacio",
    id: "52.441.908",
    time: "10:40 AM",
    status: "En Consulta",
    tone: "info" as const,
  },
  {
    initials: "GC",
    name: "Gabriel Caicedo",
    id: "1.109.223.441",
    time: "11:00 AM",
    status: "En Espera",
    tone: "warning" as const,
  },
];

export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      {/* Banner de bienvenida */}
      <div className="relative flex items-center overflow-hidden rounded-[var(--radius)] bg-gradient-to-r from-navy to-[#002954] p-8">
        <div className="absolute -right-8 -top-8 size-48 rounded-full bg-white/5" />
        <div className="relative flex flex-col gap-1">
          <h2 className="font-display text-2xl font-semibold text-white">
            Bienvenido de nuevo, Dr. Pineda
          </h2>
          <p className="text-[#d5e3ff]">
            Usted tiene 18 pacientes programados para hoy.
          </p>
          <Link
            href="/agenda"
            className="mt-6 inline-flex w-fit items-center gap-1.5 rounded-sm bg-glow-1 px-4 py-2 text-sm font-medium text-teal-700"
          >
            <CalendarDays className="size-4" />
            Ver Agenda
          </Link>
        </div>
      </div>

      {/* Indicadores */}
      <div className="flex items-center justify-between">
        <h3 className="font-display text-lg font-semibold text-ink">
          Resumen del día
        </h3>
        <Badge tone="info" className="bg-[#c1e8ff] text-[#001e2b]">
          24 de Mayo, 2026
        </Badge>
      </div>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <StatCard
          label="Consultas hoy"
          value={18}
          hint="6 completadas"
          icon={<Stethoscope className="size-6" />}
        />
        <StatCard
          label="Pendientes"
          value={12}
          icon={<UserCheck className="size-6" />}
        />
        <StatCard
          label="Consultas en espera"
          value={6}
          valueClassName="text-warning"
          icon={<Clock className="size-6 text-warning" />}
        />
      </div>

      {/* Tabla de pacientes recientes */}
      <Card className="overflow-hidden">
        <div className="flex items-center justify-between border-b border-line p-6">
          <h3 className="font-display text-xl font-semibold text-ink">
            Pacientes
          </h3>
          <Link
            href="/pacientes"
            className="flex items-center gap-1 text-sm text-teal"
          >
            Ver todos <ChevronRight className="size-4" />
          </Link>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">Paciente</th>
              <th className="px-6 py-4 font-normal">ID</th>
              <th className="px-6 py-4 font-normal">Hora</th>
              <th className="px-6 py-4 font-normal">Estado</th>
              <th className="px-6 py-4 font-normal text-right">Acción</th>
            </tr>
          </thead>
          <tbody>
            {PATIENTS.map((p) => (
              <tr key={p.id} className="border-t border-line">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <span className="flex size-8 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-xs text-teal">
                      {p.initials}
                    </span>
                    <span className="text-navy-800">{p.name}</span>
                  </div>
                </td>
                <td className="px-6 py-4 font-mono text-navy-800">{p.id}</td>
                <td className="px-6 py-4 text-slate">{p.time}</td>
                <td className="px-6 py-4">
                  <Badge tone={p.tone}>{p.status}</Badge>
                </td>
                <td className="px-6 py-4 text-right">
                  <Link
                    href="/historia-clinica"
                    className="inline-flex text-teal hover:text-teal-700"
                  >
                    <ArrowUpRight className="size-[18px]" />
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
