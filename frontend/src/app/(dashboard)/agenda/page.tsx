"use client";

import { useEffect, useState, useCallback } from "react";
import { ChevronLeft, ChevronRight, Plus, Clock, User } from "lucide-react";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const HOURS = [
  "07:00","08:00","09:00","10:00","11:00","12:00",
  "13:00","14:00","15:00","16:00","17:00","18:00",
];

type CitaAPI = {
  id: string;
  fecha_hora: string; // UTC sin Z
  estado: string;
  motivo: string | null;
  paciente: {
    id: string;
    nombre_paciente: string;
    apellidos_paciente: string;
    numero_documento: string;
  };
};

const estadoTone: Record<string, string> = {
  programada: "bg-teal/10 border-teal/30 text-teal-700",
  en_curso:   "bg-warning/10 border-warning/30 text-warning",
  completada: "bg-success/10 border-success/30 text-success",
  cancelada:  "bg-line border-line text-muted",
  no_asistio: "bg-danger/10 border-danger/30 text-danger",
};

// Convierte string UTC (sin Z) a Date tratándolo como UTC
function toDate(iso: string): Date {
  const s = iso.endsWith("Z") || iso.includes("+") ? iso : iso + "Z";
  return new Date(s);
}

// Fecha YYYY-MM-DD en zona Bogotá para un Date
function toYMD(d: Date): string {
  return d.toLocaleDateString("en-CA", { timeZone: "America/Bogota" }); // en-CA = YYYY-MM-DD
}

// Lunes de la semana que contiene la fecha dada (basado en hora local del browser)
function getLunes(d: Date): Date {
  const day = new Date(d);
  const dow = day.getDay() === 0 ? 7 : day.getDay(); // 1=lun .. 7=dom
  day.setDate(day.getDate() - (dow - 1));
  day.setHours(0, 0, 0, 0);
  return day;
}

