"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import {
  Search,
  UserPlus,
  FileText,
  Stethoscope,
  ShieldCheck,
  Pill,
  Phone,
  Calendar,
  X,
  CheckCircle,
  Copy,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";

const selectClass =
  "h-11 w-full rounded-[var(--radius)] border border-line bg-field px-4 text-sm text-navy-800 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20";

const EMPTY_FORM = {
  numero_documento: "",
  tipo_documento: "CC",
  nombre_paciente: "",
  apellidos_paciente: "",
  fecha_nacimiento: "",
  sexo: "",
  email: "",
  telefono: "",
  direccion: "",
  tipo_sangre: "",
  alergias: "",
  aseguradora: "",
};

type CredencialesGeneradas = {
  nombre: string;
  email: string;
  tempPassword: string;
};

type PacienteAPI = {
  id: string;
  numero_documento: string;
  tipo_documento: string;
  nombre_paciente: string;
  apellidos_paciente: string;
  telefono: string | null;
  email: string | null;
  ultima_consulta: string | null;
  proxima_cita: string | null;
  tiene_cita_hoy: boolean;
  es_tratante: boolean;
  estado: boolean;
};

function initials(nombre: string, apellidos: string) {
  return `${nombre[0] ?? ""}${apellidos[0] ?? ""}`.toUpperCase();
}

