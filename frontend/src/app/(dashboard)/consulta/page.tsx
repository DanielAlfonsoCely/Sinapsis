"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  User,
  FileText,
  Activity,
  Stethoscope,
  ClipboardList,
  Pill,
  Paperclip,
  Save,
  ChevronLeft,
  Plus,
  Trash2,
  Upload,
  AlertTriangle,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";

const textareaClass =
  "w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-navy-800 placeholder:text-muted outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20";

const selectClass =
  "h-11 w-full rounded-[var(--radius)] border border-line bg-field px-4 text-sm text-navy-800 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20";

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
};

type Medicamento = {
  nombre: string;
  dosis: string;
  frecuencia: string;
  duracion: string;
};

const EMPTY_FORM = {
  tipo_consulta: "Primera vez",
  motivo_consulta: "",
  anamnesis: "",
  revision_sistemas: "",
  presion_arterial: "",
  frecuencia_cardiaca: "",
  frecuencia_respiratoria: "",
  temperatura: "",
  saturacion_oxigeno: "",
  peso_kg: "",
  talla_cm: "",
  examen_fisico: "",
  hallazgos_clinicos: "",
  diagnostico_cie10: "",
  diagnostico_principal: "",
  plan_manejo: "",
  procedimientos_indicados: "",
  observaciones_medico: "",
  proxima_cita: "",
};

function initials(nombre: string, apellidos: string) {
  return `${nombre?.[0] ?? ""}${apellidos?.[0] ?? ""}`.toUpperCase();
}

function calcAge(iso: string) {
  const dob = new Date(iso);
  const diff = Date.now() - dob.getTime();
  return Math.abs(new Date(diff).getUTCFullYear() - 1970);
}

function ConsultaForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const pacienteId = searchParams.get("paciente");

  const [paciente, setPaciente] = useState<Paciente | null>(null);
  const [loadingPaciente, setLoadingPaciente] = useState(true);
  const [form, setForm] = useState(EMPTY_FORM);

  // Medicamentos y adjuntos son solo de la UI por ahora: corresponden a las
  // HU-06 (fórmulas médicas) y HU-07 (adjuntar resultados) y no se envían al back.
  const [medicamentos, setMedicamentos] = useState<Medicamento[]>([]);
  const [medForm, setMedForm] = useState<Medicamento>({
    nombre: "",
    dosis: "",
    frecuencia: "",
    duracion: "",
  });
  const [adjuntos, setAdjuntos] = useState<string[]>([]);

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!pacienteId) {
      setLoadingPaciente(false);
      return;
    }
    (async () => {
      try {
        const token = localStorage.getItem("token");
        const res = await fetch(
          `http://localhost:8080/api/v1/pacientes/${pacienteId}`,
          { headers: token ? { Authorization: `Bearer ${token}` } : undefined },
        );
        if (res.ok) setPaciente(await res.json());
      } catch {
        // se maneja mostrando el aviso de "paciente no encontrado"
      } finally {
        setLoadingPaciente(false);
      }
    })();
  }, [pacienteId]);

  function set<K extends keyof typeof EMPTY_FORM>(key: K, value: string) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  function addMedicamento() {
    if (!medForm.nombre.trim()) return;
    setMedicamentos((m) => [...m, medForm]);
    setMedForm({ nombre: "", dosis: "", frecuencia: "", duracion: "" });
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!pacienteId) return;
    if (!form.motivo_consulta.trim()) {
      setError("El motivo de consulta es obligatorio.");
      return;
    }
    setSaving(true);
    setError("");

    // Solo enviamos al backend los campos clínicos de la consulta (HU-03).
    const body: Record<string, unknown> = {
      paciente_id: pacienteId,
      motivo_consulta: form.motivo_consulta,
    };
    const textFields: (keyof typeof EMPTY_FORM)[] = [
      "tipo_consulta",
      "anamnesis",
      "revision_sistemas",
      "presion_arterial",
      "examen_fisico",
      "hallazgos_clinicos",
      "diagnostico_cie10",
      "diagnostico_principal",
      "plan_manejo",
      "procedimientos_indicados",
      "observaciones_medico",
      "proxima_cita",
    ];
    for (const k of textFields) {
      if (form[k]) body[k] = form[k];
    }
    const intFields: (keyof typeof EMPTY_FORM)[] = [
      "frecuencia_cardiaca",
      "frecuencia_respiratoria",
      "saturacion_oxigeno",
    ];
    for (const k of intFields) {
      if (form[k]) body[k] = parseInt(form[k], 10);
    }
    const floatFields: (keyof typeof EMPTY_FORM)[] = [
      "temperatura",
      "peso_kg",
      "talla_cm",
    ];
    for (const k of floatFields) {
      if (form[k]) body[k] = parseFloat(form[k]);
    }

    try {
      const token = localStorage.getItem("token");
      const res = await fetch("http://localhost:8080/api/v1/consultas", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error ?? "No se pudo registrar la consulta");
        return;
      }
      // Al finalizar, se ve reflejada en la historia clínica del paciente.
      router.push(`/historia-clinica?paciente=${pacienteId}`);
    } catch {
      setError("Error de conexión con el servidor");
    } finally {
      setSaving(false);
    }
  }

  if (!pacienteId) {
    return (
      <Card className="flex flex-col items-center gap-3 p-10 text-center">
        <Stethoscope className="size-10 text-muted" />
        <p className="font-medium text-ink">No se seleccionó ningún paciente</p>
        <p className="text-sm text-slate">
          Las consultas se inician desde la lista de pacientes.
        </p>
        <Button asChild variant="outline">
          <Link href="/pacientes">Ir a Pacientes</Link>
        </Button>
      </Card>
    );
  }

  if (loadingPaciente) {
    return (
      <Card className="p-10 text-center text-slate">Cargando paciente…</Card>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-6">
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
              Nueva Consulta
            </h2>
            <p className="text-sm text-slate">
              Registro del encuentro clínico
            </p>
          </div>
        </div>
        <Badge tone="info">
          {new Date().toLocaleDateString("es-CO", {
            day: "2-digit",
            month: "short",
            year: "numeric",
          })}
        </Badge>
      </div>

      {/* Tarjeta paciente */}
      {paciente ? (
        <Card className="flex flex-wrap items-center gap-4 p-5">
          <span className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 font-display text-lg font-semibold text-teal">
            {initials(paciente.nombre_paciente, paciente.apellidos_paciente)}
          </span>
          <div className="flex-1">
            <p className="font-display font-semibold text-ink">
              {paciente.nombre_paciente} {paciente.apellidos_paciente}
            </p>
            <p className="text-sm text-slate">
              {paciente.tipo_documento} {paciente.numero_documento} ·{" "}
              {calcAge(paciente.fecha_nacimiento)} años
              {paciente.aseguradora ? ` · ${paciente.aseguradora}` : ""}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {paciente.tipo_sangre && (
              <Badge tone="danger">{paciente.tipo_sangre}</Badge>
            )}
            {paciente.alergias && (
              <Badge tone="danger">Alergias: {paciente.alergias}</Badge>
            )}
            <Button variant="ghost" size="sm" asChild>
              <Link href={`/historia-clinica?paciente=${paciente.id}`}>
                <User className="size-4" />
                Ver historia
              </Link>
            </Button>
          </div>
        </Card>
      ) : (
        <Card className="p-5 text-sm text-danger">
          No se encontró el paciente seleccionado.
        </Card>
      )}

      {/* Motivo y anamnesis */}
      <Card className="flex flex-col gap-5 p-6">
        <SectionTitle icon={FileText} title="Motivo y anamnesis" />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-[220px_1fr]">
          <Field label="Tipo de consulta">
            <select
              className={selectClass}
              value={form.tipo_consulta}
              onChange={(e) => set("tipo_consulta", e.target.value)}
            >
              <option>Primera vez</option>
              <option>Control</option>
              <option>Urgencia</option>
              <option>Remisión</option>
            </select>
          </Field>
          <Field label="Motivo de consulta *">
            <textarea
              className={textareaClass}
              rows={2}
              value={form.motivo_consulta}
              onChange={(e) => set("motivo_consulta", e.target.value)}
              placeholder="Motivo por el que consulta el paciente…"
            />
          </Field>
        </div>
        <Field label="Enfermedad actual / Anamnesis">
          <textarea
            className={textareaClass}
            rows={3}
            value={form.anamnesis}
            onChange={(e) => set("anamnesis", e.target.value)}
            placeholder="Evolución del cuadro clínico, tiempo, síntomas asociados…"
          />
        </Field>
        <Field label="Revisión por sistemas">
          <textarea
            className={textareaClass}
            rows={2}
            value={form.revision_sistemas}
            onChange={(e) => set("revision_sistemas", e.target.value)}
            placeholder="Cardiovascular, respiratorio, digestivo, neurológico…"
          />
        </Field>
      </Card>

      {/* Signos vitales */}
      <Card className="flex flex-col gap-5 p-6">
        <SectionTitle icon={Activity} title="Signos vitales" />
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          <Field label="Presión arterial">
            <Input
              value={form.presion_arterial}
              onChange={(e) => set("presion_arterial", e.target.value)}
              placeholder="120/80"
            />
          </Field>
          <Field label="Frec. cardíaca (lpm)">
            <Input
              type="number"
              value={form.frecuencia_cardiaca}
              onChange={(e) => set("frecuencia_cardiaca", e.target.value)}
              placeholder="72"
            />
          </Field>
          <Field label="Frec. respiratoria (rpm)">
            <Input
              type="number"
              value={form.frecuencia_respiratoria}
              onChange={(e) => set("frecuencia_respiratoria", e.target.value)}
              placeholder="16"
            />
          </Field>
          <Field label="Temperatura (°C)">
            <Input
              type="number"
              step="0.1"
              value={form.temperatura}
              onChange={(e) => set("temperatura", e.target.value)}
              placeholder="36.5"
            />
          </Field>
          <Field label="Sat. O₂ (%)">
            <Input
              type="number"
              value={form.saturacion_oxigeno}
              onChange={(e) => set("saturacion_oxigeno", e.target.value)}
              placeholder="98"
            />
          </Field>
          <Field label="Peso (kg)">
            <Input
              type="number"
              step="0.1"
              value={form.peso_kg}
              onChange={(e) => set("peso_kg", e.target.value)}
              placeholder="70"
            />
          </Field>
          <Field label="Talla (cm)">
            <Input
              type="number"
              step="0.1"
              value={form.talla_cm}
              onChange={(e) => set("talla_cm", e.target.value)}
              placeholder="170"
            />
          </Field>
        </div>
      </Card>

      {/* Examen físico y hallazgos */}
      <Card className="flex flex-col gap-5 p-6">
        <SectionTitle icon={Stethoscope} title="Examen físico" />
        <Field label="Examen físico">
          <textarea
            className={textareaClass}
            rows={3}
            value={form.examen_fisico}
            onChange={(e) => set("examen_fisico", e.target.value)}
            placeholder="Hallazgos al examen físico por sistemas…"
          />
        </Field>
        <Field label="Hallazgos clínicos relevantes">
          <textarea
            className={textareaClass}
            rows={2}
            value={form.hallazgos_clinicos}
            onChange={(e) => set("hallazgos_clinicos", e.target.value)}
            placeholder="Hallazgos destacados de la evaluación…"
          />
        </Field>
      </Card>

      {/* Diagnóstico y plan */}
      <Card className="flex flex-col gap-5 p-6">
        <SectionTitle icon={ClipboardList} title="Diagnóstico y plan" />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-[160px_1fr]">
          <Field label="Código CIE-10">
            <Input
              value={form.diagnostico_cie10}
              onChange={(e) => set("diagnostico_cie10", e.target.value)}
              placeholder="J18.9"
            />
          </Field>
          <Field label="Diagnóstico principal">
            <Input
              value={form.diagnostico_principal}
              onChange={(e) => set("diagnostico_principal", e.target.value)}
              placeholder="Descripción del diagnóstico…"
            />
          </Field>
        </div>
        <Field label="Plan de manejo / tratamiento">
          <textarea
            className={textareaClass}
            rows={3}
            value={form.plan_manejo}
            onChange={(e) => set("plan_manejo", e.target.value)}
            placeholder="Indicaciones, tratamiento no farmacológico, recomendaciones…"
          />
        </Field>
        <Field label="Procedimientos / remisiones indicados">
          <textarea
            className={textareaClass}
            rows={2}
            value={form.procedimientos_indicados}
            onChange={(e) => set("procedimientos_indicados", e.target.value)}
            placeholder="Procedimientos, interconsultas, exámenes solicitados…"
          />
        </Field>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-[1fr_200px]">
          <Field label="Observaciones del médico">
            <textarea
              className={textareaClass}
              rows={2}
              value={form.observaciones_medico}
              onChange={(e) => set("observaciones_medico", e.target.value)}
              placeholder="Notas adicionales…"
            />
          </Field>
          <Field label="Próxima cita">
            <Input
              type="date"
              value={form.proxima_cita}
              onChange={(e) => set("proxima_cita", e.target.value)}
            />
          </Field>
        </div>
      </Card>

      {/* Medicamentos recetados — solo UI (HU-06) */}
      <Card className="flex flex-col gap-5 p-6">
        <SectionTitle
          icon={Pill}
          title="Medicamentos recetados"
          hint="Se guardará con la fórmula médica (HU-06)"
        />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1fr_1fr_1fr_auto]">
          <Input
            value={medForm.nombre}
            onChange={(e) => setMedForm({ ...medForm, nombre: e.target.value })}
            placeholder="Medicamento"
          />
          <Input
            value={medForm.dosis}
            onChange={(e) => setMedForm({ ...medForm, dosis: e.target.value })}
            placeholder="Dosis (500 mg)"
          />
          <Input
            value={medForm.frecuencia}
            onChange={(e) =>
              setMedForm({ ...medForm, frecuencia: e.target.value })
            }
            placeholder="Frecuencia (c/8h)"
          />
          <Input
            value={medForm.duracion}
            onChange={(e) =>
              setMedForm({ ...medForm, duracion: e.target.value })
            }
            placeholder="Duración (7 días)"
          />
          <Button type="button" variant="outline" onClick={addMedicamento}>
            <Plus className="size-4" />
            Agregar
          </Button>
        </div>
        {medicamentos.length > 0 && (
          <div className="flex flex-col gap-2">
            {medicamentos.map((m, i) => (
              <div
                key={i}
                className="flex items-center justify-between rounded-[var(--radius)] border border-line bg-shell px-4 py-2 text-sm"
              >
                <span className="text-navy-800">
                  <span className="font-medium">{m.nombre}</span>
                  {m.dosis && ` · ${m.dosis}`}
                  {m.frecuencia && ` · ${m.frecuencia}`}
                  {m.duracion && ` · ${m.duracion}`}
                </span>
                <button
                  type="button"
                  onClick={() =>
                    setMedicamentos((list) => list.filter((_, j) => j !== i))
                  }
                  className="text-danger hover:opacity-70"
                >
                  <Trash2 className="size-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Adjuntar resultados — solo UI (HU-07) */}
      <Card className="flex flex-col gap-5 p-6">
        <SectionTitle
          icon={Paperclip}
          title="Resultados adjuntos"
          hint="El almacenamiento llega con la HU-07"
        />
        <label className="flex cursor-pointer flex-col items-center gap-2 rounded-[var(--radius)] border-2 border-dashed border-line bg-field p-8 text-center transition-colors hover:border-teal hover:bg-teal/5">
          <Upload className="size-8 text-muted" />
          <p className="text-sm font-medium text-ink">
            Arrastre o seleccione resultados (labs, imágenes, informes)
          </p>
          <p className="text-xs text-muted">PDF, JPG, PNG, DICOM</p>
          <input
            type="file"
            multiple
            className="sr-only"
            onChange={(e) =>
              setAdjuntos(Array.from(e.target.files ?? []).map((f) => f.name))
            }
          />
        </label>
        {adjuntos.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {adjuntos.map((name, i) => (
              <Badge key={i} tone="neutral">
                <Paperclip className="mr-1 inline size-3" />
                {name}
              </Badge>
            ))}
          </div>
        )}
      </Card>

      {/* Error + acciones */}
      {error && (
        <div className="flex items-center gap-2 rounded-[var(--radius)] border border-danger/20 bg-danger/5 p-3 text-sm text-danger">
          <AlertTriangle className="size-4 shrink-0" />
          {error}
        </div>
      )}

      <div className="flex justify-end gap-2 pb-4">
        <Button variant="outline" asChild>
          <Link href="/pacientes">Cancelar</Link>
        </Button>
        <Button type="submit" disabled={saving}>
          <Save className="size-4" />
          {saving ? "Guardando…" : "Finalizar consulta"}
        </Button>
      </div>
    </form>
  );
}

function SectionTitle({
  icon: Icon,
  title,
  hint,
}: {
  icon: React.ElementType;
  title: string;
  hint?: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="flex size-8 items-center justify-center rounded-lg bg-teal/10 text-teal">
        <Icon className="size-4" />
      </span>
      <h3 className="font-display text-lg font-semibold text-ink">{title}</h3>
      {hint && <span className="ml-auto text-xs text-muted">{hint}</span>}
    </div>
  );
}

export default function ConsultaPage() {
  return (
    <Suspense
      fallback={
        <Card className="p-10 text-center text-slate">Cargando…</Card>
      }
    >
      <ConsultaForm />
    </Suspense>
  );
}
