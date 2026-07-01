"use client";

import { useState } from "react";
import Link from "next/link";
import { ChevronLeft, Plus, Printer, Download, Pill, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

type FormulaStatus = "Vigente" | "Reclamada" | "Vencida" | "Atrasada";

const STATUS_TONE: Record<FormulaStatus, "success" | "info" | "neutral" | "danger"> = {
  Vigente: "success",
  Reclamada: "info",
  Vencida: "neutral",
  Atrasada: "danger",
};

const FORMULAS = [
  {
    id: "F-2026-0041",
    medicamento: "Amoxicilina 500 mg",
    dosis: "1 cápsula",
    frecuencia: "Cada 8 horas",
    duracion: "7 días",
    emision: "1 Jul 2026",
    status: "Vigente" as FormulaStatus,
  },
  {
    id: "F-2026-0038",
    medicamento: "Ibuprofeno 400 mg",
    dosis: "1 tableta",
    frecuencia: "Cada 6 horas (con comida)",
    duracion: "5 días",
    emision: "1 Jul 2026",
    status: "Vigente" as FormulaStatus,
  },
  {
    id: "F-2026-0022",
    medicamento: "Loratadina 10 mg",
    dosis: "1 tableta",
    frecuencia: "Cada 24 horas",
    duracion: "30 días",
    emision: "15 May 2026",
    status: "Reclamada" as FormulaStatus,
  },
  {
    id: "F-2026-0015",
    medicamento: "Metformina 850 mg",
    dosis: "1 tableta",
    frecuencia: "2 veces al día (con comida)",
    duracion: "60 días",
    emision: "18 Mar 2026",
    status: "Vencida" as FormulaStatus,
  },
  {
    id: "F-2026-0009",
    medicamento: "Atorvastatina 20 mg",
    dosis: "1 tableta",
    frecuencia: "Cada 24 horas (noche)",
    duracion: "90 días",
    emision: "10 Feb 2026",
    status: "Atrasada" as FormulaStatus,
  },
];

const FILTERS: Array<FormulaStatus | "Todas"> = [
  "Todas", "Vigente", "Reclamada", "Vencida", "Atrasada",
];

const PATIENT = {
  name: "Ricardo Mendoza",
  id: "1.024.556.782",
  eps: "Nueva EPS",
};

export default function FormulasPage() {
  const [filter, setFilter] = useState<FormulaStatus | "Todas">("Todas");
  const [showNew, setShowNew] = useState(false);

  const visible =
    filter === "Todas" ? FORMULAS : FORMULAS.filter((f) => f.status === filter);

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/consulta">
              <ChevronLeft className="size-4" />
              Consulta
            </Link>
          </Button>
          <div className="h-5 w-px bg-line" />
          <div>
            <h2 className="font-display text-2xl font-semibold text-ink">
              Fórmulas Médicas
            </h2>
            <p className="text-sm text-slate">
              {PATIENT.name} · CC {PATIENT.id} · {PATIENT.eps}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm">
            <Printer className="size-4" />
            Imprimir
          </Button>
          <Button variant="outline" size="sm">
            <Download className="size-4" />
            Exportar
          </Button>
          <Button size="sm" onClick={() => setShowNew(true)}>
            <Plus className="size-4" />
            Nuevo medicamento
          </Button>
        </div>
      </div>

      {/* Formulario nuevo medicamento */}
      {showNew && (
        <Card className="flex flex-col gap-4 border-teal/30 bg-teal/5 p-5">
          <h3 className="font-display font-semibold text-ink">
            Agregar medicamento
          </h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Field label="Medicamento">
              <Input placeholder="Nombre + concentración" />
            </Field>
            <Field label="Dosis">
              <Input placeholder="Ej. 1 tableta" />
            </Field>
            <Field label="Frecuencia">
              <Input placeholder="Ej. Cada 8 horas" />
            </Field>
            <Field label="Duración">
              <Input placeholder="Ej. 7 días" />
            </Field>
          </div>
          <Field label="Indicaciones adicionales">
            <textarea
              className="h-16 w-full resize-none rounded-[var(--radius)] border border-line bg-white px-4 py-2 text-sm text-navy-800 placeholder:text-muted outline-none focus:border-teal focus:ring-2 focus:ring-teal/20"
              placeholder="Indicaciones al paciente (opcional)…"
            />
          </Field>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setShowNew(false)}>
              Cancelar
            </Button>
            <Button onClick={() => setShowNew(false)}>
              <Pill className="size-4" />
              Agregar
            </Button>
          </div>
        </Card>
      )}

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

      {/* Tabla */}
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">ID / Medicamento</th>
              <th className="px-6 py-4 font-normal">Dosis</th>
              <th className="px-6 py-4 font-normal">Frecuencia</th>
              <th className="px-6 py-4 font-normal">Duración</th>
              <th className="px-6 py-4 font-normal">Fecha emisión</th>
              <th className="px-6 py-4 font-normal">Estado</th>
              <th className="px-6 py-4 font-normal text-right">Acción</th>
            </tr>
          </thead>
          <tbody>
            {visible.map((f) => (
              <tr key={f.id} className="border-t border-line hover:bg-shell">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-2">
                    <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-[#e1f0f3]">
                      <Pill className="size-4 text-teal" />
                    </div>
                    <div>
                      <p className="font-medium text-navy-800">{f.medicamento}</p>
                      <p className="font-mono text-xs text-muted">{f.id}</p>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 text-slate">{f.dosis}</td>
                <td className="px-6 py-4 text-slate">{f.frecuencia}</td>
                <td className="px-6 py-4 text-slate">{f.duracion}</td>
                <td className="px-6 py-4 text-slate">{f.emision}</td>
                <td className="px-6 py-4">
                  <Badge tone={STATUS_TONE[f.status]}>{f.status}</Badge>
                </td>
                <td className="px-6 py-4 text-right">
                  {f.status === "Vigente" && (
                    <button className="inline-flex items-center gap-1 text-xs text-danger hover:opacity-75">
                      <Trash2 className="size-3.5" />
                      Retirar
                    </button>
                  )}
                </td>
              </tr>
            ))}
            {visible.length === 0 && (
              <tr>
                <td colSpan={7} className="px-6 py-12 text-center text-sm text-muted">
                  No hay fórmulas con estado "{filter}".
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
