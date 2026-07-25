import {
  Users,
  Building2,
  ShieldAlert,
  TrendingUp,
  ArrowUpRight,
  ChevronRight,
  Activity,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatCard } from "@/components/ui/stat-card";
import Link from "next/link";

const ALERTS = [
  {
    timestamp: "2023-10-24 14:32:11",
    severity: "CRITICAL" as const,
    event: "Failed MFA Brute Force",
    origin: "192.168.10.45",
  },
  {
    timestamp: "2023-10-24 14:15:04",
    severity: "HIGH" as const,
    event: "Unauthorized DB Access Trial",
    origin: "45.22.109.21",
  },
  {
    timestamp: "2023-10-24 13:58:22",
    severity: "CRITICAL" as const,
    event: "Admin Credentials Leak (Dark Web)",
    origin: "External Source",
  },
];

const NODE_STATUS = [
  { name: "Central General IPS", lat: "0.04ms", sync: "Activa", color: "bg-success" },
  { name: "Norte Diagnostic Lab", lat: "1.2ms", sync: "Ocupada", color: "bg-warning" },
  { name: "Pacific Pediatric Clinic", lat: "--", sync: "OFFLINE", color: "bg-danger" },
  { name: "MedLink Pharmacy Dist.", lat: "0.12ms", sync: "Activa", color: "bg-success" },
];

const SEVERITY_STYLES = {
  CRITICAL: "danger" as const,
  HIGH: "warning" as const,
  LOW: "neutral" as const,
};

const BAR_HEIGHTS = [252, 315, 189, 378, 273, 336, 231, 168, 294, 357];

export default function AdminDashboardPage() {
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

      {/* Tarjetas resumen */}
      <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
        <StatCard
          label="Usuarios Activos Totales"
          value="42,891"
          hint="+12.5% este mes"
          icon={<Users className="size-6" />}
        />
        <StatCard
          label="Entidades de Salud"
          value="160"
          hint="156 activas · 4 inactivas"
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
          value="07"
          hint="Requiere auditoría inmediata"
          valueClassName="text-danger"
          icon={<ShieldAlert className="size-6 text-danger" />}
        />
      </div>

      {/* Bento grid: métricas + nodos */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Gráfico de uso global */}
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

          {/* Gráfico de barras mock */}
          <div className="relative h-64">
            {/* Líneas de cuadrícula */}
            <div className="absolute inset-0 flex flex-col justify-between">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="border-t border-line/40" />
              ))}
            </div>
            {/* Barras */}
            <div className="absolute inset-x-0 bottom-0 flex items-end gap-1 px-4">
              {BAR_HEIGHTS.map((h, i) => (
                <div
                  key={i}
                  className={`flex-1 rounded-t transition-all ${
                    i === 3
                      ? "bg-navy"
                      : "bg-navy/20 hover:bg-navy/40"
                  }`}
                  style={{ height: `${(h / 378) * 100}%` }}
                />
              ))}
            </div>
            {/* Etiqueta pico */}
            <div className="absolute bottom-[100%] left-[37%] -translate-x-1/2 translate-y-2">
              <span className="rounded bg-navy px-2 py-0.5 text-[10px] text-white">
                Pico
              </span>
            </div>
          </div>
          {/* Eje X */}
          <div className="mt-2 flex justify-between px-4 text-[10px] uppercase tracking-[0.6px] text-muted">
            <span>Oct 01</span>
            <span>Oct 10</span>
            <span>Oct 20</span>
            <span>Oct 30</span>
          </div>
        </Card>

        {/* Estado de nodos */}
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

      {/* Tabla de alertas críticas */}
      <Card className="overflow-hidden">
        <div className="flex items-center justify-between border-b border-line p-6">
          <div className="flex items-center gap-2">
            <ShieldAlert className="size-5 text-danger" />
            <h3 className="font-display text-xl font-semibold text-ink">
              Alertas Críticas del Sistema
            </h3>
          </div>
          <button className="text-sm text-teal hover:text-teal-700">
            Limpiar no críticas
          </button>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">Timestamp</th>
              <th className="px-6 py-4 font-normal">Severidad</th>
              <th className="px-6 py-4 font-normal">Tipo de Evento</th>
              <th className="px-6 py-4 font-normal">IP de Origen</th>
              <th className="px-6 py-4 font-normal text-right">Acción</th>
            </tr>
          </thead>
          <tbody>
            {ALERTS.map((alert, i) => (
              <tr key={i} className="border-t border-line">
                <td className="px-6 py-4 font-mono text-xs text-slate">
                  {alert.timestamp}
                </td>
                <td className="px-6 py-4">
                  <Badge tone={SEVERITY_STYLES[alert.severity]}>
                    {alert.severity}
                  </Badge>
                </td>
                <td className="px-6 py-4 text-navy-800">{alert.event}</td>
                <td className="px-6 py-4 font-mono text-xs text-slate">
                  {alert.origin}
                </td>
                <td className="px-6 py-4 text-right">
                  <button className="rounded bg-navy px-3 py-1.5 text-xs text-white transition-colors hover:bg-navy-800">
                    Tomar Acción
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
