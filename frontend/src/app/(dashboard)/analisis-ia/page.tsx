"use client";

import { Suspense, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
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
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
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
  historia_clinica_id: string;
  estado_procesamiento: "enviado" | "completado" | "fallido";
  modelo_ia_utilizado: string;
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
function metricasAHallazgos(
  metricas?: Metricas
): Array<{ label: string; confidence: number; tone: "danger" | "warning" | "success" }> {
  if (!metricas?.metrics) return [];

  const m = metricas.metrics as Record<string, number>;
  const hallazgos: Array<{
    label: string;
    confidence: number;
    tone: "danger" | "warning" | "success";
  }> = [];

  // Segmentación de bazo: volume_ml
  if (m.volume_ml !== undefined) {
    const normalMax = 314; // ml referencia adulto
    const pct = Math.min(100, Math.round((m.volume_ml / normalMax) * 100));
    hallazgos.push({
      label: `Volumen esplénico: ${m.volume_ml.toFixed(1)} ml`,
      confidence: pct,
      tone: m.volume_ml > normalMax ? "danger" : "success",
    });
  }

  // Detección de nódulo pulmonar: lesion_count, max_diameter_mm
  if (m.lesion_count !== undefined) {
    hallazgos.push({
      label: `Nódulos detectados: ${m.lesion_count}`,
      confidence: m.lesion_count > 0 ? 90 : 100,
      tone: m.lesion_count > 0 ? "danger" : "success",
    });
  }
  if (m.max_diameter_mm !== undefined) {
    hallazgos.push({
      label: `Diámetro máximo: ${m.max_diameter_mm.toFixed(1)} mm`,
      confidence: Math.min(100, Math.round(m.max_diameter_mm)),
      tone: m.max_diameter_mm > 6 ? "warning" : "success",
    });
  }

  // Densidad mamaria: predicted_class (A-D), probability
  if (m.predicted_class !== undefined) {
    const pct = m.probability !== undefined ? Math.round((m.probability as number) * 100) : 80;
    hallazgos.push({
      label: `Densidad mamaria clase ${m.predicted_class}`,
      confidence: pct,
      tone: String(m.predicted_class) >= "C" ? "warning" : "success",
    });
  }

  // Tumor cerebral: cualquier métrica de volumen de segmentación
  if (m.tumor_volume_ml !== undefined) {
    hallazgos.push({
      label: `Volumen tumoral: ${(m.tumor_volume_ml as number).toFixed(1)} ml`,
      confidence: 85,
      tone: (m.tumor_volume_ml as number) > 0 ? "danger" : "success",
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

function SinID() {
  return (
    <div className="flex flex-col items-center gap-4 py-16 text-slate">
      <BrainCircuit className="size-10 text-muted" />
      <p className="font-semibold">Sin análisis seleccionado</p>
      <p className="text-sm text-muted">
        Accede a esta pantalla desde un examen médico para ver el resultado del
        análisis IA.
      </p>
      <Button asChild variant="ghost">
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

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Polling hasta que el estado_procesamiento salga de 'enviado'.
  useEffect(() => {
    if (!sugerenciaId) return;

    const fetchSugerencia = async () => {
      try {
        const res = await fetch(`${API}/sugerencias-ia/${sugerenciaId}`, {
          headers: authHeaders(),
        });
        if (!res.ok) {
          setError(`Error ${res.status} al obtener el análisis`);
          clearInterval(intervalRef.current!);
          return;
        }
        const data: Sugerencia = await res.json();
        setSugerencia(data);
        if (data.estado_procesamiento !== "enviado") {
          clearInterval(intervalRef.current!);
        }
        if (data.estado_revision === "revisada") {
          setAceptada(true);
        }
      } catch {
        setError("Error de red al obtener el análisis");
        clearInterval(intervalRef.current!);
      }
    };

    fetchSugerencia();
    intervalRef.current = setInterval(fetchSugerencia, 5000);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [sugerenciaId]);

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

  const hallazgos = sugerencia ? metricasAHallazgos(sugerencia.metricas) : [];

  if (!sugerenciaId) return <SinID />;

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

      {/* Estado: completado */}
      {!error && sugerencia?.estado_procesamiento === "completado" && (
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
