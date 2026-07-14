"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  ChevronLeft,
  Stethoscope,
  ChevronDown,
  ChevronUp,
  Droplet,
  AlertCircle,
  User,
  Activity,
  CalendarClock,
  FileText,
  Pill,
  Paperclip,
  ImageIcon,
  FileDown,
  Loader2,
  BrainCircuit,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

type Paciente = {
  id: string;
  numero_documento: string;
  tipo_documento: string;
  nombre_paciente: string;
  apellidos_paciente: string;
  fecha_nacimiento: string;
  tipo_sangre: string | null;
  alergias: string | null;
  aseguradora: string | null;
  tiene_cita_hoy: boolean;
};

type SugerenciaIAResumen = {
  id: string;
  examinagen_id: string;
  modelo_ia_utilizado: string;
  estado_procesamiento: "enviado" | "completado" | "fallido";
  diagnostico_sugerido: string | null;
  descripcion_hallazgo: string | null;
  estado_revision: "pendiente" | "revisada" | "rechazada";
};

type Consulta = {
  id: string;
  tipo_consulta: string | null;
  motivo_consulta: string;
  anamnesis: string | null;
  revision_sistemas: string | null;
  examen_fisico: string | null;
  hallazgos_clinicos: string | null;
  // Impresión clínica inicial del médico, requerida antes de usar IA (RF-12).
  pre_diagnostico: string | null;
  presion_arterial: string | null;
  frecuencia_cardiaca: number | null;
  frecuencia_respiratoria: number | null;
  temperatura: number | null;
  saturacion_oxigeno: number | null;
  peso_kg: number | null;
  talla_cm: number | null;
  diagnostico_principal: string | null;
  diagnostico_cie10: string | null;
  plan_manejo: string | null;
  procedimientos_indicados: string | null;
  observaciones_medico: string | null;
  proxima_cita: string | null;
  fecha_consulta: string;
  estado_consulta: string;
  medico_nombre: string;
  medico_especialidad: string;
  anexos: { id: string; nombre: string; tipo: string }[];
  sugerencias_ia: SugerenciaIAResumen[];
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
  estado_formula: string;
  fecha_prescripcion: string;
};

function initials(nombre: string, apellidos: string) {
  return `${nombre?.[0] ?? ""}${apellidos?.[0] ?? ""}`.toUpperCase();
}

