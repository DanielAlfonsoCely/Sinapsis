"use client";

import { useState } from "react";
import Link from "next/link";
import {
  User,
  FileText,
  ImageIcon,
  BrainCircuit,
  Pill,
  Save,
  CheckCircle,
  AlertTriangle,
  Upload,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";

const PATIENT = {
  initials: "RM",
  name: "Ricardo Mendoza",
  id: "1.024.556.782",
  age: 42,
  eps: "Nueva EPS",
  lastVisit: "18 Mar 2026",
};

const STEPS = [
  { id: 1, label: "Pre-diagnóstico", icon: FileText },
  { id: 2, label: "Imagen médica", icon: ImageIcon },
  { id: 3, label: "Diagnóstico final", icon: CheckCircle },
  { id: 4, label: "Tratamiento", icon: Pill },
];

export default function ConsultaPage() {
  const [step, setStep] = useState(1);
  const [preDx, setPreDx] = useState("");
  const [cie10, setCie10] = useState("");
  const [finalDx, setFinalDx] = useState("");
  const [treatment, setTreatment] = useState("");
  const [imageUploaded, setImageUploaded] = useState(false);

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink">
            Nueva Consulta
          </h2>
          <p className="text-sm text-slate">
            Registre el encuentro clínico paso a paso
          </p>
        </div>
        <Badge tone="info">1 Jul 2026 · 10:20 AM</Badge>
      </div>

      {/* Tarjeta paciente */}
      <Card className="flex items-center gap-4 p-5">
        <span className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 font-display text-lg font-semibold text-teal">
          {PATIENT.initials}
        </span>
        <div className="flex-1">
          <p className="font-display font-semibold text-ink">{PATIENT.name}</p>
          <p className="text-sm text-slate">
            CC {PATIENT.id} · {PATIENT.age} años · {PATIENT.eps}
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted">
          <User className="size-4" />
          Última visita: {PATIENT.lastVisit}
        </div>
        <Button variant="ghost" size="sm" asChild>
          <Link href="/historia-clinica">Ver historia</Link>
        </Button>
      </Card>

      {/* Stepper */}
      <div className="flex gap-0">
        {STEPS.map((s, i) => {
          const done = step > s.id;
          const active = step === s.id;
          const Icon = s.icon;
          return (
            <button
              key={s.id}
              onClick={() => setStep(s.id)}
              className="flex flex-1 items-center gap-2 border-b-2 pb-3 text-sm transition-colors"
              style={{
                borderColor: active ? "#286580" : done ? "#1f9d6b" : "#c3c6d0",
                color: active ? "#286580" : done ? "#1f9d6b" : "#747780",
              }}
            >
              <span
                className="flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
                style={{
                  background: active ? "#286580" : done ? "#1f9d6b" : "#f1f3ff",
                  color: active || done ? "white" : "#747780",
                }}
              >
                {done ? "✓" : s.id}
              </span>
              <Icon className="size-4" />
              <span className="hidden sm:inline">{s.label}</span>
            </button>
          );
        })}
      </div>

      {/* Contenido del paso */}
      {step === 1 && (
        <Card className="flex flex-col gap-5 p-6">
          <div>
            <h3 className="font-display text-lg font-semibold text-ink">
              Pre-diagnóstico
            </h3>
            <p className="mt-1 text-sm text-slate">
              Registre su razonamiento clínico antes de acceder al módulo de IA.
              Este paso es obligatorio.
            </p>
          </div>
          <Field label="Motivo de consulta">
            <textarea
              className="h-24 w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 placeholder:text-muted outline-none focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
              placeholder="Describa el motivo de consulta del paciente…"
            />
          </Field>
          <Field label="Pre-diagnóstico del médico (requerido para IA)" htmlFor="predx">
            <textarea
              id="predx"
              value={preDx}
              onChange={(e) => setPreDx(e.target.value)}
              className="h-32 w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 placeholder:text-muted outline-none focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
              placeholder="Registre su hipótesis diagnóstica inicial basada en la evaluación clínica…"
            />
          </Field>
          <div className="flex items-center gap-2 rounded-[var(--radius)] border border-info/20 bg-info/5 p-3 text-xs text-info">
            <AlertTriangle className="size-4 shrink-0" />
            Este pre-diagnóstico quedará registrado en la historia clínica y no puede eliminarse.
          </div>
          <div className="flex justify-end">
            <Button
              onClick={() => preDx.trim() && setStep(2)}
              disabled={!preDx.trim()}
            >
              Continuar
            </Button>
          </div>
        </Card>
      )}

      {step === 2 && (
        <Card className="flex flex-col gap-5 p-6">
          <div>
            <h3 className="font-display text-lg font-semibold text-ink">
              Imagen médica
            </h3>
            <p className="mt-1 text-sm text-slate">
              Suba la imagen diagnóstica (RX, TAC, RMN, etc.) para análisis
              asistido por IA.
            </p>
          </div>

          {!imageUploaded ? (
            <label className="flex flex-col items-center gap-3 rounded-[var(--radius)] border-2 border-dashed border-line bg-field p-12 text-center transition-colors hover:border-teal hover:bg-teal/5 cursor-pointer">
              <Upload className="size-10 text-muted" />
              <div>
                <p className="font-medium text-ink">
                  Arrastre la imagen o haga clic para seleccionar
                </p>
                <p className="text-xs text-muted">DICOM, JPEG, PNG · máx 50 MB</p>
              </div>
              <input
                type="file"
                accept="image/*,.dcm"
                className="sr-only"
                onChange={() => setImageUploaded(true)}
              />
            </label>
          ) : (
            <div className="flex flex-col items-center gap-3 rounded-[var(--radius)] border-2 border-success/40 bg-success/5 p-8 text-center">
              <CheckCircle className="size-10 text-success" />
              <p className="font-medium text-success">Imagen cargada correctamente</p>
              <p className="text-xs text-muted">rx_torax_mendoza_2026.jpg</p>
              <button
                onClick={() => setImageUploaded(false)}
                className="text-xs text-slate underline"
              >
                Cambiar imagen
              </button>
            </div>
          )}

          <div className="flex items-center gap-2 rounded-[var(--radius)] border border-warning/20 bg-warning/5 p-3 text-xs text-warning">
            <BrainCircuit className="size-4 shrink-0" />
            El análisis con IA (MONAI) requiere que la imagen ya esté cargada. El resultado
            no constituye un diagnóstico final.
          </div>

          <div className="flex justify-between">
            <Button variant="outline" onClick={() => setStep(1)}>
              Atrás
            </Button>
            <div className="flex gap-2">
              {imageUploaded && (
                <Button variant="secondary" asChild>
                  <Link href="/analisis-ia">
                    <BrainCircuit className="size-4" />
                    Solicitar Análisis IA
                  </Link>
                </Button>
              )}
              <Button onClick={() => setStep(3)}>
                Continuar sin IA
              </Button>
            </div>
          </div>
        </Card>
      )}

      {step === 3 && (
        <Card className="flex flex-col gap-5 p-6">
          <div>
            <h3 className="font-display text-lg font-semibold text-ink">
              Diagnóstico final
            </h3>
            <p className="mt-1 text-sm text-slate">
              Registre el diagnóstico definitivo con código CIE-10.
            </p>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-[180px_1fr]">
            <Field label="Código CIE-10" htmlFor="cie10">
              <Input
                id="cie10"
                value={cie10}
                onChange={(e) => setCie10(e.target.value)}
                placeholder="J18.9"
              />
            </Field>
            <Field label="Diagnóstico" htmlFor="finaldx">
              <Input
                id="finaldx"
                value={finalDx}
                onChange={(e) => setFinalDx(e.target.value)}
                placeholder="Descripción del diagnóstico definitivo…"
              />
            </Field>
          </div>
          <Field label="Observaciones clínicas">
            <textarea
              className="h-28 w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 placeholder:text-muted outline-none focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
              placeholder="Hallazgos adicionales, evolución, notas del médico…"
            />
          </Field>
          <div className="flex justify-between">
            <Button variant="outline" onClick={() => setStep(2)}>
              Atrás
            </Button>
            <Button onClick={() => setStep(4)}>Continuar</Button>
          </div>
        </Card>
      )}

      {step === 4 && (
        <Card className="flex flex-col gap-5 p-6">
          <div>
            <h3 className="font-display text-lg font-semibold text-ink">
              Plan de tratamiento
            </h3>
            <p className="mt-1 text-sm text-slate">
              Defina el tratamiento y emita la fórmula médica si aplica.
            </p>
          </div>
          <Field label="Plan de tratamiento" htmlFor="treatment">
            <textarea
              id="treatment"
              value={treatment}
              onChange={(e) => setTreatment(e.target.value)}
              className="h-36 w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 placeholder:text-muted outline-none focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
              placeholder="Indique el plan terapéutico, indicaciones al paciente y seguimiento…"
            />
          </Field>
          <Field label="Remisiones / interconsultas">
            <textarea
              className="h-20 w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 placeholder:text-muted outline-none focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
              placeholder="Especialidades a las que se remite (opcional)…"
            />
          </Field>
          <div className="flex items-center justify-between pt-2">
            <Button variant="outline" onClick={() => setStep(3)}>
              Atrás
            </Button>
            <div className="flex gap-2">
              <Button variant="secondary" asChild>
                <Link href="/formulas">
                  <Pill className="size-4" />
                  Emitir Fórmula
                </Link>
              </Button>
              <Button variant="secondary">
                <Save className="size-4" />
                Guardar borrador
              </Button>
              <Button>
                <CheckCircle className="size-4" />
                Finalizar consulta
              </Button>
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
