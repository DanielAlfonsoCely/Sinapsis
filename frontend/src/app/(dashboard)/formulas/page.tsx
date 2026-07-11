"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ChevronLeft, Pill, FileText, Ban } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

type Paciente = {
  id: string;
  nombre_paciente: string;
  apellidos_paciente: string;
  tipo_documento: string;
  numero_documento: string;
};

type Medicamento = {
  nombre: string;
  dosis: string;
  frecuencia: string;
  duracion: string;
  cantidad: string;
};

type Formula = {
  id: string;
  consulta_id: string | null;
  medicamentos: Medicamento[];
  indicaciones: string | null;
  fecha_prescripcion: string;
  fecha_vencimiento: string | null;
  estado_formula: string;
  medico_nombre: string;
};

function formatDate(iso: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("es-CO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  });
}

function Formulas() {
  const searchParams = useSearchParams();
  const pacienteId = searchParams.get("paciente");

  const [paciente, setPaciente] = useState<Paciente | null>(null);
  const [formulas, setFormulas] = useState<Formula[]>([]);
  const [loading, setLoading] = useState(true);

  const token = () =>
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  const fetchFormulas = useCallback(async () => {
    if (!pacienteId) return;
    const t = token();
    const res = await fetch(
      `http://localhost:8080/api/v1/pacientes/${pacienteId}/formulas`,
      { headers: t ? { Authorization: `Bearer ${t}` } : undefined },
    );
    if (res.ok) setFormulas((await res.json()).formulas ?? []);
  }, [pacienteId]);

  useEffect(() => {
    (async () => {
      if (!pacienteId) {
        setLoading(false);
        return;
      }
      setLoading(true);
      try {
        const t = token();
        const headers = t ? { Authorization: `Bearer ${t}` } : undefined;
        const pRes = await fetch(
          `http://localhost:8080/api/v1/pacientes/${pacienteId}`,
          { headers },
        );
        if (pRes.ok) setPaciente(await pRes.json());
        await fetchFormulas();
      } finally {
        setLoading(false);
      }
    })();
  }, [pacienteId, fetchFormulas]);

  async function anular(id: string) {
    const t = token();
    const res = await fetch(
      `http://localhost:8080/api/v1/formulas/${id}/anular`,
      {
        method: "POST",
        headers: t ? { Authorization: `Bearer ${t}` } : undefined,
      },
    );
    if (res.ok) fetchFormulas();
  }

  if (!pacienteId) {
    return (
      <Card className="flex flex-col items-center gap-3 p-10 text-center">
        <Pill className="size-10 text-muted" />
        <p className="font-medium text-ink">No se seleccionó ningún paciente</p>
        <Button asChild variant="outline">
          <Link href="/pacientes">Ir a Pacientes</Link>
        </Button>
      </Card>
    );
  }

  if (loading) {
    return <Card className="p-10 text-center text-slate">Cargando…</Card>;
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
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
            Fórmulas médicas
          </h2>
          {paciente && (
            <p className="text-sm text-slate">
              {paciente.nombre_paciente} {paciente.apellidos_paciente} ·{" "}
              {paciente.tipo_documento} {paciente.numero_documento}
            </p>
          )}
        </div>
      </div>

      <p className="text-xs text-muted">
        Las fórmulas se emiten durante la consulta del paciente.
      </p>

      {/* Listado */}
      {formulas.length === 0 ? (
        <Card className="p-10 text-center text-slate">
          Este paciente aún no tiene fórmulas registradas.
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          {formulas.map((f) => {
            const vigente = f.estado_formula === "vigente";
            return (
              <Card key={f.id} className="flex flex-col gap-3 p-5">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span className="flex size-8 items-center justify-center rounded-lg bg-teal/10 text-teal">
                      <Pill className="size-4" />
                    </span>
                    <div>
                      <p className="text-sm font-medium text-ink">
                        Fórmula del {formatDate(f.fecha_prescripcion)}
                      </p>
                      <p className="text-xs text-muted">
                        {f.medico_nombre}
                        {f.fecha_vencimiento
                          ? ` · vence ${formatDate(f.fecha_vencimiento)}`
                          : ""}
                      </p>
                    </div>
                  </div>
                  <Badge tone={vigente ? "success" : "neutral"}>
                    {vigente ? "Vigente" : "Anulada"}
                  </Badge>
                </div>

                <ul className="flex flex-col gap-1 rounded-[var(--radius)] border border-line bg-shell p-3">
                  {f.medicamentos.map((m, i) => (
                    <li key={i} className="text-sm text-navy-800">
                      <span className="font-medium">{m.nombre}</span>
                      {m.dosis && ` · ${m.dosis}`}
                      {m.frecuencia && ` · ${m.frecuencia}`}
                      {m.duracion && ` · ${m.duracion}`}
                      {m.cantidad && ` · ${m.cantidad}`}
                    </li>
                  ))}
                </ul>

                {f.indicaciones && (
                  <p className="text-sm text-slate">
                    <span className="text-xs uppercase tracking-[0.6px] text-label">
                      Indicaciones:{" "}
                    </span>
                    {f.indicaciones}
                  </p>
                )}

                <div className="flex flex-wrap justify-end gap-2">
                  {f.consulta_id && (
                    <Button variant="outline" size="sm" asChild>
                      <Link
                        href={`/historia-clinica?paciente=${pacienteId}&consulta=${f.consulta_id}`}
                      >
                        <FileText className="size-3.5" />
                        Ver en HC
                      </Link>
                    </Button>
                  )}
                  {vigente && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => anular(f.id)}
                    >
                      <Ban className="size-3.5" />
                      Anular
                    </Button>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default function FormulasPage() {
  return (
    <Suspense
      fallback={<Card className="p-10 text-center text-slate">Cargando…</Card>}
    >
      <Formulas />
    </Suspense>
  );
}
