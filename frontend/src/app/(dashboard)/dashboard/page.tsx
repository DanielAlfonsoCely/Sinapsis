"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Stethoscope,
  UserCheck,
  CheckCircle2,
  CalendarDays,
  ArrowUpRight,
  ChevronRight,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { StatCard } from "@/components/ui/stat-card";

type PacienteCita = {
  id: string;
  nombre_paciente: string;
  apellidos_paciente: string;
  numero_documento: string;
  fecha_hora: string; // de la cita de hoy
};

type Stats = {
  total: number;
  completadas: number;
  pendientes: number;
};

function initials(nombre: string, apellidos: string) {
  return `${nombre[0] ?? ""}${apellidos[0] ?? ""}`.toUpperCase();
}

function formatHora(iso: string) {
  try {
    // El backend retorna timestamps sin sufijo de timezone (asumidos UTC).
    // Añadimos "Z" para forzar interpretación UTC y mostramos en hora Colombia.
    const utc = iso.endsWith("Z") ? iso : iso + "Z";
    return new Date(utc).toLocaleTimeString("es-CO", {
      hour: "2-digit",
      minute: "2-digit",
      timeZone: "America/Bogota",
    });
  } catch {
    return "";
  }
}

export default function DashboardPage() {
  const [nombreMedico, setNombreMedico] = useState("");
  const [esTriage, setEsTriage] = useState(false);
  const [stats, setStats] = useState<Stats>({ total: 0, completadas: 0, pendientes: 0 });
  const [pacientes, setPacientes] = useState<PacienteCita[]>([]);
  const [loading, setLoading] = useState(true);

  const hoy = new Date().toLocaleDateString("es-CO", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });

  useEffect(() => {
    try {
      const raw = localStorage.getItem("usuario");
      if (raw) {
        const u = JSON.parse(raw);
        setNombreMedico(u.apellidos ?? "");
        const esp: string = u.especialidad ?? "";
        setEsTriage(
          esp.toLowerCase().includes("triage") || esp.toLowerCase().includes("triagge")
        );
      }
    } catch {}
  }, []);

  useEffect(() => {
    async function fetchData() {
      const token = localStorage.getItem("token");
      if (!token) return;

      try {
        // Citas de hoy: reutilizamos el endpoint de pacientes que ya filtra correctamente
        const [citasRes] = await Promise.all([
          fetch("http://localhost:8080/api/v1/citas/hoy", {
            headers: { Authorization: `Bearer ${token}` },
          }),
        ]);

        if (citasRes.ok) {
          const data = await citasRes.json();
          const citas: { estado: string; paciente: PacienteCita; fecha_hora: string }[] =
            data.citas ?? [];
          const total = citas.length;
          const completadas = citas.filter((c) => c.estado === "completada").length;
          const pendientes = citas.filter(
            (c) => c.estado === "programada" || c.estado === "en_curso"
          ).length;
          setStats({ total, completadas, pendientes });

          // Solo mostrar pacientes en tabla si es triage
          if (!esTriage) {
            setPacientes(
              citas.slice(0, 5).map((c) => ({
                ...c.paciente,
                fecha_hora: c.fecha_hora,
              }))
            );
          }
        }
      } catch {
        // Si el endpoint aún no existe, dejamos los stats en 0
      } finally {
        setLoading(false);
      }
    }

    fetchData();
  }, [esTriage]);

  return (
    <div className="flex flex-col gap-6">
      {/* Banner de bienvenida */}
      <div className="relative flex items-center overflow-hidden rounded-[var(--radius)] bg-gradient-to-r from-navy to-[#002954] p-8">
        <div className="absolute -right-8 -top-8 size-48 rounded-full bg-white/5" />
        <div className="relative flex flex-col gap-1">
          <h2 className="font-display text-2xl font-semibold text-white">
            Bienvenido de nuevo{nombreMedico ? `, Dr. ${nombreMedico}` : ""}
          </h2>
          <p className="text-[#d5e3ff]">
            {stats.total > 0
              ? `Tiene ${stats.total} paciente${stats.total !== 1 ? "s" : ""} programado${stats.total !== 1 ? "s" : ""} para hoy.`
              : "No hay pacientes programados para hoy."}
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
        <h3 className="font-display text-lg font-semibold text-ink">Resumen del día</h3>
        <Badge tone="info" className="bg-[#c1e8ff] text-[#001e2b]">
          {hoy}
        </Badge>
      </div>

      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <StatCard
          label="Consultas hoy"
          value={loading ? "—" : stats.total}
          hint={`${stats.completadas} completada${stats.completadas !== 1 ? "s" : ""}`}
          icon={<Stethoscope className="size-6" />}
        />
        <StatCard
          label="Pendientes"
          value={loading ? "—" : stats.pendientes}
          icon={<UserCheck className="size-6" />}
        />
        <StatCard
          label="Completadas hoy"
          value={loading ? "—" : stats.completadas}
          valueClassName="text-success"
          icon={<CheckCircle2 className="size-6 text-success" />}
        />
      </div>

      {/* Tabla de pacientes — solo visible para médico de triage */}
      {!esTriage && (
        <Card className="overflow-hidden">
          <div className="flex items-center justify-between border-b border-line p-6">
            <h3 className="font-display text-xl font-semibold text-ink">Pacientes</h3>
            <Link href="/pacientes" className="flex items-center gap-1 text-sm text-teal">
              Ver todos <ChevronRight className="size-4" />
            </Link>
          </div>
          {pacientes.length === 0 ? (
            <p className="px-6 py-8 text-sm text-slate">
              No hay pacientes con cita programada para hoy.
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
                  <th className="px-6 py-4 font-normal">Paciente</th>
                  <th className="px-6 py-4 font-normal">Documento</th>
                  <th className="px-6 py-4 font-normal">Hora</th>
                  <th className="px-6 py-4 font-normal text-right">Acción</th>
                </tr>
              </thead>
              <tbody>
                {pacientes.map((p) => (
                  <tr key={p.id} className="border-t border-line">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <span className="flex size-8 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-xs text-teal">
                          {initials(p.nombre_paciente, p.apellidos_paciente)}
                        </span>
                        <span className="text-navy-800">
                          {p.nombre_paciente} {p.apellidos_paciente}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 font-mono text-navy-800">{p.numero_documento}</td>
                    <td className="px-6 py-4 text-slate">{formatHora(p.fecha_hora)}</td>
                    <td className="px-6 py-4 text-right">
                      <Link
                        href="/pacientes"
                        className="inline-flex text-teal hover:text-teal-700"
                      >
                        <ArrowUpRight className="size-[18px]" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
      )}
    </div>
  );
}