// YYYY-MM-DD para enviar al backend (usa hora local)
function toYMDLocal(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${dd}`;
}

function citaHour(iso: string): number {
  // Hora en Bogotá como número
  const s = toDate(iso).toLocaleString("en-US", { hour: "numeric", hour12: false, timeZone: "America/Bogota" });
  return parseInt(s, 10);
}

function citaDay(iso: string): string {
  return toYMD(toDate(iso)); // YYYY-MM-DD en Bogotá
}

function nombreCompleto(c: CitaAPI) {
  return `${c.paciente.nombre_paciente} ${c.paciente.apellidos_paciente}`;
}

function formatHora(iso: string) {
  return toDate(iso).toLocaleTimeString("es-CO", {
    hour: "2-digit", minute: "2-digit", timeZone: "America/Bogota",
  });
}

export default function AgendaPage() {
  const [view, setView] = useState<"semana" | "dia">("semana");
  const [refDate, setRefDate] = useState(() => {
    const now = new Date();
    return getLunes(now);
  });
  const [selectedDay, setSelectedDay] = useState<Date>(() => {
    const now = new Date();
    now.setUTCHours(0, 0, 0, 0);
    return now;
  });
  const [citas, setCitas] = useState<CitaAPI[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchCitas = useCallback(async (lunes: Date) => {
    setLoading(true);
    try {
      const token = localStorage.getItem("token");
      const fecha = toYMDLocal(lunes);
      const res = await fetch(
        `http://localhost:8080/api/v1/citas/semana?fecha=${fecha}`,
        { headers: token ? { Authorization: `Bearer ${token}` } : undefined }
      );
      if (res.ok) {
        const data = await res.json();
        setCitas(data.citas ?? []);
      }
    } catch {
      setCitas([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCitas(refDate);
  }, [refDate, fetchCitas]);

  // Días de la semana visible (lunes a viernes, hora local)
  const weekDays: Date[] = Array.from({ length: 5 }, (_, i) => {
    const d = new Date(refDate);
    d.setDate(d.getDate() + i);
    return d;
  });

  const domingo = new Date(refDate);
  domingo.setDate(domingo.getDate() + 4);

  // Nombre mes/año para cabecera
  const semanaLabel = `${refDate.toLocaleDateString("es-CO", { day: "numeric", month: "long" })} – ${domingo.toLocaleDateString("es-CO", { day: "numeric", month: "long", year: "numeric" })}`;

  function prevSemana() {
    const d = new Date(refDate);
    d.setDate(d.getDate() - 7);
    setRefDate(d);
  }
  function nextSemana() {
    const d = new Date(refDate);
    d.setDate(d.getDate() + 7);
    setRefDate(d);
  }

  function citasDelDia(day: Date) {
    const ymd = toYMD(day);
    return citas.filter((c) => citaDay(c.fecha_hora) === ymd);
  }

  function citaEnHora(day: Date, hourLabel: string): CitaAPI | undefined {
    const h = parseInt(hourLabel.split(":")[0], 10);
    return citasDelDia(day).find((c) => citaHour(c.fecha_hora) === h);
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink">Agenda</h2>
          <p className="text-sm text-slate">{semanaLabel}</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex rounded-[var(--radius)] border border-line bg-field p-1">
            {(["semana", "dia"] as const).map((v) => (
              <button
                key={v}
                onClick={() => setView(v)}
                className={cn(
                  "rounded px-3 py-1 text-sm capitalize transition-colors",
                  view === v
                    ? "bg-white font-medium text-ink shadow-sm"
                    : "text-slate hover:text-ink"
                )}
              >
                {v === "semana" ? "Semana" : "Día"}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Navegación semana */}
      <div className="flex items-center gap-4">
        <button
          onClick={prevSemana}
          className="flex size-8 items-center justify-center rounded border border-line hover:bg-field"
        >
          <ChevronLeft className="size-4 text-slate" />
        </button>
        <span className="text-sm font-medium text-ink">
          Semana del {semanaLabel}
        </span>
        <button
          onClick={nextSemana}
          className="flex size-8 items-center justify-center rounded border border-line hover:bg-field"
        >
          <ChevronRight className="size-4 text-slate" />
        </button>
      </div>

      {loading && (
        <p className="text-sm text-slate">Cargando citas…</p>
      )}

      {!loading && view === "semana" && (
        <Card className="overflow-hidden">
          {/* Cabecera días */}
          <div className="grid grid-cols-[64px_repeat(5,1fr)] border-b border-line bg-[#e6f2fa]">
            <div className="border-r border-line" />
            {weekDays.map((d, i) => {
              const ymd = toYMD(d);
              const hoy = toYMD(new Date());
              return (
                <button
                  key={ymd}
                  onClick={() => { setSelectedDay(d); setView("dia"); }}
                  className={cn(
                    "flex flex-col items-center py-3 text-xs transition-colors hover:bg-[#d2e8f3]",
                    ymd === hoy && "bg-teal/10"
                  )}
                >
                  <span className="uppercase tracking-[0.6px] text-label">
                    {["Lun","Mar","Mié","Jue","Vie"][i]}
                  </span>
                  <span className="mt-0.5 font-display text-lg font-semibold text-ink">
                    {d.getDate()}
                  </span>
                  <span className="text-label">
                    {d.toLocaleDateString("es-CO", { month: "short" })}
                  </span>
                </button>
              );
            })}
          </div>

          {/* Grilla horas */}
          <div className="grid grid-cols-[64px_repeat(5,1fr)] overflow-auto" style={{ maxHeight: "540px" }}>
            {HOURS.map((h) => (
              <>
                <div
                  key={`h-${h}`}
                  className="border-b border-r border-line px-2 py-3 text-right text-xs text-muted"
                >
                  {h}
                </div>
                {weekDays.map((d) => {
                  const appt = citaEnHora(d, h);
                  return (
                    <div
                      key={`${toYMD(d)}-${h}`}
                      className="relative border-b border-r border-line p-1"
                    >
                      {appt && (
                        <div
                          className={cn(
                            "rounded border p-1.5 text-xs",
                            estadoTone[appt.estado] ?? estadoTone.programada
                          )}
                        >
                          <p className="font-medium leading-snug">{nombreCompleto(appt)}</p>
                          <p className="opacity-75">{formatHora(appt.fecha_hora)}</p>
                        </div>
                      )}
                    </div>
                  );
                })}
              </>
            ))}
          </div>
        </Card>
      )}

      {!loading && view === "dia" && (
        <div className="flex flex-col gap-4">
          {/* Selector de día */}
          <div className="flex gap-2">
            {weekDays.map((d, i) => {
              const ymd = toYMD(d);
              const selYmd = toYMD(selectedDay);
              return (
                <button
                  key={ymd}
                  onClick={() => setSelectedDay(d)}
                  className={cn(
                    "flex flex-col items-center rounded-[var(--radius)] border px-4 py-2 text-xs transition-colors",
                    ymd === selYmd
                      ? "border-teal bg-teal/10 text-teal"
                      : "border-line bg-white text-slate hover:bg-field"
                  )}
                >
                  <span className="uppercase tracking-[0.6px]">
                    {["Lun","Mar","Mié","Jue","Vie"][i]}
                  </span>
                  <span className="font-display text-lg font-semibold">{d.getDate()}</span>
                </button>
              );
            })}
          </div>

          <Card className="divide-y divide-line">
            {HOURS.map((h) => {
              const appt = citaEnHora(selectedDay, h);
              return (
                <div key={h} className="flex items-start gap-4 px-6 py-4">
                  <span className="w-12 shrink-0 text-sm text-muted">{h}</span>
                  {appt ? (
                    <div
                      className={cn(
                        "flex flex-1 items-center gap-4 rounded-[var(--radius)] border p-3",
                        estadoTone[appt.estado] ?? estadoTone.programada
                      )}
                    >
                      <div className="flex size-8 items-center justify-center rounded-full bg-white/60">
                        <User className="size-4" />
                      </div>
                      <div className="flex-1">
                        <p className="font-medium">{nombreCompleto(appt)}</p>
                        <p className="text-xs opacity-75">
                          {appt.motivo ?? appt.estado}
                        </p>
                      </div>
                      <div className="flex items-center gap-1 text-xs opacity-75">
                        <Clock className="size-3" />
                        {formatHora(appt.fecha_hora)}
                      </div>
                    </div>
                  ) : (
                    <div className="h-10 flex-1 rounded-[var(--radius)] border border-dashed border-line" />
                  )}
                </div>
              );
            })}
          </Card>
        </div>
      )}
    </div>
  );
}