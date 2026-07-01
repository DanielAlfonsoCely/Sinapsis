"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ChevronLeft,
  Stethoscope,
  BrainCircuit,
  Pill,
  FileText,
  ChevronDown,
  ChevronUp,
  Droplet,
  AlertCircle,
  User,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const PATIENT = {
  initials: "RM",
  name: "Ricardo Mendoza",
  id: "1.024.556.782",
  dob: "12 Mar 1984",
  age: 42,
  blood: "O+",
  eps: "Nueva EPS",
  allergies: ["Penicilina", "Sulfas"],
};

type EntryType = "Consulta" | "Análisis IA" | "Fórmula" | "Nota";

const ENTRIES: Array<{
  id: string;
  date: string;
  type: EntryType;
  doctor: string;
  summary: string;
  detail: string;
  tone: "info" | "navy" | "success" | "neutral";
}> = [
  {
    id: "E-041",
    date: "1 Jul 2026",
    type: "Consulta",
    doctor: "Dr. Camilo Pineda",
    summary: "Infección respiratoria — neumonía adquirida en comunidad (J18.9)",
    detail:
      "Paciente acude con fiebre de 38.5°C, tos productiva y dolor pleurítico. Se solicitó RX de tórax. Análisis IA sugirió opacidad en LID con 87% de confianza. Diagnóstico final: NAC. Tratamiento: Amoxicilina 500 mg c/8h × 7d + Ibuprofeno 400 mg c/6h × 5d. Reposo relativo 5 días.",
    tone: "info",
  },
  {
    id: "E-040",
    date: "1 Jul 2026",
    type: "Análisis IA",
    doctor: "Dr. Camilo Pineda",
    summary: "RX tórax PA — opacidad LID (87%), patrón intersticial bilateral (61%)",
    detail:
      "Análisis MONAI v2.3 procesado en 4.2 s. Hallazgos: opacidad en LID (87%), patrón intersticial bilateral leve (61%), sin derrame pleural (94%), silueta cardíaca normal (96%). Confianza general: 82%. Sugerencia diagnóstica: NAC J18.9.",
    tone: "navy",
  },
  {
    id: "E-038",
    date: "1 Jul 2026",
    type: "Fórmula",
    doctor: "Dr. Camilo Pineda",
    summary: "Amoxicilina 500 mg · Ibuprofeno 400 mg — emitidas",
    detail:
      "F-2026-0041: Amoxicilina 500 mg — 1 cáp c/8h × 7d. F-2026-0038: Ibuprofeno 400 mg — 1 tab c/6h × 5d (con comida). Ambas fórmulas en estado Vigente.",
    tone: "success",
  },
  {
    id: "E-022",
    date: "15 May 2026",
    type: "Consulta",
    doctor: "Dr. Camilo Pineda",
    summary: "Rinitis alérgica estacional (J30.1) — control",
    detail:
      "Paciente con rinorrea hialina, estornudos y prurito nasal. Se ajusta tratamiento: Loratadina 10 mg c/24h × 30d. Paciente refiere mejoría respecto a visita anterior.",
    tone: "info",
  },
  {
    id: "E-015",
    date: "18 Mar 2026",
    type: "Consulta",
    doctor: "Dra. Valentina Ríos",
    summary: "Diabetes mellitus tipo 2 (E11) — primera vez",
    detail:
      "Glucemia en ayunas: 142 mg/dL. HbA1c: 7.4%. IMC: 27.8. Se inicia manejo con Metformina 850 mg c/12h. Indicaciones nutricionales y actividad física. Control en 2 meses.",
    tone: "info",
  },
  {
    id: "E-009",
    date: "10 Feb 2026",
    type: "Nota",
    doctor: "Dr. Camilo Pineda",
    summary: "Perfil lipídico alterado — se agrega Atorvastatina 20 mg",
    detail:
      "LDL: 158 mg/dL. HDL: 42 mg/dL. Triglicéridos: 210 mg/dL. Se agrega Atorvastatina 20 mg c/24h (noche). Control lipídico en 3 meses.",
    tone: "neutral",
  },
];

const FILTERS: Array<EntryType | "Todos"> = [
  "Todos", "Consulta", "Análisis IA", "Fórmula", "Nota",
];

const TYPE_ICON: Record<EntryType, React.ElementType> = {
  Consulta: Stethoscope,
  "Análisis IA": BrainCircuit,
  Fórmula: Pill,
  Nota: FileText,
};

const TYPE_TONE: Record<EntryType, string> = {
  Consulta: "bg-teal/10 text-teal border-teal/20",
  "Análisis IA": "bg-navy/10 text-navy border-navy/20",
  Fórmula: "bg-success/10 text-success border-success/20",
  Nota: "bg-field text-slate border-line",
};

