"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import {
  LogOut,
  User,
  Stethoscope,
  CalendarPlus,
  CalendarClock,
  X,
  ShieldCheck,
  CalendarDays,
  Bot,
} from "lucide-react";
import { Wordmark, BrandMark } from "@/components/brand";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

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
  const m = iso.match(/(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/);
  if (!m) return iso;
  const [y, mo, d, hh, mm] = [m[1], m[2], m[3], m[4], m[5]].map(Number);
  return new Date(y, mo - 1, d, hh, mm).toLocaleString("es-CO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

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

const NAV = [
  { href: "/paciente", label: "Mi agenda", icon: CalendarDays },
  { href: "/paciente/chat", label: "Asistente IA", icon: Bot },
];

function PacienteSidebar({ nombre }: { nombre: string }) {
  const pathname = usePathname();
  const router = useRouter();

  function handleLogout() {
    localStorage.removeItem("token");
    localStorage.removeItem("usuario");
    document.cookie = "token=; path=/; max-age=0";
    router.push("/login");
  }

  return (
    <aside className="fixed inset-y-0 left-0 z-30 flex w-60 flex-col justify-between bg-gradient-to-b from-[#001e4b] to-navy">
      <div className="flex flex-col gap-6 p-4">
        {/* Marca */}
        <div className="flex items-center gap-2.5 px-2 py-4">
          <BrandMark className="size-7 text-[#76aecc]" />
          <div className="leading-none">
            <p className="font-display text-xl font-bold tracking-tight text-white">
              SINAPSIS
            </p>
            <p className="mt-1 text-[10px] uppercase tracking-[1px] text-[#91b9cf]">
              Portal Paciente
            </p>
          </div>
        </div>

        {/* Avatar paciente */}
        {nombre && (
          <div className="flex items-center gap-3 rounded-lg bg-white/5 px-3 py-2">
            <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[#91b9cf]/20 text-[#76aecc]">
              <User className="size-4" />
            </div>
            <p className="truncate text-sm font-medium text-white">{nombre}</p>
          </div>
        )}

        {/* Navegación */}
        <nav className="flex flex-col gap-1">
          {NAV.map(({ href, label, icon: Icon }) => {
            const active = pathname === href;
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  "flex items-center gap-3 rounded px-4 py-3 text-sm transition-colors",
                  active
                    ? "border-l-4 border-[#76aecc] bg-[#76aecc]/10 pl-5 font-medium text-white"
                    : "text-[#c3c6d0] hover:bg-white/5 hover:text-white",
                )}
              >
                <Icon className="size-5 shrink-0" />
                {label}
              </Link>
            );
          })}
        </nav>
      </div>

      {/* Cerrar sesión */}
      <div className="border-t border-white/10 p-4">
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded px-4 py-3 text-sm text-[#d9534f] transition-colors hover:bg-white/5"
        >
          <LogOut className="size-5" />
          Cerrar Sesión
        </button>
      </div>
    </aside>
  );
}

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
      // el select queda vacío
    } finally {
      setLoadingHorarios(false);
    }
  }, []);

  useEffect(() => {
    if (!target || !fecha) return;
    (async () => { await loadHorarios(target.id, fecha); })();
  }, [target, fecha, loadHorarios]);

  useEffect(() => {
    const t = token();
    if (!t) { router.push("/login"); return; }
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
    if (!fecha || !hora) { setModalError("Selecciona fecha y hora de la cita."); return; }
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
        if (res.status === 409) { setHora(""); loadHorarios(target.id, fecha); }
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

  const nombrePaciente = paciente
    ? `${paciente.nombre_paciente} ${paciente.apellidos_paciente}`
    : "";

  return (
    <div className="flex min-h-screen">
      <PacienteSidebar nombre={nombrePaciente} />

      {/* Contenido principal — margen izquierdo igual al ancho de la sidebar */}
      <main className="ml-60 flex flex-1 flex-col items-center bg-canvas px-4 py-10">
        <div className="flex w-full max-w-lg flex-col gap-6">

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
              <h2 className="font-display font-semibold text-ink">Mi Médico general</h2>
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <div className="flex size-10 items-center justify-center rounded-lg bg-teal/10 text-teal">
                    <Stethoscope className="size-5" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-ink">{agenda.medico_tratante.nombre}</p>
                    <p className="text-xs text-muted">{agenda.medico_tratante.especialidad}</p>
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
                Tu médico general autorizó estas especialidades. Agenda cuando quieras.
              </p>
              {agenda.autorizaciones.map((a) => (
                <div key={a.especialidad} className="flex flex-col gap-2">
                  <p className="text-xs uppercase tracking-[0.6px] text-label">{a.especialidad}</p>
                  {a.especialistas.map((m) => (
                    <div
                      key={m.id}
                      className="flex items-center justify-between gap-3 rounded-[var(--radius)] border border-line bg-shell px-3 py-2"
                    >
                      <span className="text-sm text-navy-800">{m.nombre}</span>
                      <Button size="sm" variant="outline" onClick={() => openAgendar(m)}>
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
                        <span className="text-xs font-normal text-muted">· {c.especialidad}</span>
                      </p>
                      <p className="text-xs text-muted">
                        {formatDateTime(c.fecha_hora)}
                        {c.motivo ? ` · ${c.motivo}` : ""}
                      </p>
                    </div>
                    <span className={`rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize ${estadoTone[c.estado] ?? estadoTone.cancelada}`}>
                      {c.estado}
                    </span>
                  </div>
                ))
              )}
            </Card>
          )}
        </div>
      </main>

      {/* Modal agendar */}
      {target && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-sm flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">Agendar cita</h3>
              <button type="button" onClick={() => setTarget(null)} className="text-muted hover:text-ink">
                <X className="size-4" />
              </button>
            </div>
            <p className="text-sm text-slate">
              Con <span className="font-medium text-ink">{target.nombre}</span> · {target.especialidad}
            </p>
            <form onSubmit={submitAgendar} className="flex flex-col gap-4">
              <Field label="Fecha">
                <Input
                  type="date"
                  required
                  min={todayForInput()}
                  value={fecha}
                  onChange={(e) => { setFecha(e.target.value); setHora(""); }}
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
                  {horarios.filter((h) => h.disponible).map((h) => (
                    <option key={h.hora} value={h.hora}>{h.hora}</option>
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
              {modalError && <p className="text-sm text-danger">{modalError}</p>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setTarget(null)}>Cancelar</Button>
                <Button type="submit" disabled={saving}>{saving ? "Agendando…" : "Agendar cita"}</Button>
              </div>
            </form>
          </Card>
        </div>
      )}
    </div>
  );
}