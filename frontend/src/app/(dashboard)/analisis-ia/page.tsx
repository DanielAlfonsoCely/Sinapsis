"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import {
  BrainCircuit,
  AlertTriangle,
  CheckCircle,
  ChevronLeft,
  BarChart2,
  Info,
  ShieldAlert,
  Clock,
  Loader2,
  XCircle,
  Send,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

// ---------------------------------------------------------------------------
// Tipos — espejo de SugerenciaIAResponse del backend
// ---------------------------------------------------------------------------

interface Metricas {
  metrics?: Record<string, unknown>;
  artifacts?: Array<{ type: string; uri: string }>;
}

interface Sugerencia {
  id: string;
  examinagen_id: string;
  consulta_id?: string;
  historia_clinica_id: string;
  estado_procesamiento: "enviado" | "completado" | "fallido";
  modelo_ia_utilizado: string;
  // RF-12/RN-007: sin pre-diagnóstico registrado en la consulta, el backend
  // oculta confianza_prediccion/descripcion_hallazgo/diagnostico_sugerido/metricas.
  pre_diagnostico_registrado: boolean;
  confianza_prediccion?: number;
  descripcion_hallazgo?: string;
  diagnostico_sugerido?: string;
  metricas?: Metricas;
  fecha_analisis?: string;
  estado_revision: "pendiente" | "revisada" | "rechazada";
  observaciones_medico?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const API = "http://localhost:8080/api/v1";

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

function authHeaders(): HeadersInit {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// Mapea las métricas numéricas del modelo a hallazgos visualizables.
//
// Campos reales por tipo de bundle (microservice/.../output_extractors.py):
//   - Segmentación (bazo, tumor cerebral) -> SegmentationExtractor: { volume_voxels }
//   - Detección de nódulo pulmonar        -> DetectionExtractor:    { lesion_count, max_diameter_mm }
//   - Clasificación (densidad mamaria)    -> BreastDensityCSVExtractor: { predicted_class, probability }
function metricasAHallazgos(
  metricas?: Metricas,
  modelo?: string
): Array<{ label: string; confidence: number; tone: "danger" | "warning" | "success" }> {
  if (!metricas?.metrics) return [];

  const m = metricas.metrics as Record<string, number | string>;
  const hallazgos: Array<{
    label: string;
    confidence: number;
    tone: "danger" | "warning" | "success";
  }> = [];

  // Segmentación (bazo o tumor cerebral, según el bundle usado): volume_voxels.
  if (m.volume_voxels !== undefined) {
    const voxels = Number(m.volume_voxels);
    const esBazo = (modelo || "").includes("spleen");
    const label = esBazo ? "Volumen segmentado (bazo)" : "Volumen segmentado";
    hallazgos.push({
      label: `${label}: ${voxels.toLocaleString("es-CO")} vóxeles`,
      confidence: voxels > 0 ? 90 : 100,
      tone: voxels > 0 ? (esBazo ? "success" : "danger") : "success",
    });
  }

  // Detección de nódulo pulmonar: lesion_count, max_diameter_mm
  if (m.lesion_count !== undefined) {
    const lesionCount = Number(m.lesion_count);
    hallazgos.push({
      label: `Nódulos detectados: ${lesionCount}`,
      confidence: lesionCount > 0 ? 90 : 100,
      tone: lesionCount > 0 ? "danger" : "success",
    });
  }
  if (m.max_diameter_mm !== undefined) {
    const maxDiameter = Number(m.max_diameter_mm);
    hallazgos.push({
      label: `Diámetro máximo: ${maxDiameter.toFixed(1)} mm`,
      confidence: Math.min(100, Math.round(maxDiameter)),
      tone: maxDiameter > 6 ? "warning" : "success",
    });
  }

  // Clasificación (densidad mamaria): predicted_class (A-D), probability
  if (m.predicted_class !== undefined) {
    const pct = m.probability !== undefined ? Math.round(Number(m.probability) * 100) : 80;
    hallazgos.push({
      label: `Densidad mamaria clase ${m.predicted_class}`,
      confidence: pct,
      tone: String(m.predicted_class) >= "C" ? "warning" : "success",
    });
  }

  return hallazgos;
}

// ---------------------------------------------------------------------------
// Componentes de estado
// ---------------------------------------------------------------------------

function EstadoProcesando() {
  return (
    <div className="flex flex-col items-center gap-4 py-16 text-slate">
      <Loader2 className="size-10 animate-spin text-teal" />
      <p className="font-semibold">Análisis en curso...</p>
      <p className="text-sm text-muted">
        El microservicio MONAI está procesando la imagen. Esto puede tardar hasta
        varios minutos según el modelo.
      </p>
    </div>
  );
}

function EstadoFallido({ mensaje }: { mensaje?: string }) {
  return (
    <div className="flex flex-col items-center gap-4 py-16 text-danger">
      <XCircle className="size-10" />
      <p className="font-semibold">El análisis falló</p>
      {mensaje && <p className="text-sm text-muted">{mensaje}</p>}
    </div>
  );
}

// RF-12/RN-007: gate de pre-diagnóstico. El médico debe registrar su propia
// impresión clínica inicial en la consulta ANTES de solicitar o visualizar
// hallazgos de análisis IA (evita que la sugerencia del modelo sesgue el
// juicio clínico inicial). Se usa tanto al solicitar como al ver resultados.
function PreDiagnosticoGate({
  consultaId,
  onRegistered,
}: {
  consultaId: string;
  onRegistered: () => void;
}) {
  const [texto, setTexto] = useState("");
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRegistrar = useCallback(async () => {
    if (!texto.trim() || enviando) return;
    setEnviando(true);
    setError(null);
    try {
      const res = await fetch(`${API}/consultas/${consultaId}/pre-diagnostico`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ pre_diagnostico: texto.trim() }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || `Error ${res.status}`);
        return;
      }
      onRegistered();
    } catch {
      setError("Error de red al registrar el pre-diagnóstico");
    } finally {
      setEnviando(false);
    }
  }, [texto, enviando, consultaId, onRegistered]);

