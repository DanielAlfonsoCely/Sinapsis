"use client";

import { useState } from "react";
import Link from "next/link";
import {
  BrainCircuit,
  AlertTriangle,
  CheckCircle,
  ChevronLeft,
  ImageIcon,
  BarChart2,
  Info,
  ShieldAlert,
  Clock,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const FINDINGS = [
  {
    label: "Opacidad en lóbulo inferior derecho",
    confidence: 87,
    tone: "danger" as const,
  },
  {
    label: "Patrón intersticial bilateral leve",
    confidence: 61,
    tone: "warning" as const,
  },
  {
    label: "Sin derrame pleural evidente",
    confidence: 94,
    tone: "success" as const,
  },
  {
    label: "Silueta cardíaca normal",
    confidence: 96,
    tone: "success" as const,
  },
];

const PATIENT = {
  name: "Ricardo Mendoza",
  id: "1.024.556.782",
  type: "Radiografía de tórax PA",
  date: "1 Jul 2026 · 10:35 AM",
};

export default function AnalisisIAPage() {
  const [analysisReady] = useState(true);
  const [accepted, setAccepted] = useState(false);

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/consulta">
              <ChevronLeft className="size-4" />
              Volver a consulta
            </Link>
          </Button>
          <div className="h-5 w-px bg-line" />
          <div>
            <h2 className="font-display text-2xl font-semibold text-ink">
              Análisis IA
            </h2>
            <p className="text-sm text-slate">
              Asistido por MONAI · {PATIENT.type}
            </p>
          </div>
        </div>
        <Badge tone="navy">
          <BrainCircuit className="size-3" />
          MONAI v2.3
        </Badge>
      </div>

      {/* Aviso ético — siempre visible */}
      <div className="flex items-start gap-3 rounded-[var(--radius)] border border-warning/30 bg-warning/8 p-4">
        <ShieldAlert className="size-5 shrink-0 text-warning mt-0.5" />
        <div className="text-sm">
          <p className="font-semibold text-warning">
            Aviso de responsabilidad clínica
          </p>
          <p className="mt-0.5 text-slate">
            Este análisis es una <strong>sugerencia de apoyo diagnóstico</strong>{" "}
            generada por inteligencia artificial y{" "}
            <strong>no constituye un diagnóstico clínico</strong>. El médico
            tratante conserva la responsabilidad clínica y legal sobre el
            diagnóstico y el tratamiento del paciente. (Resolución 1995/1999 —
            Ley 1581/2012)
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_380px]">
        {/* Panel imagen */}
        <Card className="flex flex-col gap-4 p-5">
          <div className="flex items-center justify-between">
            <h3 className="font-display font-semibold text-ink">
              Imagen analizada
            </h3>
            <span className="flex items-center gap-1 text-xs text-muted">
              <Clock className="size-3" />
              Procesado en 4.2 s
            </span>
          </div>
          <div className="flex aspect-[4/3] items-center justify-center rounded-[var(--radius)] bg-navy/90">
            <div className="flex flex-col items-center gap-2 text-white/40">
              <ImageIcon className="size-16" />
              <p className="text-sm">rx_torax_mendoza_2026.jpg</p>
            </div>
          </div>
          <div className="flex items-center gap-3 text-xs text-muted">
            <span>Paciente: <span className="text-ink">{PATIENT.name}</span></span>
            <span>·</span>
            <span>CC {PATIENT.id}</span>
            <span>·</span>
            <span>{PATIENT.date}</span>
          </div>
        </Card>

        {/* Panel resultados */}
        <div className="flex flex-col gap-4">
          <Card className="flex flex-col gap-4 p-5">
            <div className="flex items-center gap-2">
              <BarChart2 className="size-5 text-teal" />
              <h3 className="font-display font-semibold text-ink">
                Hallazgos sugeridos
              </h3>
            </div>

            <div className="flex flex-col gap-3">
              {FINDINGS.map((f) => (
                <div key={f.label} className="flex flex-col gap-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate">{f.label}</span>
                    <span
                      className={cn(
                        "font-semibold",
                        f.tone === "danger"
                          ? "text-danger"
                          : f.tone === "warning"
                          ? "text-warning"
                          : "text-success",
                      )}
                    >
                      {f.confidence}%
                    </span>
                  </div>
                  <div className="h-1.5 w-full rounded-full bg-field">
                    <div
                      className={cn(
                        "h-1.5 rounded-full",
                        f.tone === "danger"
                          ? "bg-danger"
                          : f.tone === "warning"
                          ? "bg-warning"
                          : "bg-success",
                      )}
                      style={{ width: `${f.confidence}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </Card>

          <Card className="flex flex-col gap-3 p-5">
            <div className="flex items-center gap-2">
              <Info className="size-5 text-teal" />
              <h3 className="font-display font-semibold text-ink">
                Sugerencia diagnóstica
              </h3>
            </div>
            <p className="text-sm text-slate leading-relaxed">
              Los hallazgos sugieren un patrón compatible con{" "}
              <strong className="text-ink">neumonía adquirida en comunidad</strong>{" "}
              (CIE-10: J18.9) con mayor probabilidad en el lóbulo inferior
              derecho. Se recomienda correlación clínica y laboratorio.
            </p>
            <div className="flex items-center gap-1.5 text-xs text-muted">
              <AlertTriangle className="size-3 text-warning" />
              Confianza general del modelo: <strong>82%</strong>
            </div>
          </Card>

          {/* Acción */}
          {!accepted ? (
            <Button onClick={() => setAccepted(true)} className="w-full">
              <CheckCircle className="size-4" />
              Usar esta sugerencia en el diagnóstico
            </Button>
          ) : (
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-2 rounded-[var(--radius)] border border-success/30 bg-success/8 p-3 text-sm text-success">
                <CheckCircle className="size-4 shrink-0" />
                Sugerencia incorporada. Puede ajustarla en el paso de diagnóstico.
              </div>
              <Button asChild>
                <Link href="/consulta">
                  <ChevronLeft className="size-4" />
                  Continuar con la consulta
                </Link>
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