function formatDate(iso: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("es-CO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  });
}

export default function PacientesPage() {
  const [pacientes, setPacientes] = useState<PacienteAPI[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [showNew, setShowNew] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const [credenciales, setCredenciales] = useState<CredencialesGeneradas | null>(
    null,
  );
  const [copied, setCopied] = useState(false);
  const [form, setForm] = useState(EMPTY_FORM);

  // Autorizar especialidad (remisión)
  const [autPaciente, setAutPaciente] = useState<PacienteAPI | null>(null);
  const [especialidades, setEspecialidades] = useState<string[]>([]);
  const [especialidadSel, setEspecialidadSel] = useState("");
  const [autMotivo, setAutMotivo] = useState("");
  const [autError, setAutError] = useState("");
  const [autOk, setAutOk] = useState("");
  const [autSaving, setAutSaving] = useState(false);

  useEffect(() => {
    const timeout = setTimeout(() => {
      fetchPacientes(query);
    }, 300); // debounce simple para no pegarle al backend en cada tecla

    return () => clearTimeout(timeout);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  async function openAutorizar(p: PacienteAPI) {
    setAutPaciente(p);
    setEspecialidadSel("");
    setAutMotivo("");
    setAutError("");
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("http://localhost:8080/api/v1/especialidades", {
        headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      });
      if (res.ok) setEspecialidades((await res.json()).especialidades ?? []);
    } catch {
      // el modal mostrará la lista vacía
    }
  }

  async function submitAutorizar(e: React.FormEvent) {
    e.preventDefault();
    if (!autPaciente || !especialidadSel) return;
    setAutSaving(true);
    setAutError("");
    try {
      const token = localStorage.getItem("token");
      const res = await fetch(
        `http://localhost:8080/api/v1/pacientes/${autPaciente.id}/remisiones`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({
            especialidad: especialidadSel,
            motivo: autMotivo || undefined,
          }),
        },
      );
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setAutError(data.error ?? "No se pudo autorizar la especialidad");
        return;
      }
      const nombre = `${autPaciente.nombre_paciente} ${autPaciente.apellidos_paciente}`;
      setAutPaciente(null);
      setAutOk(
        `Autorizaste a ${nombre} para consultar ${especialidadSel}. Ahora el paciente puede agendar esa cita.`,
      );
    } catch {
      setAutError("Error de conexión con el servidor");
    } finally {
      setAutSaving(false);
    }
  }

  async function fetchPacientes(q: string) {
    setLoading(true);
    setError("");
    try {
      const token = localStorage.getItem("token");
      const url = new URL("http://localhost:8080/api/v1/pacientes");
      if (q) url.searchParams.set("q", q);

      const res = await fetch(url.toString(), {
        headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      });

      if (!res.ok) {
        setError("No se pudieron cargar los pacientes");
        return;
      }

      const data = await res.json();
      setPacientes(data.pacientes ?? []);
    } catch {
      setError("Error de conexión con el servidor");
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setCreateError("");
    try {
      const token = localStorage.getItem("token");
      const body: Record<string, string> = {
        numero_documento: form.numero_documento,
        tipo_documento: form.tipo_documento,
        nombre_paciente: form.nombre_paciente,
        apellidos_paciente: form.apellidos_paciente,
        fecha_nacimiento: form.fecha_nacimiento,
        email: form.email,
      };
      if (form.sexo) body.sexo = form.sexo;
      if (form.telefono) body.telefono = form.telefono;
      if (form.direccion) body.direccion = form.direccion;
      if (form.tipo_sangre) body.tipo_sangre = form.tipo_sangre;
      if (form.alergias) body.alergias = form.alergias;
      if (form.aseguradora) body.aseguradora = form.aseguradora;

      const res = await fetch("http://localhost:8080/api/v1/pacientes", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(body),
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setCreateError(data.error ?? "No se pudo registrar el paciente");
        return;
      }

      setForm(EMPTY_FORM);
      setShowNew(false);
      setCredenciales({
        nombre: `${data.nombre_paciente} ${data.apellidos_paciente}`,
        email: data.email,
        tempPassword: data.temp_password,
      });
      fetchPacientes(query);
    } catch {
      setCreateError("Error de conexión con el servidor");
    } finally {
      setCreating(false);
    }
  }

  async function copyPassword() {
    if (!credenciales) return;
    await navigator.clipboard.writeText(credenciales.tempPassword);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink">
            Pacientes
          </h2>
          <p className="text-sm text-slate">
            {pacientes.length} pacientes registrados
          </p>
        </div>
        <Button size="sm" onClick={() => setShowNew((v) => !v)}>
          <UserPlus className="size-4" />
          Nuevo Paciente
        </Button>
      </div>

      {/* Modal de confirmación con credenciales generadas */}
      {credenciales && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-md flex-col gap-4 p-6">
            <div className="flex flex-col items-center gap-2 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-success/10 text-success">
                <CheckCircle className="size-6" />
              </div>
              <h3 className="font-display text-lg font-semibold text-ink">
                Paciente registrado
              </h3>
              <p className="text-sm text-slate">
                {credenciales.nombre} ya puede iniciar sesión con estas
                credenciales temporales.
              </p>
            </div>

            <div className="flex flex-col gap-3 rounded-[var(--radius)] border border-line bg-shell p-4">
              <div>
                <p className="text-xs uppercase tracking-[0.6px] text-label">
                  Correo
                </p>
                <p className="text-sm text-navy-800">{credenciales.email}</p>
              </div>
              <div>
                <p className="text-xs uppercase tracking-[0.6px] text-label">
                  Contraseña temporal
                </p>
                <div className="mt-1 flex items-center justify-between gap-2 rounded border border-line bg-white px-3 py-2">
                  <span className="font-mono text-sm text-ink">
                    {credenciales.tempPassword}
                  </span>
                  <button
                    type="button"
                    onClick={copyPassword}
                    className="flex items-center gap-1 text-xs text-teal hover:text-teal-700"
                  >
                    <Copy className="size-3.5" />
                    {copied ? "¡Copiado!" : "Copiar"}
                  </button>
                </div>
              </div>
            </div>

            <p className="text-xs text-muted">
              Esto es temporal mientras no haya envío real de correo: en
              producción, esta contraseña solo llegaría al correo del
              paciente.
            </p>

            <Button onClick={() => setCredenciales(null)}>Cerrar</Button>
          </Card>
        </div>
      )}

      {autOk && (
        <div className="rounded-[var(--radius)] border border-success/30 bg-success/8 px-4 py-3 text-sm text-success">
          {autOk}
        </div>
      )}

      {/* Modal: autorizar especialidad (remisión) */}
      {autPaciente && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-md flex-col gap-4 p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold text-ink">
                Autorizar especialista
              </h3>
              <button
                type="button"
                onClick={() => setAutPaciente(null)}
                className="text-muted hover:text-ink"
              >
                <X className="size-4" />
              </button>
            </div>
            <p className="text-sm text-slate">
              Autoriza a{" "}
              <span className="font-medium text-ink">
                {autPaciente.nombre_paciente} {autPaciente.apellidos_paciente}
              </span>{" "}
              a consultar una especialidad. El paciente agenda esa cita cuando
              quiera. Sigues siendo su médico a cargo.
            </p>
            <form onSubmit={submitAutorizar} className="flex flex-col gap-4">
              <Field label="Especialidad">
                <select
                  className={selectClass}
                  required
                  value={especialidadSel}
                  onChange={(e) => setEspecialidadSel(e.target.value)}
                >
                  <option value="">Seleccione una especialidad…</option>
                  {especialidades.map((e) => (
                    <option key={e} value={e}>
                      {e}
                    </option>
                  ))}
                </select>
              </Field>
              {especialidades.length === 0 && (
                <p className="text-xs text-muted">
                  No hay especialistas registrados todavía.
                </p>
              )}
              <Field label="Motivo (opcional)">
                <Input
                  value={autMotivo}
                  onChange={(e) => setAutMotivo(e.target.value)}
                  placeholder="Motivo de la remisión…"
                />
              </Field>
              {autError && <p className="text-sm text-danger">{autError}</p>}
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setAutPaciente(null)}
                >
                  Cancelar
                </Button>
                <Button type="submit" disabled={autSaving || !especialidadSel}>
                  {autSaving ? "Autorizando…" : "Autorizar"}
                </Button>
              </div>
            </form>
          </Card>
        </div>
      )}

      {/* Formulario nuevo paciente */}
      {showNew && (
        <Card className="flex flex-col gap-4 border-teal/30 bg-teal/5 p-5">
          <div className="flex items-center justify-between">
            <h3 className="font-display font-semibold text-ink">
              Registrar paciente
            </h3>
            <button
              type="button"
              onClick={() => setShowNew(false)}
              className="text-muted hover:text-ink"
            >
              <X className="size-4" />
            </button>
          </div>

          <form onSubmit={handleCreate} className="flex flex-col gap-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <Field label="Tipo de documento">
                <select
                  className={selectClass}
                  value={form.tipo_documento}
                  onChange={(e) =>
                    setForm({ ...form, tipo_documento: e.target.value })
                  }
                >
                  <option value="CC">Cédula de ciudadanía</option>
                  <option value="TI">Tarjeta de identidad</option>
                  <option value="CE">Cédula de extranjería</option>
                  <option value="PA">Pasaporte</option>
                  <option value="RC">Registro civil</option>
                </select>
              </Field>
              <Field label="Número de documento">
                <Input
                  required
                  value={form.numero_documento}
                  onChange={(e) =>
                    setForm({ ...form, numero_documento: e.target.value })
                  }
                />
              </Field>
              <Field label="Nombres">
                <Input
                  required
                  value={form.nombre_paciente}
                  onChange={(e) =>
                    setForm({ ...form, nombre_paciente: e.target.value })
                  }
                />
              </Field>
              <Field label="Apellidos">
                <Input
                  required
                  value={form.apellidos_paciente}
                  onChange={(e) =>
                    setForm({ ...form, apellidos_paciente: e.target.value })
                  }
                />
              </Field>
              <Field label="Fecha de nacimiento">
                <Input
                  type="date"
                  required
                  value={form.fecha_nacimiento}
                  onChange={(e) =>
                    setForm({ ...form, fecha_nacimiento: e.target.value })
                  }
                />
              </Field>
              <Field label="Sexo">
                <select
                  className={selectClass}
                  value={form.sexo}
                  onChange={(e) => setForm({ ...form, sexo: e.target.value })}
                >
                  <option value="">Prefiere no decir</option>
                  <option value="M">Masculino</option>
                  <option value="F">Femenino</option>
                  <option value="O">Otro</option>
                </select>
              </Field>
              <Field label="Correo electrónico" hint={<span className="text-[10px] text-muted">Recibirá su contraseña</span>}>
                <Input
                  type="email"
                  required
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                />
              </Field>
              <Field label="Teléfono">
                <Input
                  value={form.telefono}
                  onChange={(e) =>
                    setForm({ ...form, telefono: e.target.value })
                  }
                />
              </Field>
              <Field label="Grupo sanguíneo">
                <select
                  className={selectClass}
                  value={form.tipo_sangre}
                  onChange={(e) => setForm({ ...form, tipo_sangre: e.target.value })}
                >
                  <option value="">Desconocido</option>
                  <option value="A+">A+</option>
                  <option value="A-">A-</option>
                  <option value="B+">B+</option>
                  <option value="B-">B-</option>
                  <option value="AB+">AB+</option>
                  <option value="AB-">AB-</option>
                  <option value="O+">O+</option>
                  <option value="O-">O-</option>
                </select>
              </Field>
              <Field label="Aseguradora">
                <Input
                  value={form.aseguradora}
                  onChange={(e) => setForm({ ...form, aseguradora: e.target.value })}
                  placeholder="Ej: Sura, Nueva EPS…"
                />
              </Field>
            </div>
            <Field label="Alergias">
              <Input
                value={form.alergias}
                onChange={(e) => setForm({ ...form, alergias: e.target.value })}
                placeholder="Ej: Penicilina, mariscos… (opcional)"
              />
            </Field>
            <Field label="Dirección">
              <Input
                value={form.direccion}
                onChange={(e) =>
                  setForm({ ...form, direccion: e.target.value })
                }
              />
            </Field>

            {createError && (
              <p className="text-sm text-danger">{createError}</p>
            )}

            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setShowNew(false)}
              >
                Cancelar
              </Button>
              <Button type="submit" disabled={creating}>
                {creating ? "Registrando…" : "Registrar paciente"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      {/* Barra de búsqueda */}
      <Card className="flex flex-col gap-4 p-5">
        <Input
          icon={<Search className="size-4" />}
          placeholder="Buscar por nombre o número de documento…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </Card>

      {error && <p className="text-sm text-red-500">{error}</p>}

      {/* Tabla */}
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[#e6f2fa] text-left text-xs uppercase tracking-[0.6px] text-label">
              <th className="px-6 py-4 font-normal">Paciente</th>
              <th className="px-6 py-4 font-normal">
                <span className="flex items-center gap-1.5">
                  <Phone className="size-3" />
                  Contacto
                </span>
              </th>
              <th className="px-6 py-4 font-normal">
                <span className="flex items-center gap-1.5">
                  <Calendar className="size-3" />
                  Última consulta
                </span>
              </th>
              <th className="px-6 py-4 font-normal">Próxima cita</th>
              <th className="px-6 py-4 font-normal text-right">Acción</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-slate">
                  Cargando pacientes…
                </td>
              </tr>
            )}

            {!loading && pacientes.length === 0 && (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-slate">
                  No se encontraron pacientes.
                </td>
              </tr>
            )}

            {!loading &&
              pacientes.map((p) => (
                <tr key={p.id} className="border-t border-line hover:bg-shell">
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-[#91b9cf]/20 text-xs font-medium text-teal">
                        {initials(p.nombre_paciente, p.apellidos_paciente)}
                      </span>
                      <div>
                        <p className="flex items-center gap-2 font-medium text-navy-800">
                          {p.nombre_paciente} {p.apellidos_paciente}
                          {!p.es_tratante && (
                            <span
                              title="Paciente remitido por su médico general. Lo atiendes de forma temporal; su médico a cargo no cambia."
                              className="rounded-full border border-warning/30 bg-warning/10 px-2 py-0.5 text-[10px] font-medium text-warning"
                            >
                              Temporal
                            </span>
                          )}
                        </p>
                        <p className="font-mono text-xs text-muted">
                          {p.numero_documento}
                        </p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-slate">{p.telefono ?? "—"}</td>
                  <td className="px-6 py-4 text-slate">
                    {formatDate(p.ultima_consulta)}
                  </td>
                  <td className="px-6 py-4 text-slate">
                    {formatDate(p.proxima_cita)}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex flex-wrap items-center justify-end gap-2">
                      {/* Consultar: solo si tiene cita programada para hoy */}
                      {p.tiene_cita_hoy ? (
                        <Link
                          href={`/consulta?paciente=${p.id}`}
                          className="inline-flex items-center gap-1.5 rounded-[var(--radius)] bg-navy px-3 py-1.5 text-xs font-medium text-white hover:bg-navy-800"
                        >
                          <Stethoscope className="size-3.5" />
                          Consultar
                        </Link>
                      ) : (
                        <span
                          title="Sin cita programada para hoy. Agenda una cita para poder consultar."
                          className="inline-flex cursor-not-allowed items-center gap-1.5 rounded-[var(--radius)] border border-line bg-field px-3 py-1.5 text-xs font-medium text-muted"
                        >
                          <Stethoscope className="size-3.5" />
                          Consultar
                        </span>
                      )}

                      {/* Autorizar especialista: solo el médico tratante */}
                      {p.es_tratante && (
                        <button
                          type="button"
                          onClick={() => openAutorizar(p)}
                          className="inline-flex items-center gap-1.5 rounded-[var(--radius)] border border-line px-3 py-1.5 text-xs font-medium text-slate hover:bg-field"
                        >
                          <ShieldCheck className="size-3.5" />
                          Autorizar especialista
                        </button>
                      )}

                      {/* Fórmulas médicas */}
                      <Link
                        href={`/formulas?paciente=${p.id}`}
                        className="inline-flex items-center gap-1.5 rounded-[var(--radius)] border border-line px-3 py-1.5 text-xs font-medium text-slate hover:bg-field"
                      >
                        <Pill className="size-3.5" />
                        Fórmulas
                      </Link>

                      {/* Ver historia clínica */}
                      <Link
                        href={`/historia-clinica?paciente=${p.id}`}
                        className="inline-flex items-center gap-1.5 rounded-[var(--radius)] border border-teal/30 px-3 py-1.5 text-xs font-medium text-teal hover:bg-teal/5"
                      >
                        <FileText className="size-3.5" />
                        Ver HC
                      </Link>
                    </div>
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
        <div className="flex items-center justify-between border-t border-line px-6 py-4 text-xs text-muted">
          <span>
            {pacientes.length > 0
              ? `Mostrando 1–${pacientes.length} de ${pacientes.length} pacientes`
              : ""}
          </span>
        </div>
      </Card>
    </div>
  );
}