  return (
    <Card className="flex flex-col gap-3 border-warning/30 bg-warning/5 p-5">
      <div className="flex items-center gap-2">
        <ShieldAlert className="size-5 text-warning" />
        <h3 className="font-display font-semibold text-ink">
          Pre-diagnóstico requerido (RF-12)
        </h3>
      </div>
      <p className="text-sm text-slate">
        Debes registrar tu impresión clínica inicial{" "}
        <strong>antes de ver las sugerencias de IA</strong>, para que el
        modelo no sesgue tu juicio clínico. Esta consulta aún no tiene un
        pre-diagnóstico registrado.
      </p>
      <textarea
        className="w-full resize-none rounded-[var(--radius)] border border-line bg-field px-4 py-3 text-sm text-ink placeholder:text-muted outline-none focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
        rows={3}
        value={texto}
        onChange={(e) => setTexto(e.target.value)}
        placeholder="Impresión diagnóstica inicial, antes de ver sugerencias de IA…"
      />
      {error && (
        <div className="rounded-[var(--radius)] border border-danger/30 bg-danger/8 p-3 text-sm text-danger">
          {error}
        </div>
      )}
      <Button onClick={handleRegistrar} disabled={enviando || !texto.trim()}>
        {enviando ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <CheckCircle className="size-4" />
        )}
        Registrar pre-diagnóstico y continuar
      </Button>
    </Card>
  );
}

const ANALYSIS_TYPES = [
  { value: "ct_spleen_segmentation", label: "Segmentación de bazo (TC)" },
  { value: "ct_lung_nodule_detection", label: "Detección de nódulo pulmonar (TC)" },
  { value: "mri_brain_tumor_segmentation", label: "Segmentación de tumor cerebral (RM)" },
  { value: "xr_breast_density_classification", label: "Clasificación de densidad mamaria (RX)" },
];

