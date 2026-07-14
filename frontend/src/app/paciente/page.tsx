"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  LogOut,
  User,
  Stethoscope,
  CalendarPlus,
  CalendarClock,
  X,
  ShieldCheck,
} from "lucide-react";
import { Wordmark } from "@/components/brand";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";

type Medico = { id: string; nombre: string; especialidad: string };
type Autorizacion = { especialidad: string; especialistas: Medico[] };
type Cita = {
  id: string;
  medico_nombre: string;
  especialidad: string;
  fecha_hora: string;
  motivo: string | null;
  estado: string;
};
type Agenda = {
  medico_tratante: Medico | null;
  autorizaciones: Autorizacion[];
  citas: Cita[];
};
type Horario = { hora: string; disponible: boolean };
type PacienteDetalle = {
  nombre_paciente: string;
  apellidos_paciente: string;
  numero_documento: string;
  tipo_documento: string;
};

function formatDateTime(iso: string) {
  // El backend devuelve la fecha/hora ya en horario de Colombia (aunque el
  // timestamp venga marcado como UTC). NO convertimos de zona: leemos la hora
  // "de pared" tal cual, si no restaríamos 5 horas.
  const m = iso.match(/(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/);
  if (!m) return iso;
  const [y, mo, d, hh, mm] = [m[1], m[2], m[3], m[4], m[5]].map(Number);
  // Fecha construida con componentes locales (sin conversión de zona).
  return new Date(y, mo - 1, d, hh, mm).toLocaleString("es-CO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}


// Fecha de hoy en formato YYYY-MM-DD (para el valor y el mínimo del input date).
function todayForInput() {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

const estadoTone: Record<string, string> = {
  programada: "border-teal/30 bg-teal/10 text-teal",
  completada: "border-success/30 bg-success/10 text-success",
  cancelada: "border-line bg-field text-muted",
};

export default function PacienteHomePage() {
  const router = useRouter();
  const [paciente, setPaciente] = useState<PacienteDetalle | null>(null);
  const [agenda, setAgenda] = useState<Agenda | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Modal agendar
  const [target, setTarget] = useState<Medico | null>(null);
  const [fecha, setFecha] = useState("");
  const [hora, setHora] = useState("");
  const [motivo, setMotivo] = useState("");
  const [saving, setSaving] = useState(false);
  const [modalError, setModalError] = useState("");
  const [horarios, setHorarios] = useState<Horario[]>([]);
  const [loadingHorarios, setLoadingHorarios] = useState(false);

  const token = () =>
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  const loadAgenda = useCallback(async () => {
    const t = token();
    const res = await fetch("http://localhost:8080/api/v1/mi/agenda", {
      headers: t ? { Authorization: `Bearer ${t}` } : undefined,
    });
    if (res.ok) setAgenda(await res.json());
  }, []);

  // Carga las franjas horarias del día para el médico seleccionado, indicando
  // cuáles ya están ocupadas por otro paciente.
  const loadHorarios = useCallback(async (medicoId: string, fechaStr: string) => {
    setLoadingHorarios(true);
    setHorarios([]);
    try {
      const t = token();
      const res = await fetch(
        `http://localhost:8080/api/v1/citas/disponibilidad?medico_id=${medicoId}&fecha=${fechaStr}`,
        { headers: t ? { Authorization: `Bearer ${t}` } : undefined },
      );
      if (res.ok) {
        const data = await res.json();
        setHorarios(data.horarios ?? []);
      }
    } catch {
      // el select queda vacío; cambiar la fecha reintenta la carga
    } finally {
      setLoadingHorarios(false);
    }
  }, []);

  useEffect(() => {
    if (!target || !fecha) return;
    (async () => {
      await loadHorarios(target.id, fecha);
    })();
  }, [target, fecha, loadHorarios]);

  useEffect(() => {
    const t = token();
    if (!t) {
      router.push("/login");
      return;
    }
    (async () => {
      try {
        const res = await fetch("http://localhost:8080/api/v1/pacientes/me", {
          headers: { Authorization: `Bearer ${t}` },
        });
        if (res.ok) setPaciente(await res.json());
        else setError("No se pudieron cargar sus datos.");
        await loadAgenda();
      } catch {
        setError("Error de conexión con el servidor");
      } finally {
        setLoading(false);
      }
    })();
  }, [router, loadAgenda]);

  function handleLogout() {
    localStorage.removeItem("token");
    localStorage.removeItem("usuario");
    document.cookie = "token=; path=/; max-age=0";
    router.push("/login");
  }

  function openAgendar(m: Medico) {
    setTarget(m);
    setFecha(todayForInput());
    setHora("");
    setMotivo("");
    setModalError("");
  }

  async function submitAgendar(e: React.FormEvent) {
    e.preventDefault();
    if (!target) return;
    if (!fecha || !hora) {
      setModalError("Selecciona fecha y hora de la cita.");
      return;
    }
    setSaving(true);
    setModalError("");
    try {
      const t = token();
      const res = await fetch("http://localhost:8080/api/v1/citas", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(t ? { Authorization: `Bearer ${t}` } : {}),
        },
        body: JSON.stringify({
          medico_id: target.id,
          fecha_hora: `${fecha}T${hora}`,
          motivo: motivo || undefined,
        }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setModalError(data.error ?? "No se pudo agendar la cita");
        if (res.status === 409) {
          // El horario pudo haber sido tomado por otro paciente justo ahora:
          // refrescamos la lista para que ya no aparezca disponible.
          setHora("");
          loadHorarios(target.id, fecha);
        }
        return;
      }
      setTarget(null);
      loadAgenda();
    } catch {
      setModalError("Error de conexión con el servidor");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex min-h-full flex-col items-center bg-canvas px-4 py-10">
      <div className="flex w-full max-w-lg flex-col gap-6">
        <div className="flex items-center justify-between">
          <Wordmark />
          <Button variant="outline" size="sm" onClick={handleLogout}>
            <LogOut className="size-4" />
            Cerrar Sesión
          </Button>
        </div>

        {loading && (
          <Card className="p-8 text-center text-sm text-slate">Cargando…</Card>
        )}
        {error && (
          <Card className="p-6 text-center text-sm text-danger">{error}</Card>
        )}

        {paciente && (
          <Card className="flex items-center gap-4 p-6">
            <div className="flex size-14 shrink-0 items-center justify-center rounded-full bg-[#91b9cf]/20 text-teal">
              <User className="size-7" />
            </div>
            <div>
              <h1 className="font-display text-lg font-semibold text-ink">
                {paciente.nombre_paciente} {paciente.apellidos_paciente}
              </h1>
              <p className="font-mono text-sm text-muted">
                {paciente.tipo_documento} {paciente.numero_documento}
              </p>
            </div>
          </Card>
        )}

        {/* Médico general */}
        {agenda?.medico_tratante && (
          <Card className="flex flex-col gap-3 p-6">
            <h2 className="font-display font-semibold text-ink">
              Mi Médico general
            </h2>
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="flex size-10 items-center justify-center rounded-lg bg-teal/10 text-teal">
                  <Stethoscope className="size-5" />
                </div>
                <div>
                  <p className="text-sm font-medium text-ink">
                    {agenda.medico_tratante.nombre}
                  </p>
                  <p className="text-xs text-muted">
                    {agenda.medico_tratante.especialidad}
                  </p>
                </div>
              </div>
              <Button size="sm" onClick={() => openAgendar(agenda.medico_tratante!)}>
                <CalendarPlus className="size-4" />
                Agendar cita
              </Button>
            </div>
          </Card>
        )}

        {/* Especialistas autorizados */}
        {agenda && agenda.autorizaciones.length > 0 && (
          <Card className="flex flex-col gap-3 p-6">
            <h2 className="flex items-center gap-2 font-display font-semibold text-ink">
              <ShieldCheck className="size-4 text-teal" />
              Especialistas autorizados
            </h2>
            <p className="text-xs text-muted">
              Tu médico general autorizó estas especialidades. Agenda cuando
              quieras.
            </p>
            {agenda.autorizaciones.map((a) => (
              <div key={a.especialidad} className="flex flex-col gap-2">
                <p className="text-xs uppercase tracking-[0.6px] text-label">
                  {a.especialidad}
                </p>
                {a.especialistas.map((m) => (
                  <div
                    key={m.id}
                    className="flex items-center justify-between gap-3 rounded-[var(--radius)] border border-line bg-shell px-3 py-2"
                  >
                    <span className="text-sm text-navy-800">{m.nombre}</span>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => openAgendar(m)}
                    >
                      <CalendarPlus className="size-4" />
                      Agendar
                    </Button>
                  </div>
                ))}
              </div>
            ))}
          </Card>
        )}

        {/* Mis citas */}
        {agenda && (
          <Card className="flex flex-col gap-3 p-6">
            <h2 className="flex items-center gap-2 font-display font-semibold text-ink">
              <CalendarClock className="size-4 text-teal" />
              Mis citas
            </h2>
            {agenda.citas.length === 0 ? (
              <p className="text-sm text-slate">Aún no tienes citas agendadas.</p>
            ) : (
              agenda.citas.map((c) => (
                <div
                  key={c.id}
                  className="flex items-center justify-between gap-3 border-t border-line pt-3 first:border-0 first:pt-0"
                >
                  <div>
                    <p className="text-sm font-medium text-ink">
                      {c.medico_nombre}{" "}
                      <span className="text-xs font-normal text-muted">
                        · {c.especialidad}
                      </span>
                    </p>
                    <p className="text-xs text-muted">
                      {formatDateTime(c.fecha_hora)}
                      {c.motivo ? ` · ${c.motivo}` : ""}
                    </p>
                  </div>
                  <span
                    className={`rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize ${
                      estadoTone[c.estado] ?? estadoTone.cancelada
                    }`}
                  >
                    {c.estado}
                  </span>
                </div>
              ))
            )}
          </Card>
        )}
      </div>

      {/* Modal agendar */}
      {target && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-sm flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">
                Agendar cita
              </h3>
              <button
                type="button"
                onClick={() => setTarget(null)}
                className="text-muted hover:text-ink"
              >
                <X className="size-4" />
              </button>
            </div>
            <p className="text-sm text-slate">
              Con <span className="font-medium text-ink">{target.nombre}</span> ·{" "}
              {target.especialidad}
            </p>
            <form onSubmit={submitAgendar} className="flex flex-col gap-4">
              <Field label="Fecha">
                <Input
                  type="date"
                  required
                  min={todayForInput()}
                  value={fecha}
                  onChange={(e) => {
                    setFecha(e.target.value);
                    setHora("");
                  }}
                />
              </Field>
              <Field label="Hora" hint={<span className="text-xs text-muted">6:00 a 19:30</span>}>
                <select
                  required
                  value={hora}
                  onChange={(e) => setHora(e.target.value)}
                  disabled={loadingHorarios}
                  className="h-11 w-full rounded-[var(--radius)] border border-line bg-field px-4 text-base text-navy-800 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
                >
                  <option value="" disabled>
                    {loadingHorarios
                      ? "Cargando horarios…"
                      : horarios.some((h) => h.disponible)
                        ? "Selecciona una hora…"
                        : "No hay horarios disponibles este día"}
                  </option>
                  {horarios
                    .filter((h) => h.disponible)
                    .map((h) => (
                      <option key={h.hora} value={h.hora}>
                        {h.hora}
                      </option>
                    ))}
                </select>
              </Field>
              <Field label="Motivo (opcional)">
                <Input
                  value={motivo}
                  onChange={(e) => setMotivo(e.target.value)}
                  placeholder="Motivo de la consulta…"
                />
              </Field>
              {modalError && (
                <p className="text-sm text-danger">{modalError}</p>
              )}
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setTarget(null)}
                >
                  Cancelar
                </Button>
                <Button type="submit" disabled={saving}>
                  {saving ? "Agendando…" : "Agendar cita"}
                </Button>
              </div>
            </form>
          </Card>
        </div>
      )}
    </div>
  );
}