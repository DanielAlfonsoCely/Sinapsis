"use client";

import { useState, useEffect, useRef, useCallback } from "react";

import {
  ShieldAlert,
  Search,
  Filter,
  ChevronLeft,
  ChevronRight,
  Eye,
  Download,
  RefreshCw,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { fetchAuditoria, exportToCSV, TipoOperacion, AuditLogEntry } from "./api";

// ---------------------------------------------------------------------------
// Tipos
// ---------------------------------------------------------------------------
type RegistroAuditoria = {
  id: string;
  usuario: string;
  usuarioEmail: string;
  tipo_operacion: TipoOperacion;
  tabla_afectada: string;
  registro_id: string | null;
  detalles: string | null;
  ip_origen: string | null;
  fecha_operacion: string;
  gravedad: string;
};

const LIMIT = 50;

// ---------------------------------------------------------------------------
// Helpers visuales
// ---------------------------------------------------------------------------
const OPERACION_STYLES: Record<
  TipoOperacion,
  { tone: "success" | "info" | "danger" | "warning" | "neutral"; label: string }
> = {
  crear:            { tone: "success", label: "Crear" },
  actualizar:       { tone: "info",    label: "Actualizar" },
  eliminar:         { tone: "danger",  label: "Eliminar" },
  consultar:        { tone: "neutral", label: "Consultar" },
  exportar:         { tone: "warning", label: "Exportar" },
  cambiar_permisos: { tone: "warning", label: "Cambiar Permisos" },
  usar_ia:          { tone: "info",    label: "Usar IA" },
};



const TABLAS_LEGIBLES: Record<string, string> = {
  historia_clinica: "Historia Clínica",
  consulta:         "Consulta",
  paciente:         "Paciente",
  usuario:          "Usuario",
  entidad:          "Entidad",
  cita:             "Cita",
};

function formatFecha(iso: string) {
  return new Date(iso + (iso.endsWith("Z") ? "" : "Z")).toLocaleString("es-CO", {
    timeZone: "America/Bogota",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function initials(nombre: string) {
  return nombre
    .split(" ")
    .filter((_, i) => i < 2)
    .map((w) => w[0])
    .join("")
    .toUpperCase();
}
// ---------------------------------------------------------------------------
// Componente principal
// ---------------------------------------------------------------------------
export default function RegistrosSistemaPage() {
  const [registros, setRegistros] = useState<RegistroAuditoria[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);

  const [search, setSearch] = useState("");
  const [filtroOp, setFiltroOp] = useState<string>("");
  const [filtroTabla, setFiltroTabla] = useState<string>("");
  const [detalle, setDetalle] = useState<RegistroAuditoria | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cargarRegistros = useCallback(async (page: number) => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchAuditoria(LIMIT, (page - 1) * LIMIT);

      const mapped: RegistroAuditoria[] = (data.registros ?? []).map(
        (e: AuditLogEntry) => ({
          id: e.id,
          usuario: e.usuario_nombre,
          usuarioEmail: e.usuario_email,
          tipo_operacion: e.tipo_operacion,
          tabla_afectada: e.tabla_afectada,
          registro_id: e.registro_id,
          detalles: e.detalles,
          ip_origen: e.ip_origen,
          fecha_operacion: e.fecha_operacion,
          gravedad: e.gravedad ?? "",
        })
      );

      setRegistros(mapped);
      setTotal(data.total ?? 0);
    } catch {
      setError("No se pudo cargar la bitácora de auditoría.");
    } finally {
      setLoading(false);
    }
  }, []);

  const handleExport = () => {
  const rows = filtered.map((r) => ({
    "Fecha y hora": formatFecha(r.fecha_operacion),
    "Usuario": r.usuario,
    "Email": r.usuarioEmail,
    "Operación": OPERACION_STYLES[r.tipo_operacion]?.label ?? r.tipo_operacion,
    "Tabla afectada": TABLAS_LEGIBLES[r.tabla_afectada] ?? r.tabla_afectada,
    "Detalles": r.detalles ?? "",
    "IP origen": r.ip_origen ?? "",
    "Gravedad": r.gravedad,
  }));

  const fecha = new Date().toISOString().slice(0, 10);
  exportToCSV(rows, `auditoria_sinapsis_${fecha}.csv`);
};

  useEffect(() => {
    void cargarRegistros(currentPage);
  }, [currentPage, cargarRegistros]);

  // Filtrado client-side sobre los registros ya cargados
  const filtered = registros.filter((r) => {
    const matchSearch =
      !search ||
      r.usuario.toLowerCase().includes(search.toLowerCase()) ||
      r.usuarioEmail.toLowerCase().includes(search.toLowerCase()) ||
      (r.detalles ?? "").toLowerCase().includes(search.toLowerCase()) ||
      (r.registro_id ?? "").toLowerCase().includes(search.toLowerCase());
    const matchOp = !filtroOp || r.tipo_operacion === filtroOp;
    const matchTabla = !filtroTabla || r.tabla_afectada === filtroTabla;
    return matchSearch && matchOp && matchTabla;
  });

  // Stats calculadas sobre los registros cargados
  const statsModHC = registros.filter(
    (r) => r.tabla_afectada === "historia_clinica" &&
      (r.tipo_operacion === "crear" || r.tipo_operacion === "actualizar"),
  ).length;
  const statsExport = registros.filter((r) => r.tipo_operacion === "exportar").length;
  const statsPermisos = registros.filter((r) => r.tipo_operacion === "cambiar_permisos").length;

  const totalPages = Math.ceil(total / LIMIT);

  const selectClass =
    "rounded border border-line bg-field px-3 py-2 text-sm text-slate focus:outline-none focus:ring-1 focus:ring-teal";

  return (
    <div className="flex flex-col gap-6">
      {/* Modal detalle */}
      {detalle && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-lg flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">
                Detalle del registro
              </h3>
              <button
                onClick={() => setDetalle(null)}
                className="text-muted hover:text-ink"
              >
                ✕
              </button>
            </div>

            <div className="flex flex-col gap-3 rounded border border-line bg-shell p-4 text-sm">
              <Row label="ID Registro" value={detalle.id} mono />
              <Row label="Usuario" value={`${detalle.usuario} (${detalle.usuarioEmail})`} />
              <Row
                label="Operación"
                value={OPERACION_STYLES[detalle.tipo_operacion]?.label ?? detalle.tipo_operacion}
              />
              <Row
                label="Tabla afectada"
                value={TABLAS_LEGIBLES[detalle.tabla_afectada] ?? detalle.tabla_afectada}
              />
              {detalle.registro_id && (
                <Row label="ID del registro afectado" value={detalle.registro_id} mono />
              )}
              {detalle.ip_origen && (
                <Row label="IP de origen" value={detalle.ip_origen} mono />
              )}
              <Row label="Fecha y hora" value={formatFecha(detalle.fecha_operacion)} />
              {detalle.detalles && (
                <div>
                  <p className="text-[10px] uppercase tracking-[0.6px] text-muted">
                    Detalles
                  </p>
                  <p className="mt-0.5 text-navy-800">{detalle.detalles}</p>
                </div>
              )}
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
      <div className="flex items-start justify-between border-b border-line pb-5">
        <div className="flex items-center gap-3">
          <ShieldAlert className="size-5 text-teal" />
          <div>
            <h2 className="font-display text-2xl font-semibold text-ink">
              Registros del Sistema
            </h2>
            <p className="text-sm text-slate">
              Bitácora inmutable de auditoría — HU-08
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleExport}
            disabled={filtered.length === 0}
            className="flex items-center gap-2 rounded border border-line px-4 py-2.5 text-sm text-slate hover:bg-field disabled:cursor-not-allowed disabled:opacity-40"
          >
            <Download className="size-4" />
            Exportar
          </button>
          <button
            onClick={() => void cargarRegistros(currentPage)}
            className="flex items-center gap-2 rounded border border-line px-4 py-2.5 text-sm text-slate hover:bg-field"
          >
            <RefreshCw className="size-4" />
            Actualizar
          </button>
        </div>
      </div>

      {/* Stats rápidas */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {[
          { label: "Total registros",     value: total.toString() },
          { label: "Modificaciones HC",   value: statsModHC.toString() },
          { label: "Exportaciones",        value: statsExport.toString() },
          { label: "Cambios de permisos", value: statsPermisos.toString() },
        ].map(({ label, value }) => (
          <Card key={label} className="p-4">
            <p className="text-[10px] uppercase tracking-[0.6px] text-muted">{label}</p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">{value}</p>
          </Card>
        ))}
      </div>

      {/* Filtros */}
      <Card className="p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2 text-sm text-slate">
            <Filter className="size-4" />
            <span className="font-medium">Filtrar:</span>
          </div>

          <div className="relative flex-1 min-w-48">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted" />
            <input
              type="text"
              placeholder="Buscar usuario, detalle o ID…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-9 w-full rounded border border-line bg-field pl-9 pr-4 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
            />
          </div>

          <select
            className={selectClass}
            value={filtroOp}
            onChange={(e) => setFiltroOp(e.target.value)}
          >
            <option value="">Todas las operaciones</option>
            <option value="crear">Crear</option>
            <option value="actualizar">Actualizar</option>
            <option value="eliminar">Eliminar</option>
            <option value="consultar">Consultar</option>
            <option value="exportar">Exportar</option>
            <option value="cambiar_permisos">Cambiar Permisos</option>
            <option value="usar_ia">Usar IA</option>
          </select>

          <select
            className={selectClass}
            value={filtroTabla}
            onChange={(e) => setFiltroTabla(e.target.value)}
          >
            <option value="">Todas las tablas</option>
            <option value="historia_clinica">Historia Clínica</option>
            <option value="consulta">Consulta</option>
            <option value="paciente">Paciente</option>
            <option value="usuario">Usuario</option>
            <option value="entidad">Entidad</option>
            <option value="cita">Cita</option>
          </select>
        </div>
      </Card>

      {/* Banner error */}
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
              <th className="px-6 py-4 font-normal">Fecha y hora</th>
              <th className="px-6 py-4 font-normal">Usuario</th>
              <th className="px-6 py-4 font-normal">Operación</th>
              <th className="px-6 py-4 font-normal">Tabla afectada</th>
              <th className="px-6 py-4 font-normal">Detalles</th>
              <th className="px-6 py-4 font-normal">IP origen</th>
              <th className="px-6 py-4 font-normal text-center">Ver</th>
            </tr>
          </thead>
          <tbody>
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={7} className="px-6 py-8 text-center text-slate">
                  No se encontraron registros.
                </td>
              </tr>
            )}
            {filtered.map((r) => {
              const op = OPERACION_STYLES[r.tipo_operacion] ?? { tone: "neutral" as const, label: r.tipo_operacion };
              return (
                <tr key={r.id} className="border-t border-line hover:bg-shell">
                  <td className="px-6 py-4 font-mono text-xs text-slate whitespace-nowrap">
                    {formatFecha(r.fecha_operacion)}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <span className="flex size-7 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-[10px] font-medium text-teal">
                        {initials(r.usuario)}
                      </span>
                      <div>
                        <p className="text-xs font-medium text-navy-800">{r.usuario}</p>
                        <p className="text-[10px] text-muted">{r.usuarioEmail}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <Badge tone={op.tone}>{op.label}</Badge>
                  </td>
                  <td className="px-6 py-4 text-slate">
                    {TABLAS_LEGIBLES[r.tabla_afectada] ?? r.tabla_afectada}
                  </td>
                  <td className="px-6 py-4 max-w-xs">
                    <p className="truncate text-xs text-slate" title={r.detalles ?? ""}>
                      {r.detalles ?? "—"}
                    </p>
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-muted">
                    {r.ip_origen ?? "—"}
                  </td>
                  <td className="px-6 py-4 text-center">
                    <button
                      onClick={() => setDetalle(r)}
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

        <div className="flex items-center justify-between border-t border-line px-6 py-4">
          <p className="text-sm text-muted">
            {total === 0
              ? "No hay registros para mostrar"
              : `Mostrando ${filtered.length} de ${total} registros`}
          </p>
          <div className="flex items-center gap-1">
            <button
              disabled={currentPage === 1}
              onClick={() => setCurrentPage((p) => p - 1)}
              className="flex size-8 items-center justify-center rounded border border-line text-slate hover:bg-field disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronLeft className="size-4" />
            </button>
            {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
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
              className="flex size-8 items-center justify-center rounded border border-line text-slate hover:bg-field disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronRight className="size-4" />
            </button>
          </div>
        </div>
      </Card>

      <p className="text-xs text-muted text-center">
        La bitácora de auditoría es inmutable. Ningún rol puede modificar ni eliminar estos registros (RN-010).
      </p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helper de fila para el modal
// ---------------------------------------------------------------------------
function Row({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div>
      <p className="text-[10px] uppercase tracking-[0.6px] text-muted">{label}</p>
      <p className={`mt-0.5 text-navy-800 ${mono ? "font-mono text-xs" : "text-sm"}`}>
        {value}
      </p>
    </div>
  );
}