export default function HistoriaClinicaPage() {
  const [filter, setFilter] = useState<EntryType | "Todos">("Todos");
  const [expanded, setExpanded] = useState<string | null>("E-041");

  const visible =
    filter === "Todos" ? ENTRIES : ENTRIES.filter((e) => e.type === filter);

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/pacientes">
              <ChevronLeft className="size-4" />
              Pacientes
            </Link>
          </Button>
          <div className="h-5 w-px bg-line" />
          <div>
            <h2 className="font-display text-2xl font-semibold text-ink">
              Historia Clínica
            </h2>
            <p className="text-sm text-slate">
              Registro unificado — inmutable
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href="/consulta">
              <Stethoscope className="size-4" />
              Nueva consulta
            </Link>
          </Button>
        </div>
      </div>

      {/* Banner paciente */}
      <Card className="flex flex-col gap-4 p-5 sm:flex-row sm:items-start">
        <span className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 font-display text-xl font-bold text-teal">
          {PATIENT.initials}
        </span>
        <div className="flex-1 grid grid-cols-2 gap-x-8 gap-y-2 sm:grid-cols-4">
          <div>
            <p className="text-xs uppercase tracking-[0.6px] text-label">Nombre</p>
            <p className="font-semibold text-ink">{PATIENT.name}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.6px] text-label">Cédula</p>
            <p className="font-mono text-navy-800">{PATIENT.id}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.6px] text-label">
              <User className="mr-1 inline size-3" />Edad / Fecha nacimiento
            </p>
            <p className="text-navy-800">{PATIENT.age} años · {PATIENT.dob}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.6px] text-label">EPS</p>
            <p className="text-navy-800">{PATIENT.eps}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.6px] text-label">
              <Droplet className="mr-1 inline size-3" />Grupo sanguíneo
            </p>
            <Badge tone="danger">{PATIENT.blood}</Badge>
          </div>
          <div className="col-span-2">
            <p className="text-xs uppercase tracking-[0.6px] text-label">
              <AlertCircle className="mr-1 inline size-3" />Alergias
            </p>
            <div className="flex flex-wrap gap-1 mt-0.5">
              {PATIENT.allergies.map((a) => (
                <Badge key={a} tone="danger">{a}</Badge>
              ))}
            </div>
          </div>
        </div>
      </Card>

      {/* Filtros */}
      <div className="flex gap-2 flex-wrap">
        {FILTERS.map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={cn(
              "rounded-full border px-3.5 py-1 text-xs font-medium transition-colors",
              filter === f
                ? "border-teal bg-teal text-white"
                : "border-line bg-white text-slate hover:bg-field",
            )}
          >
            {f}
          </button>
        ))}
      </div>

      {/* Línea de tiempo */}
      <div className="relative flex flex-col gap-0 pl-6">
        {/* línea vertical */}
        <div className="absolute left-[11px] top-0 h-full w-px bg-line" />

        {visible.map((entry) => {
          const Icon = TYPE_ICON[entry.type];
          const open = expanded === entry.id;
          return (
            <div key={entry.id} className="relative mb-4">
              {/* punto en la línea */}
              <div
                className={cn(
                  "absolute -left-6 flex size-5 items-center justify-center rounded-full border-2 border-white",
                  entry.tone === "info"
                    ? "bg-teal"
                    : entry.tone === "navy"
                    ? "bg-navy"
                    : entry.tone === "success"
                    ? "bg-success"
                    : "bg-line",
                )}
              />
              <Card className="ml-2 overflow-hidden">
                <button
                  onClick={() => setExpanded(open ? null : entry.id)}
                  className="flex w-full items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-shell"
                >
                  {/* icono tipo */}
                  <div
                    className={cn(
                      "flex size-9 shrink-0 items-center justify-center rounded-lg border",
                      TYPE_TONE[entry.type],
                    )}
                  >
                    <Icon className="size-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <Badge tone={entry.tone as "info" | "navy" | "success" | "neutral"}>
                        {entry.type}
                      </Badge>
                      <span className="font-mono text-xs text-muted">{entry.id}</span>
                    </div>
                    <p className="mt-0.5 truncate text-sm font-medium text-ink">
                      {entry.summary}
                    </p>
                    <p className="text-xs text-muted">
                      {entry.date} · {entry.doctor}
                    </p>
                  </div>
                  {open ? (
                    <ChevronUp className="size-4 shrink-0 text-muted" />
                  ) : (
                    <ChevronDown className="size-4 shrink-0 text-muted" />
                  )}
                </button>
                {open && (
                  <div className="border-t border-line bg-shell px-5 py-4 text-sm leading-relaxed text-slate">
                    {entry.detail}
                  </div>
                )}
              </Card>
            </div>
          );
        })}

        {visible.length === 0 && (
          <p className="ml-2 py-8 text-center text-sm text-muted">
            No hay entradas del tipo "{filter}".
          </p>
        )}
      </div>
    </div>
  );
}
