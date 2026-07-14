"use client";

import { Fragment, useEffect, useState, useCallback } from "react";
import { ChevronLeft, ChevronRight, Plus, Clock, User } from "lucide-react";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

// Franjas de media hora de 06:00 a 19:30 (mismo rango en que puede agendar el paciente).
const HOURS = Array.from({ length: 28 }, (_, i) => {
  const total = 6 * 60 + i * 30; // 06:00 .. 19:30
  const h = Math.floor(total / 60);
  const m = total % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
});

type CitaAPI = {
  id: string;
  fecha_hora: string; // hora local de Colombia (pared), aunque venga marcada como UTC
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

// El backend guarda y devuelve la fecha/hora de la cita en hora local de
// Colombia (Bogotá), aunque el timestamp venga marcado como UTC ("...Z").
// Por eso NO convertimos de zona: leemos la hora "de pared" tal cual viene
// en el string. Convertir aquí restaría 5 horas y descuadraría la agenda.
function wallParts(iso: string): { ymd: string; hour: number; minute: number } {
  const m = iso.match(/(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/);
  if (m) {
    return {
      ymd: `${m[1]}-${m[2]}-${m[3]}`,
      hour: parseInt(m[4], 10),
      minute: parseInt(m[5], 10),
    };
  }
  // Fallback por si el formato es inesperado.
  const d = new Date(iso);
  return { ymd: toYMD(d), hour: d.getHours(), minute: d.getMinutes() };
}

// Fecha YYYY-MM-DD de un Date usando sus componentes locales.
function toYMD(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${dd}`;
}

// Lunes de la semana que contiene la fecha dada (basado en hora local del browser)
function getLunes(d: Date): Date {
  const day = new Date(d);
  const dow = day.getDay() === 0 ? 7 : day.getDay(); // 1=lun .. 7=dom
  day.setDate(day.getDate() - (dow - 1));
  day.setHours(0, 0, 0, 0);
  return day;
}

// Etiqueta de franja "HH:MM" a la que pertenece una cita, redondeando hacia
// abajo a la media hora (18:00 -> "18:00", 18:15 -> "18:00", 18:45 -> "18:30").
function citaSlot(iso: string): string {
  const { hour, minute } = wallParts(iso);
  const m = minute < 30 ? 0 : 30;
  return `${String(hour).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

function citaDay(iso: string): string {
  return wallParts(iso).ymd;
}

function nombreCompleto(c: CitaAPI) {
  return `${c.paciente.nombre_paciente} ${c.paciente.apellidos_paciente}`;
}

function formatHora(iso: string) {
  const { hour, minute } = wallParts(iso);
  return `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
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
      const fecha = toYMD(lunes);
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
    (async () => {
      await fetchCitas(refDate);
    })();
  }, [refDate, fetchCitas]);

  // Días de la semana visible (lunes a domingo, hora local)
  const weekDays: Date[] = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(refDate);
    d.setDate(d.getDate() + i);
    return d;
  });

  const domingo = new Date(refDate);
  domingo.setDate(domingo.getDate() + 6);

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
    return citasDelDia(day).find((c) => citaSlot(c.fecha_hora) === hourLabel);
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
          <div className="grid grid-cols-[64px_repeat(7,1fr)] border-b border-line bg-[#e6f2fa]">
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
                    {["Lun","Mar","Mié","Jue","Vie","Sáb","Dom"][i]}
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
          <div className="grid grid-cols-[64px_repeat(7,1fr)] overflow-auto" style={{ maxHeight: "540px" }}>
            {HOURS.map((h) => (
              <Fragment key={h}>
                <div className="border-b border-r border-line px-2 py-3 text-right text-xs text-muted">
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
              </Fragment>
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
                    {["Lun","Mar","Mié","Jue","Vie","Sáb","Dom"][i]}
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