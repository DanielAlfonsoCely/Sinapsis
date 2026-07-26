"use client";

import { useState, useEffect } from "react";
import {
  Users,
  Building2,
  ShieldAlert,
  FileText,
  Eye,
  ChevronRight,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatCard } from "@/components/ui/stat-card";
import Link from "next/link";

// ---------------------------------------------------------------------------
// Tipos
// ---------------------------------------------------------------------------
type ImportanceLevel = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

type AuditLogEntry = {
  id: string;
  usuario_id: string;
  usuario_nombre: string;
  usuario_email: string;
  tipo_operacion: string;
  tabla_afectada: string;
  registro_id: string | null;
  ip_origen: string | null;
  detalles: string | null;
  fecha_operacion: string;
  gravedad: ImportanceLevel;
};

type Entidad = {
  id: string;
  nombre_entidad: string;
  tipo_entidad: string;
  nit: string;
  ciudad: string | null;
  estado: boolean;
  fecha_creacion: string;
};

type Stats = {
  total_consultas: number;
  total_pacientes_activos: number;
  total_usuarios_activos: number;
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const SEVERITY_STYLES: Record<ImportanceLevel, { tone: "danger" | "warning" | "info" | "neutral"; label: string }> = {
  CRITICAL: { tone: "danger",   label: "CRÍTICO" },
  HIGH:     { tone: "warning",  label: "ALTO" },
  MEDIUM:   { tone: "info",     label: "MEDIO" },
  LOW:      { tone: "neutral",  label: "BAJO" },
};

const TABLAS_LEGIBLES: Record<string, string> = {
  historia_clinica: "Historia Clínica",
  consulta:         "Consulta",
  paciente:         "Paciente",
  usuario:          "Usuario",
  entidad:          "Entidad",
  cita:             "Cita",
  medico:           "Médico",
};

const OPERACION_LABELS: Record<string, string> = {
  crear:            "Crear",
  actualizar:       "Actualizar",
  eliminar:         "Eliminar",
  consultar:        "Consultar",
  exportar:         "Exportar",
  cambiar_permisos: "Cambiar Permisos",
  usar_ia:          "Usar IA",
};

function formatFecha(iso: string) {
  return new Date(iso + (iso.endsWith("Z") ? "" : "Z")).toLocaleString("es-CO", {
    timeZone: "America/Bogota",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function getToken() {
  return document.cookie.split("; ").find((c) => c.startsWith("token="))?.split("=")[1];
}

function authHeaders() {
  return { Authorization: `Bearer ${getToken()}` };
}

// ---------------------------------------------------------------------------
// Helper fila modal
// ---------------------------------------------------------------------------
function Row({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <p className="text-[10px] uppercase tracking-[0.6px] text-muted">{label}</p>
      <p className={`mt-0.5 text-navy-800 ${mono ? "font-mono text-xs" : "text-sm"}`}>{value}</p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Componente principal
// ---------------------------------------------------------------------------
export default function AdminDashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [entidades, setEntidades] = useState<Entidad[]>([]);
  const [alertas, setAlertas] = useState<AuditLogEntry[]>([]);
  const [totalAlertas, setTotalAlertas] = useState(0);
  const [detalle, setDetalle] = useState<AuditLogEntry | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const headers = authHeaders();

    Promise.all([
      fetch("http://localhost:8080/api/v1/admin/stats", { headers }).then((r) => r.json()),
      fetch("http://localhost:8080/api/v1/admin/entidades?limit=4", { headers }).then((r) => r.json()),
      fetch("http://localhost:8080/api/v1/admin/auditoria/criticas?limit=5", { headers }).then((r) => r.json()),
    ])
      .then(([statsData, entidadesData, alertasData]) => {
        setStats(statsData as Stats);
        setEntidades((entidadesData as { entidades: Entidad[] }).entidades ?? []);
        setAlertas((alertasData as { registros: AuditLogEntry[] }).registros ?? []);
        setTotalAlertas((alertasData as { total: number }).total ?? 0);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const entidadesActivas = entidades.filter((e) => e.estado).length;
  const entidadesInactivas = entidades.filter((e) => !e.estado).length;

  return (
    <div className="flex flex-col gap-6">
      {/* Modal detalle alerta */}
      {detalle && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-lg flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">Detalle del registro</h3>
              <button onClick={() => setDetalle(null)} className="text-muted hover:text-ink">✕</button>
            </div>
            <div className="flex flex-col gap-3 rounded border border-line bg-shell p-4 text-sm">
              <Row label="ID Registro" value={detalle.id} mono />
              <Row label="Usuario" value={`${detalle.usuario_nombre} (${detalle.usuario_email})`} />
              <Row label="Operación" value={OPERACION_LABELS[detalle.tipo_operacion] ?? detalle.tipo_operacion} />
              <Row label="Tabla afectada" value={TABLAS_LEGIBLES[detalle.tabla_afectada] ?? detalle.tabla_afectada} />
              <Row label="Severidad" value={SEVERITY_STYLES[detalle.gravedad]?.label ?? detalle.gravedad} />
              {detalle.registro_id && <Row label="ID del registro afectado" value={detalle.registro_id} mono />}
              <Row label="Fecha y hora" value={formatFecha(detalle.fecha_operacion)} />
              {detalle.detalles && <Row label="Detalles" value={detalle.detalles} />}
            </div>
            <p className="text-xs text-muted">
              Este registro es inmutable — no puede ser modificado ni eliminado por ningún rol.
            </p>
            <div className="flex justify-end">
              <button
                onClick={() => setDetalle(null)}
                className="rounded bg-navy px-4 py-2 text-sm text-white hover:bg-navy-800"
              >
                Cerrar
              </button>
            </div>
          </Card>
        </div>
      )}

      {/* Encabezado */}
      <div>
        <h2 className="font-display text-3xl font-semibold text-ink">Master Dashboard</h2>
        <p className="mt-1 text-sm text-label">
          <span className="text-muted">Platform Admin</span>
          <span className="mx-2 text-muted">›</span>
          System Overview
        </p>
      </div>

      {/* Tarjetas resumen */}
      <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
        <StatCard
          label="Usuarios Activos"
          value={loading ? "—" : (stats?.total_usuarios_activos.toLocaleString("es-CO") ?? "—")}
          icon={<Users className="size-6" />}
        />
        <StatCard
          label="Pacientes Activos"
          value={loading ? "—" : (stats?.total_pacientes_activos.toLocaleString("es-CO") ?? "—")}
          icon={<Users className="size-6" />}
        />
        <StatCard
          label="Consultas Totales"
          value={loading ? "—" : (stats?.total_consultas.toLocaleString("es-CO") ?? "—")}
          icon={<FileText className="size-6" />}
        />
        <StatCard
          label="Alertas de Seguridad"
          value={loading ? "—" : totalAlertas.toString().padStart(2, "0")}
          hint={totalAlertas > 0 ? "Requiere revisión" : "Sin alertas críticas"}
          valueClassName={totalAlertas > 0 ? "text-danger" : undefined}
          icon={<ShieldAlert className={`size-6 ${totalAlertas > 0 ? "text-danger" : ""}`} />}
        />
      </div>

      {/* Bento: entidades + alertas */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Entidades de Salud */}
        <Card className="p-6">
          <h3 className="mb-1 font-display text-lg font-semibold text-ink">Entidades de Salud</h3>
          <p className="mb-5 text-sm text-slate">
            {loading ? "Cargando..." : `${entidadesActivas} activas · ${entidadesInactivas} inactivas`}
          </p>
          <div className="flex flex-col gap-3">
            {entidades.slice(0, 4).map((ent) => (
              <div key={ent.id} className="flex items-center gap-3 rounded border border-line p-3">
                <div className="flex size-8 items-center justify-center rounded bg-navy">
                  <Building2 className="size-4 text-[#76aecc]" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="truncate text-sm font-medium text-navy-800">{ent.nombre_entidad}</p>
                  <p className="text-xs text-muted">{ent.tipo_entidad}{ent.ciudad ? ` · ${ent.ciudad}` : ""}</p>
                </div>
                <span
                  className={`size-2.5 shrink-0 rounded-full ring-4 ring-current/10 ${ent.estado ? "bg-success" : "bg-danger"}`}
                />
              </div>
            ))}
            {!loading && entidades.length === 0 && (
              <p className="text-sm text-muted text-center py-4">No hay entidades registradas.</p>
            )}
          </div>
          <Link href="/admin/entidades">
            <button className="mt-4 w-full rounded border border-teal py-2 text-sm text-teal transition-colors hover:bg-teal/5">
              Revisar Entidades de Salud
            </button>
          </Link>
        </Card>

        {/* Alertas críticas — ocupa 2 columnas */}
        <Card className="col-span-2 overflow-hidden">
          <div className="flex items-center justify-between border-b border-line p-6">
            <div className="flex items-center gap-2">
              <ShieldAlert className="size-5 text-danger" />
              <h3 className="font-display text-xl font-semibold text-ink">Alertas Críticas del Sistema</h3>
            </div>
            <Link href="/admin/registros" className="flex items-center gap-1 text-sm text-teal hover:text-teal-700">
              Ver todas <ChevronRight className="size-4" />
            </Link>
          </div>

          {loading ? (
            <p className="px-6 py-8 text-center text-sm text-muted">Cargando...</p>
          ) : alertas.length === 0 ? (
            <p className="px-6 py-8 text-center text-sm text-muted">No hay alertas críticas o altas registradas.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
                  <th className="px-6 py-4 font-normal">Timestamp</th>
                  <th className="px-6 py-4 font-normal">Severidad</th>
                  <th className="px-6 py-4 font-normal">Operación</th>
                  <th className="px-6 py-4 font-normal">Tabla</th>
                  <th className="px-6 py-4 font-normal text-center">Ver</th>
                </tr>
              </thead>
              <tbody>
                {alertas.map((alert) => {
                  const sev = SEVERITY_STYLES[alert.gravedad] ?? { tone: "neutral" as const, label: alert.gravedad };
                  return (
                    <tr key={alert.id} className="border-t border-line hover:bg-shell">
                      <td className="px-6 py-4 font-mono text-xs text-slate whitespace-nowrap">
                        {formatFecha(alert.fecha_operacion)}
                      </td>
                      <td className="px-6 py-4">
                        <Badge tone={sev.tone}>{sev.label}</Badge>
                      </td>
                      <td className="px-6 py-4 text-navy-800">
                        {OPERACION_LABELS[alert.tipo_operacion] ?? alert.tipo_operacion}
                      </td>
                      <td className="px-6 py-4 text-slate">
                        {TABLAS_LEGIBLES[alert.tabla_afectada] ?? alert.tabla_afectada}
                      </td>
                      <td className="px-6 py-4 text-center">
                        <button
                          onClick={() => setDetalle(alert)}
                          className="flex size-8 items-center justify-center rounded text-slate transition-colors hover:bg-field hover:text-teal mx-auto"
                        >
                          <Eye className="size-4" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </Card>
      </div>
    </div>
  );
}