"use client";

import { useState } from "react";
import { ChevronLeft, ChevronRight, Plus, Clock, User } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const DAYS = [
  { label: "Lun", date: 26 },
  { label: "Mar", date: 27 },
  { label: "Mié", date: 28 },
  { label: "Jue", date: 29 },
  { label: "Vie", date: 30 },
];

const HOURS = [
  "08:00", "09:00", "10:00", "11:00", "12:00",
  "13:00", "14:00", "15:00", "16:00", "17:00",
];

type Appt = {
  day: number;
  hour: number;
  duration: number;
  patient: string;
  type: string;
  tone: "info" | "success" | "warning" | "navy";
};

const APPOINTMENTS: Appt[] = [
  { day: 26, hour: 0, duration: 1, patient: "Ricardo Mendoza", type: "Control", tone: "info" },
  { day: 26, hour: 2, duration: 1, patient: "Sofía Palacio", type: "Primera Vez", tone: "navy" },
  { day: 27, hour: 1, duration: 1, patient: "Gabriel Caicedo", type: "Seguimiento", tone: "success" },
  { day: 27, hour: 4, duration: 1, patient: "Lorena Vargas", type: "Urgencia", tone: "warning" },
  { day: 28, hour: 0, duration: 1, patient: "Andrés Ospina", type: "Control", tone: "info" },
  { day: 28, hour: 3, duration: 1, patient: "Carolina Mora", type: "Seguimiento", tone: "success" },
  { day: 29, hour: 2, duration: 1, patient: "Ricardo Mendoza", type: "Resultado", tone: "navy" },
  { day: 30, hour: 1, duration: 1, patient: "María Torres", type: "Primera Vez", tone: "navy" },
  { day: 30, hour: 5, duration: 1, patient: "José Ramírez", type: "Control", tone: "info" },
];

const toneStyles: Record<Appt["tone"], string> = {
  info: "bg-teal/10 border-teal/30 text-teal-700",
  success: "bg-success/10 border-success/30 text-success",
  warning: "bg-warning/10 border-warning/30 text-warning",
  navy: "bg-navy/10 border-navy/30 text-navy",
};

export default function AgendaPage() {
  const [view, setView] = useState<"semana" | "dia">("semana");
  const [selectedDay, setSelectedDay] = useState(0);

  const dayAppts = APPOINTMENTS.filter((a) => a.day === DAYS[selectedDay].date);

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink">Agenda</h2>
          <p className="text-sm text-slate">Mayo 26 – 30, 2026</p>
        </div>
        <div className="flex items-center gap-3">
          {/* Toggle semana/día */}
          <div className="flex rounded-[var(--radius)] border border-line bg-field p-1">
            {(["semana", "dia"] as const).map((v) => (
              <button
                key={v}
                onClick={() => setView(v)}
                className={cn(
                  "rounded px-3 py-1 text-sm capitalize transition-colors",
                  view === v
                    ? "bg-white font-medium text-ink shadow-sm"
                    : "text-slate hover:text-ink",
                )}
              >
                {v === "semana" ? "Semana" : "Día"}
              </button>
            ))}
          </div>
          <button className="flex items-center gap-1.5 rounded-[var(--radius)] bg-navy px-4 py-2 text-sm font-medium text-white hover:bg-navy-800">
            <Plus className="size-4" />
            Nueva cita
          </button>
        </div>
      </div>

      {/* Navegación de semana */}
      <div className="flex items-center gap-4">
        <button className="flex size-8 items-center justify-center rounded border border-line hover:bg-field">
          <ChevronLeft className="size-4 text-slate" />
        </button>
        <span className="text-sm font-medium text-ink">Semana del 26 de Mayo al 30 de Mayo</span>
        <button className="flex size-8 items-center justify-center rounded border border-line hover:bg-field">
          <ChevronRight className="size-4 text-slate" />
        </button>
      </div>

      {view === "semana" ? (
        /* Vista semanal */
        <Card className="overflow-hidden">
          {/* Cabecera días */}
          <div className="grid grid-cols-[64px_repeat(5,1fr)] border-b border-line bg-[#e6f2fa]">
            <div className="border-r border-line" />
            {DAYS.map((d, i) => (
              <button
                key={d.date}
                onClick={() => { setSelectedDay(i); setView("dia"); }}
                className="flex flex-col items-center py-3 text-xs transition-colors hover:bg-[#d2e8f3]"
              >
                <span className="uppercase tracking-[0.6px] text-label">{d.label}</span>
                <span className="mt-0.5 font-display text-lg font-semibold text-ink">
                  {d.date}
                </span>
                <span className="text-label">Mayo</span>
              </button>
            ))}
          </div>

          {/* Grilla de horas */}
          <div className="grid grid-cols-[64px_repeat(5,1fr)] overflow-auto" style={{ maxHeight: "540px" }}>
            {HOURS.map((h, hi) => (
              <>
                <div
                  key={`h-${h}`}
                  className="border-b border-r border-line px-2 py-3 text-right text-xs text-muted"
                >
                  {h}
                </div>
                {DAYS.map((d) => {
                  const appt = APPOINTMENTS.find(
                    (a) => a.day === d.date && a.hour === hi,
                  );
                  return (
                    <div
                      key={`${d.date}-${h}`}
                      className="relative border-b border-r border-line p-1"
                    >
                      {appt && (
                        <div
                          className={cn(
                            "rounded border p-1.5 text-xs",
                            toneStyles[appt.tone],
                          )}
                        >
                          <p className="font-medium leading-snug">{appt.patient}</p>
                          <p className="opacity-75">{appt.type}</p>
                        </div>
                      )}
                    </div>
                  );
                })}
              </>
            ))}
          </div>
        </Card>
      ) : (
        /* Vista diaria */
        <div className="flex flex-col gap-4">
          {/* Selector de día */}
          <div className="flex gap-2">
            {DAYS.map((d, i) => (
              <button
                key={d.date}
                onClick={() => setSelectedDay(i)}
                className={cn(
                  "flex flex-col items-center rounded-[var(--radius)] border px-4 py-2 text-xs transition-colors",
                  selectedDay === i
                    ? "border-teal bg-teal/10 text-teal"
                    : "border-line bg-white text-slate hover:bg-field",
                )}
              >
                <span className="uppercase tracking-[0.6px]">{d.label}</span>
                <span className="font-display text-lg font-semibold">{d.date}</span>
              </button>
            ))}
          </div>

          <Card className="divide-y divide-line">
            {HOURS.map((h, hi) => {
              const appt = dayAppts.find((a) => a.hour === hi);
              return (
                <div key={h} className="flex items-start gap-4 px-6 py-4">
                  <span className="w-12 shrink-0 text-sm text-muted">{h}</span>
                  {appt ? (
                    <div
                      className={cn(
                        "flex flex-1 items-center gap-4 rounded-[var(--radius)] border p-3",
                        toneStyles[appt.tone],
                      )}
                    >
                      <div className="flex size-8 items-center justify-center rounded-full bg-white/60">
                        <User className="size-4" />
                      </div>
                      <div className="flex-1">
                        <p className="font-medium">{appt.patient}</p>
                        <p className="text-xs opacity-75">{appt.type}</p>
                      </div>
                      <div className="flex items-center gap-1 text-xs opacity-75">
                        <Clock className="size-3" />
                        {h}
                      </div>
                      <Badge tone={appt.tone}>{appt.type}</Badge>
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