function calcAge(iso: string) {
  const dob = new Date(iso);
  const diff = Date.now() - dob.getTime();
  return Math.abs(new Date(diff).getUTCFullYear() - 1970);
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("es-CO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  });
}

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString("es-CO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function HistoriaClinica() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const pacienteId = searchParams.get("paciente");
  const consultaParam = searchParams.get("consulta"); // deep-link desde una fórmula

  const [paciente, setPaciente] = useState<Paciente | null>(null);
  const [consultas, setConsultas] = useState<Consulta[]>([]);
  const [formulas, setFormulas] = useState<Formula[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [pdfLoading, setPdfLoading] = useState(false);
  const [pdfError, setPdfError] = useState<string | null>(null);

  // Abre un anexo en una pestaña nueva (se pide con token y se muestra como blob).
  async function verAnexo(id: string) {
    const token = localStorage.getItem("token");
    const res = await fetch(
      `http://localhost:8080/api/v1/anexos/${id}/archivo`,
      { headers: token ? { Authorization: `Bearer ${token}` } : undefined },
    );
    if (!res.ok) return;
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank", "noopener,noreferrer");
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  }

  async function exportarPDF() {
    if (!pacienteId) return;
    setPdfLoading(true);
    setPdfError(null);
    try {
      const token = localStorage.getItem("token");
      const res = await fetch(
        `http://localhost:8080/api/v1/pacientes/${pacienteId}/historia-clinica/pdf`,
        { headers: token ? { Authorization: `Bearer ${token}` } : {} },
      );
      if (res.status === 403) {
        setPdfError("No tienes permisos para exportar esta historia clínica.");
        return;
      }
      if (res.status === 404) {
        setPdfError("Este paciente no tiene historia clínica registrada.");
        return;
      }
      if (!res.ok) {
        setPdfError("No se pudo generar el PDF. Intenta de nuevo.");
        return;
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      const cd = res.headers.get("Content-Disposition") ?? "";
      const match = cd.match(/filename="?([^";\s]+)"?/);
      a.download = match?.[1] ?? "historia_clinica.pdf";
      a.href = url;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      setPdfError("Error de red. Intenta de nuevo.");
    } finally {
      setPdfLoading(false);
    }
  }

  useEffect(() => {
    (async () => {
      if (!pacienteId) {
        setLoading(false);
        return;
      }
      setLoading(true);
      try {
        const token = localStorage.getItem("token");
        const headers = token ? { Authorization: `Bearer ${token}` } : undefined;
        const [pRes, cRes, fRes] = await Promise.all([
          fetch(`http://localhost:8080/api/v1/pacientes/${pacienteId}`, {
            headers,
          }),
          fetch(
            `http://localhost:8080/api/v1/pacientes/${pacienteId}/consultas`,
            { headers },
          ),
          fetch(
            `http://localhost:8080/api/v1/pacientes/${pacienteId}/formulas`,
            { headers },
          ),
        ]);
        if (pRes.ok) setPaciente(await pRes.json());
        if (fRes.ok) setFormulas((await fRes.json()).formulas ?? []);
        if (cRes.ok) {
          const data = await cRes.json();
          const list: Consulta[] = data.consultas ?? [];
          setConsultas(list);
          // Si venimos desde una fórmula, abrir esa consulta; si no, la más reciente.
          const target =
            consultaParam && list.some((c) => c.id === consultaParam)
              ? consultaParam
              : list[0]?.id ?? null;
          setExpanded(target);
        }
      } catch {
        // se refleja como historia vacía
      } finally {
        setLoading(false);
      }
    })();
  }, [pacienteId, consultaParam]);

  if (!pacienteId) {
    return (
      <Card className="flex flex-col items-center gap-3 p-10 text-center">
        <FileText className="size-10 text-muted" />
        <p className="font-medium text-ink">No se seleccionó ningún paciente</p>
        <Button asChild variant="outline">
          <Link href="/pacientes">Ir a Pacientes</Link>
        </Button>
      </Card>
    );
  }

  if (loading) {
    return (
      <Card className="p-10 text-center text-slate">Cargando historia…</Card>
    );
  }

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
            <p className="text-sm text-slate">Registro unificado — inmutable</p>
          </div>
        </div>
        {/* Solo se puede consultar con cita activa para hoy */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={exportarPDF}
            disabled={pdfLoading}
          >
            {pdfLoading
              ? <Loader2 className="size-4 animate-spin" />
              : <FileDown className="size-4" />}
            {pdfLoading ? "Generando…" : "Exportar PDF"}
          </Button>
          {paciente?.tiene_cita_hoy ? (
            <Button variant="outline" size="sm" asChild>
              <Link href={`/consulta?paciente=${pacienteId}`}>
                <Stethoscope className="size-4" />
                Nueva consulta
              </Link>
            </Button>
          ) : (
            <span
              title="Sin cita programada para hoy. Agenda una cita desde Pacientes para poder consultar."
              className="inline-flex cursor-not-allowed items-center gap-1.5 rounded-[var(--radius)] border border-line bg-field px-3 py-1.5 text-sm font-medium text-muted"
            >
              <Stethoscope className="size-4" />
              Nueva consulta
            </span>
          )}
        </div>
      </div>
      {pdfError && (
        <p className="mt-1 text-xs text-danger">{pdfError}</p>
      )}

      {/* Banner paciente */}
      {paciente && (
        <Card className="flex flex-col gap-4 p-5 sm:flex-row sm:items-start">
          <span className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 font-display text-xl font-bold text-teal">
            {initials(paciente.nombre_paciente, paciente.apellidos_paciente)}
          </span>
          <div className="grid flex-1 grid-cols-2 gap-x-8 gap-y-2 sm:grid-cols-4">
            <div>
              <p className="text-xs uppercase tracking-[0.6px] text-label">
                Nombre
              </p>
              <p className="font-semibold text-ink">
                {paciente.nombre_paciente} {paciente.apellidos_paciente}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.6px] text-label">
                Documento
              </p>
              <p className="font-mono text-navy-800">
                {paciente.tipo_documento} {paciente.numero_documento}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.6px] text-label">
                <User className="mr-1 inline size-3" />
                Edad / Nacimiento
              </p>
              <p className="text-navy-800">
                {calcAge(paciente.fecha_nacimiento)} años ·{" "}
                {formatDate(paciente.fecha_nacimiento)}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.6px] text-label">
                Aseguradora
              </p>
              <p className="text-navy-800">{paciente.aseguradora ?? "—"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.6px] text-label">
                <Droplet className="mr-1 inline size-3" />
                Grupo sanguíneo
              </p>
              {paciente.tipo_sangre ? (
                <Badge tone="danger">{paciente.tipo_sangre}</Badge>
              ) : (
                <p className="text-navy-800">—</p>
              )}
            </div>
            <div className="col-span-2">
              <p className="text-xs uppercase tracking-[0.6px] text-label">
                <AlertCircle className="mr-1 inline size-3" />
                Alergias
              </p>
              <p className="text-navy-800">{paciente.alergias ?? "Ninguna registrada"}</p>
            </div>
          </div>
        </Card>
      )}

      {/* Resumen */}
      <p className="text-sm text-slate">
        {consultas.length}{" "}
        {consultas.length === 1 ? "consulta registrada" : "consultas registradas"}
      </p>

      {/* Línea de tiempo de consultas */}
      {consultas.length === 0 ? (
        <Card className="flex flex-col items-center gap-3 p-10 text-center">
          <Stethoscope className="size-10 text-muted" />
          <p className="font-medium text-ink">
            Este paciente aún no tiene consultas
          </p>
          <p className="text-sm text-slate">
            Registre la primera consulta para iniciar su historial clínico.
          </p>
          <Button asChild>
            <Link href={`/consulta?paciente=${pacienteId}`}>
              <Stethoscope className="size-4" />
              Registrar consulta
            </Link>
          </Button>
        </Card>
      ) : (
        <div className="relative flex flex-col gap-0 pl-6">
          <div className="absolute left-[11px] top-0 h-full w-px bg-line" />
          {consultas.map((c) => {
            const open = expanded === c.id;
            return (
              <div key={c.id} className="relative mb-4">
                <div className="absolute -left-6 flex size-5 items-center justify-center rounded-full border-2 border-white bg-teal" />
                <Card className="ml-2 overflow-hidden">
                  <button
                    onClick={() => setExpanded(open ? null : c.id)}
                    className="flex w-full items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-shell"
                  >
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-teal/20 bg-teal/10 text-teal">
                      <Stethoscope className="size-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge tone="info">
                          {c.tipo_consulta ?? "Consulta"}
                        </Badge>
                        {c.diagnostico_cie10 && (
                          <span className="font-mono text-xs text-muted">
                            {c.diagnostico_cie10}
                          </span>
                        )}
                      </div>
                      <p className="mt-0.5 truncate text-sm font-medium text-ink">
                        {c.diagnostico_principal || c.motivo_consulta}
                      </p>
                      <p className="text-xs text-muted">
                        {formatDateTime(c.fecha_consulta)} · {c.medico_nombre}
                        {c.medico_especialidad
                          ? ` · ${c.medico_especialidad}`
                          : ""}
                      </p>
                    </div>
                    {open ? (
                      <ChevronUp className="size-4 shrink-0 text-muted" />
                    ) : (
                      <ChevronDown className="size-4 shrink-0 text-muted" />
                    )}
                  </button>

                  {open && (
                    <div className="flex flex-col gap-4 border-t border-line bg-shell px-5 py-4 text-sm">
                      <DetailRow label="Motivo de consulta" value={c.motivo_consulta} />

                      {/* Pre-diagnóstico: impresión clínica inicial del médico,
                          registrada ANTES de usar cualquier herramienta de IA (RF-12). */}
                      {c.pre_diagnostico && (
                        <div className="flex items-start gap-2 rounded-[var(--radius)] border border-teal/25 bg-teal/5 p-3">
                          <BrainCircuit className="size-4 shrink-0 text-teal mt-0.5" />
                          <div>
                            <p className="text-xs uppercase tracking-[0.6px] text-teal">
                              Pre-diagnóstico (impresión clínica inicial)
                            </p>
                            <p className="mt-0.5 whitespace-pre-wrap leading-relaxed text-slate">
                              {c.pre_diagnostico}
                            </p>
                          </div>
                        </div>
                      )}

                      <DetailRow label="Enfermedad actual / Anamnesis" value={c.anamnesis} />
                      <DetailRow label="Revisión por sistemas" value={c.revision_sistemas} />

                      {(c.presion_arterial ||
                        c.frecuencia_cardiaca ||
                        c.frecuencia_respiratoria ||
                        c.temperatura ||
                        c.saturacion_oxigeno ||
                        c.peso_kg ||
                        c.talla_cm) && (
                        <div>
                          <p className="mb-1.5 flex items-center gap-1 text-xs uppercase tracking-[0.6px] text-label">
                            <Activity className="size-3" />
                            Signos vitales
                          </p>
                          <div className="flex flex-wrap gap-2">
                            <Vital label="PA" value={c.presion_arterial} />
                            <Vital label="FC" value={c.frecuencia_cardiaca} unit="lpm" />
                            <Vital label="FR" value={c.frecuencia_respiratoria} unit="rpm" />
                            <Vital label="Temp" value={c.temperatura} unit="°C" />
                            <Vital label="SatO₂" value={c.saturacion_oxigeno} unit="%" />
                            <Vital label="Peso" value={c.peso_kg} unit="kg" />
                            <Vital label="Talla" value={c.talla_cm} unit="cm" />
                          </div>
                        </div>
                      )}

                      <DetailRow label="Examen físico" value={c.examen_fisico} />
                      <DetailRow label="Hallazgos clínicos" value={c.hallazgos_clinicos} />
                      <DetailRow
                        label="Diagnóstico"
                        value={
                          c.diagnostico_principal
                            ? `${c.diagnostico_principal}${c.diagnostico_cie10 ? ` (${c.diagnostico_cie10})` : ""}`
                            : null
                        }
                      />
                      <DetailRow label="Plan de manejo" value={c.plan_manejo} />
                      <DetailRow label="Procedimientos / remisiones" value={c.procedimientos_indicados} />
                      <DetailRow label="Observaciones del médico" value={c.observaciones_medico} />
                      {c.proxima_cita && (
                        <p className="flex items-center gap-1.5 text-xs text-slate">
                          <CalendarClock className="size-3.5" />
                          Próxima cita: {formatDate(c.proxima_cita)}
                        </p>
                      )}

                      {/* Anexos: no se muestran inline; se abren con un botón */}
                      {c.anexos.length > 0 && (
                        <div>
                          <p className="mb-1.5 flex items-center gap-1 text-xs uppercase tracking-[0.6px] text-label">
                            <Paperclip className="size-3" />
                            Anexos ({c.anexos.length})
                          </p>
                          <div className="flex flex-wrap gap-2">
                              {c.anexos.map((a) => (
                              <div key={a.id} className="flex items-center gap-1.5">
                                <button
                                  type="button"
                                  onClick={() => verAnexo(a.id)}
                                  className="inline-flex items-center gap-1.5 rounded-[var(--radius)] border border-teal/30 bg-teal/5 px-3 py-1.5 text-xs font-medium text-teal hover:bg-teal/10"
                                >
                                  {a.tipo === "imagen" ? (
                                    <ImageIcon className="size-3.5" />
                                  ) : (
                                    <FileText className="size-3.5" />
                                  )}
                                  Ver {a.nombre}
                                </button>
                                <button
                                  type="button"
                                  onClick={() => router.push(`/analisis-ia?examinagenId=${a.id}`)}
                                  className="inline-flex items-center gap-1.5 rounded-[var(--radius)] border border-navylight/30 bg-navylight/5 px-3 py-1.5 text-xs font-medium text-ink hover:bg-navylight/10"
                                >
                                  <BrainCircuit className="size-3.5" />
                                  IA
                                </button>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* Sugerencias de análisis IA sobre exámenes de esta consulta.
                          Aquí queda visible el resultado de "usar la sugerencia"
                          (marcarla como revisada) desde la pantalla de análisis IA. */}
                      {c.sugerencias_ia.length > 0 && (
                        <div>
                          <p className="mb-1.5 flex items-center gap-1 text-xs uppercase tracking-[0.6px] text-label">
                            <BrainCircuit className="size-3" />
                            Sugerencias de IA ({c.sugerencias_ia.length})
                          </p>
                          <div className="flex flex-col gap-2">
                            {c.sugerencias_ia.map((s) => (
                              <Link
                                key={s.id}
                                href={`/analisis-ia?id=${s.id}`}
                                className="flex flex-col gap-1 rounded-[var(--radius)] border border-line bg-white p-3 transition-colors hover:border-teal/40"
                              >
                                <div className="flex flex-wrap items-center justify-between gap-2">
                                  <span className="text-xs font-medium text-ink">
                                    {s.modelo_ia_utilizado || "Modelo IA"}
                                  </span>
                                  <div className="flex items-center gap-1.5">
                                    {s.estado_procesamiento === "enviado" && (
                                      <Badge tone="info">Procesando</Badge>
                                    )}
                                    {s.estado_procesamiento === "fallido" && (
                                      <Badge tone="danger">Falló</Badge>
                                    )}
                                    {s.estado_procesamiento === "completado" && (
                                      <Badge
                                        tone={
                                          s.estado_revision === "revisada"
                                            ? "success"
                                            : s.estado_revision === "rechazada"
                                            ? "danger"
                                            : "warning"
                                        }
                                      >
                                        {s.estado_revision === "revisada"
                                          ? "Revisada por el médico"
                                          : s.estado_revision === "rechazada"
                                          ? "Rechazada"
                                          : "Pendiente de revisión"}
                                      </Badge>
                                    )}
                                  </div>
                                </div>
                                {s.diagnostico_sugerido && (
                                  <p className="text-sm text-slate">
                                    {s.diagnostico_sugerido}
                                  </p>
                                )}
                              </Link>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* Fórmulas recetadas en esta consulta */}
                      {formulas
                        .filter((f) => f.consulta_id === c.id)
                        .map((f) => {
                          const vigente = f.estado_formula === "vigente";
                          return (
                            <div
                              key={f.id}
                              className="rounded-[var(--radius)] border border-line bg-white p-3"
                            >
                              <div className="mb-1.5 flex items-center gap-2">
                                <Pill className="size-3.5 text-teal" />
                                <span className="text-xs uppercase tracking-[0.6px] text-label">
                                  Fórmula médica
                                </span>
                                <Badge tone={vigente ? "success" : "neutral"}>
                                  {vigente ? "Vigente" : "Anulada"}
                                </Badge>
                              </div>
                              <ul className="flex flex-col gap-0.5">
                                {f.medicamentos.map((m, i) => (
                                  <li key={i} className="text-sm text-navy-800">
                                    <span className="font-medium">
                                      {m.nombre}
                                    </span>
                                    {m.dosis && ` · ${m.dosis}`}
                                    {m.frecuencia && ` · ${m.frecuencia}`}
                                    {m.duracion && ` · ${m.duracion}`}
                                  </li>
                                ))}
                              </ul>
                            </div>
                          );
                        })}
                    </div>
                  )}
                </Card>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string | null }) {
  if (!value) return null;
  return (
    <div>
      <p className="text-xs uppercase tracking-[0.6px] text-label">{label}</p>
      <p className="mt-0.5 whitespace-pre-wrap leading-relaxed text-slate">
        {value}
      </p>
    </div>
  );
}

function Vital({
  label,
  value,
  unit,
}: {
  label: string;
  value: string | number | null;
  unit?: string;
}) {
  if (value === null || value === "") return null;
  return (
    <span className="inline-flex items-baseline gap-1 rounded-[var(--radius)] border border-line bg-white px-3 py-1.5">
      <span className="text-xs text-muted">{label}</span>
      <span className="text-sm font-medium text-navy-800">
        {value}
        {unit ? ` ${unit}` : ""}
      </span>
    </span>
  );
}

export default function HistoriaClinicaPage() {
  return (
    <Suspense
      fallback={
        <Card className="p-10 text-center text-slate">Cargando…</Card>
      }
    >
      <HistoriaClinica />
    </Suspense>
  );
}