function SolicitarAnalisis() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const examinagenIdParam = searchParams.get("examinagenId") || "";
  const [examinagenId, setExaminagenId] = useState(examinagenIdParam);
  const [analysisType, setAnalysisType] = useState(ANALYSIS_TYPES[0].value);
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // RF-12/RN-007: si el backend rechaza la solicitud por falta de
  // pre-diagnóstico, guardamos el consulta_id que devuelve para ofrecer
  // registrarlo aquí mismo sin salir de la página.
  const [consultaIdSinPreDiagnostico, setConsultaIdSinPreDiagnostico] =
    useState<string | null>(null);

  // El componente puede quedar montado al navegar entre exámenes de distintos
  // pacientes (misma ruta /analisis-ia, solo cambia el query param). useState
  // solo toma el valor inicial una vez, así que sin este efecto el campo se
  // queda con el examen anterior y el análisis se dispara sobre el examen
  // equivocado. Sincronizamos explícitamente cada vez que cambia el param.
  useEffect(() => {
    setExaminagenId(examinagenIdParam);
    setError(null);
    setConsultaIdSinPreDiagnostico(null);
  }, [examinagenIdParam]);

  const handleSubmit = useCallback(async () => {
    if (!examinagenId.trim() || enviando) return;
    setEnviando(true);
    setError(null);
    setConsultaIdSinPreDiagnostico(null);
    try {
      const res = await fetch(`${API}/examenes/${examinagenId.trim()}/analisis-ia`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ analysis_type: analysisType }),
      });
      const data = await res.json();
      if (!res.ok) {
        if (data.code === "pre_diagnostico_required" && data.consulta_id) {
          setConsultaIdSinPreDiagnostico(data.consulta_id);
        } else {
          setError(data.error || `Error ${res.status}`);
        }
        return;
      }
      router.push(`/analisis-ia?id=${data.id}`);
    } catch {
      setError("Error de red al solicitar el análisis");
    } finally {
      setEnviando(false);
    }
  }, [examinagenId, analysisType, enviando, router]);

  return (
    <div className="mx-auto flex max-w-lg flex-col gap-6 py-12">
      <div className="flex items-center gap-2">
        <BrainCircuit className="size-6 text-teal" />
        <h2 className="font-display text-2xl font-semibold text-ink">
          Solicitar análisis IA
        </h2>
      </div>

      <p className="text-sm text-slate">
        Ingresa el ID del examen (examinagen) que ya tiene una imagen cargada y
        selecciona el tipo de análisis a ejecutar con MONAI.
      </p>

      <Card className="flex flex-col gap-4 p-5">
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-ink">ID del examen</label>
          <Input
            placeholder="Ej: a1b2c3d4-e5f6-7890-abcd-ef1234567890"
            value={examinagenId}
            onChange={(e) => setExaminagenId(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-ink">Tipo de análisis</label>
          <select
            className="flex h-10 w-full rounded-[var(--radius)] border border-line bg-field px-3 py-2 text-sm text-ink outline-none focus:border-teal"
            value={analysisType}
            onChange={(e) => setAnalysisType(e.target.value)}
          >
            {ANALYSIS_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>

        {error && (
          <div className="rounded-[var(--radius)] border border-danger/30 bg-danger/8 p-3 text-sm text-danger">
            {error}
          </div>
        )}

        <Button onClick={handleSubmit} disabled={enviando || !examinagenId.trim()}>
          {enviando ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Send className="size-4" />
          )}
          Solicitar análisis
        </Button>
      </Card>

      {consultaIdSinPreDiagnostico && (
        <PreDiagnosticoGate
          consultaId={consultaIdSinPreDiagnostico}
          onRegistered={() => {
            setConsultaIdSinPreDiagnostico(null);
            handleSubmit();
          }}
        />
      )}

      <Button asChild variant="ghost" className="self-start">
        <Link href="/consulta">
          <ChevronLeft className="size-4" />
          Volver a consulta
        </Link>
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Página principal
// ---------------------------------------------------------------------------

function AnalisisIAContent() {
  const searchParams = useSearchParams();
  const sugerenciaId = searchParams.get("id");

  const [sugerencia, setSugerencia] = useState<Sugerencia | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [revisando, setRevisando] = useState(false);
  const [aceptada, setAceptada] = useState(false);
  // Incrementar fuerza un refetch inmediato fuera del intervalo de polling
  // (p.ej. tras registrar el pre-diagnóstico requerido por RF-12).
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Polling hasta que el estado_procesamiento salga de 'enviado'.
  //
  // Importante: esta página puede quedar montada al navegar entre exámenes de
  // distintos pacientes (misma ruta /analisis-ia, solo cambia ?id=). Si no
  // reseteamos el estado local al cambiar sugerenciaId, la UI sigue mostrando
  // datos/estado del examen anterior (p.ej. "aceptada") mientras llega la
  // respuesta del nuevo — pareciendo que es "el mismo examen".
  useEffect(() => {
    setSugerencia(null);
    setError(null);
    setAceptada(false);

    if (!sugerenciaId) return;

    let cancelado = false;

    const fetchSugerencia = async () => {
      try {
        const res = await fetch(`${API}/sugerencias-ia/${sugerenciaId}`, {
          headers: authHeaders(),
        });
        if (cancelado) return;
        if (!res.ok) {
          setError(`Error ${res.status} al obtener el análisis`);
          clearInterval(intervalRef.current!);
          return;
        }
        const data: Sugerencia = await res.json();
        if (cancelado) return;
        setSugerencia(data);
        if (data.estado_procesamiento !== "enviado") {
          clearInterval(intervalRef.current!);
        }
        // Refleja el estado real del backend, no solo cuando es "revisada":
        // evita arrastrar el "aceptada=true" de un examen previamente revisado.
        setAceptada(data.estado_revision === "revisada");
      } catch {
        if (cancelado) return;
        setError("Error de red al obtener el análisis");
        clearInterval(intervalRef.current!);
      }
    };

    fetchSugerencia();
    intervalRef.current = setInterval(fetchSugerencia, 5000);

    return () => {
      cancelado = true;
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [sugerenciaId, refetchTrigger]);

  const handleAceptar = async () => {
    if (!sugerenciaId || revisando) return;
    setRevisando(true);
    try {
      const res = await fetch(`${API}/sugerencias-ia/${sugerenciaId}/revision`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ estado_revision: "revisada" }),
      });
      if (res.ok) {
        setAceptada(true);
        setSugerencia((prev) =>
          prev ? { ...prev, estado_revision: "revisada" } : prev
        );
      }
    } finally {
      setRevisando(false);
    }
  };

  const hallazgos = sugerencia
    ? metricasAHallazgos(sugerencia.metricas, sugerencia.modelo_ia_utilizado)
    : [];

  if (!sugerenciaId) return <SolicitarAnalisis />;

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
              Asistido por MONAI ·{" "}
              {sugerencia?.modelo_ia_utilizado || "cargando modelo..."}
            </p>
          </div>
        </div>
        <Badge tone="navy">
          <BrainCircuit className="size-3" />
          MONAI
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
            Este análisis es una{" "}
            <strong>sugerencia de apoyo diagnóstico</strong> generada por
            inteligencia artificial y{" "}
            <strong>no constituye un diagnóstico clínico</strong>. El médico
            tratante conserva la responsabilidad clínica y legal sobre el
            diagnóstico y el tratamiento del paciente. (Resolución 1995/1999 —
            Ley 1581/2012)
          </p>
        </div>
      </div>

      {/* Error de red */}
      {error && (
        <div className="rounded-[var(--radius)] border border-danger/30 bg-danger/8 p-4 text-sm text-danger">
          {error}
        </div>
      )}

      {/* Estado: procesando */}
      {!error && sugerencia?.estado_procesamiento === "enviado" && (
        <EstadoProcesando />
      )}

      {/* Estado: fallido */}
      {!error && sugerencia?.estado_procesamiento === "fallido" && (
        <EstadoFallido />
      )}

      {/* Estado: completado, pero falta pre-diagnóstico (RF-12/RN-007) */}
      {!error &&
        sugerencia?.estado_procesamiento === "completado" &&
        !sugerencia.pre_diagnostico_registrado &&
        sugerencia.consulta_id && (
          <PreDiagnosticoGate
            consultaId={sugerencia.consulta_id}
            onRegistered={() => setRefetchTrigger((n) => n + 1)}
          />
        )}

      {/* Estado: completado y con pre-diagnóstico registrado -> se ven los hallazgos */}
      {!error &&
        sugerencia?.estado_procesamiento === "completado" &&
        sugerencia.pre_diagnostico_registrado && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_380px]">
          {/* Panel info examen */}
          <Card className="flex flex-col gap-4 p-5">
            <div className="flex items-center justify-between">
              <h3 className="font-display font-semibold text-ink">
                Examen analizado
              </h3>
              {sugerencia.fecha_analisis && (
                <span className="flex items-center gap-1 text-xs text-muted">
                  <Clock className="size-3" />
                  {new Date(sugerencia.fecha_analisis).toLocaleString("es-CO")}
                </span>
              )}
            </div>
            <div className="flex flex-col gap-1 text-sm text-slate">
              <span>
                Examen ID:{" "}
                <span className="font-mono text-ink text-xs">
                  {sugerencia.examinagen_id}
                </span>
              </span>
              <span>
                Modelo:{" "}
                <span className="text-ink">{sugerencia.modelo_ia_utilizado}</span>
              </span>
              {sugerencia.confianza_prediccion !== undefined && (
                <span>
                  Confianza general:{" "}
                  <strong className="text-ink">
                    {sugerencia.confianza_prediccion}%
                  </strong>
                </span>
              )}
            </div>
          </Card>

          {/* Panel resultados */}
          <div className="flex flex-col gap-4">
            {/* Hallazgos */}
            {hallazgos.length > 0 && (
              <Card className="flex flex-col gap-4 p-5">
                <div className="flex items-center gap-2">
                  <BarChart2 className="size-5 text-teal" />
                  <h3 className="font-display font-semibold text-ink">
                    Hallazgos del modelo
                  </h3>
                </div>
                <div className="flex flex-col gap-3">
                  {hallazgos.map((f) => (
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
                              : "text-success"
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
                              : "bg-success"
                          )}
                          style={{ width: `${f.confidence}%` }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            )}

            {/* Sugerencia diagnóstica */}
            {(sugerencia.descripcion_hallazgo ||
              sugerencia.diagnostico_sugerido) && (
              <Card className="flex flex-col gap-3 p-5">
                <div className="flex items-center gap-2">
                  <Info className="size-5 text-teal" />
                  <h3 className="font-display font-semibold text-ink">
                    Sugerencia diagnóstica
                  </h3>
                </div>
                {sugerencia.descripcion_hallazgo && (
                  <p className="text-sm text-slate leading-relaxed">
                    {sugerencia.descripcion_hallazgo}
                  </p>
                )}
                {sugerencia.diagnostico_sugerido && (
                  <p className="text-sm font-semibold text-ink">
                    {sugerencia.diagnostico_sugerido}
                  </p>
                )}
              </Card>
            )}

            {/* Aviso cuando el modelo no devuelve texto diagnóstico */}
            {!sugerencia.descripcion_hallazgo &&
              !sugerencia.diagnostico_sugerido &&
              hallazgos.length === 0 && (
                <Card className="p-5 text-sm text-muted flex items-center gap-2">
                  <AlertTriangle className="size-4 text-warning" />
                  El modelo no devolvió métricas textuales. Revisa las métricas
                  crudas en historia clínica.
                </Card>
              )}

            {/* Acción de revisión */}
            {!aceptada ? (
              <Button
                onClick={handleAceptar}
                disabled={revisando}
                className="w-full"
              >
                {revisando ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <CheckCircle className="size-4" />
                )}
                Usar esta sugerencia en el diagnóstico
              </Button>
            ) : (
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2 rounded-[var(--radius)] border border-success/30 bg-success/8 p-3 text-sm text-success">
                  <CheckCircle className="size-4 shrink-0" />
                  Sugerencia incorporada. Puede ajustarla en el paso de
                  diagnóstico.
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
      )}
    </div>
  );
}

export default function AnalisisIAPage() {
  return (
    <Suspense>
      <AnalisisIAContent />
    </Suspense>
  );
}
