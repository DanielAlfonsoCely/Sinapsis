"use client";

import { useState, useEffect, useRef } from "react";
import Link from "next/link";
import {
  Building2,
  Plus,
  Search,
  MapPin,
  ChevronLeft,
  ChevronRight,
  X,
  CheckCircle,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/input";

// ── Tipos para el listado dinámico ─────────────────────────────────────────

interface AdminEntidadListItem {
  id: string;
  nombre_entidad: string;
  tipo_entidad: string;
  nit: string;
  ciudad: string | null;
  estado: boolean;
  fecha_creacion: string;
}

interface AdminEntidadListResponse {
  entidades: AdminEntidadListItem[];
  total: number;
}

// ── Formulario de registro (no modificar) ──────────────────────────────────

const selectClass =
  "h-11 w-full rounded-[var(--radius)] border border-line bg-field px-4 text-sm text-navy-800 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20";

const EMPTY_FORM = {
  nombre_entidad: "",
  tipo_entidad: "IPS",
  nit: "",
  ciudad: "",
  direccion: "",
  telefono: "",
};

type FormState = typeof EMPTY_FORM;

function validateForm(form: FormState): string | null {
  if (!form.nombre_entidad.trim()) return "El nombre es obligatorio.";
  if (form.nombre_entidad.length > 150) return "El nombre no puede superar 150 caracteres.";
  if (!form.nit.trim()) return "El NIT es obligatorio.";
  if (form.nit.length > 50) return "El NIT no puede superar 50 caracteres.";
  if (form.ciudad.length > 100) return "La ciudad no puede superar 100 caracteres.";
  if (form.direccion.length > 255) return "La dirección no puede superar 255 caracteres.";
  if (form.telefono.length > 50) return "El teléfono no puede superar 50 caracteres.";
  return null;
}

// ── Componente principal ───────────────────────────────────────────────────

export default function EntidadesPage() {
  // Estado del modal de registro (NO SE TOCA)
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  // Estado para el listado dinámico
  const [entidades, setEntidades] = useState<AdminEntidadListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState("");
  const [loadingList, setLoadingList] = useState(true);
  const [listError, setListError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [usuariosActivos, setUsuariosActivos] = useState(0)
  const [totalConsultas, setTotalConsultas] = useState(0)
  const [totalPacientes, setTotalPacientes] = useState(0)

  async function fetchStats() {
    try {
      const token = document.cookie
        .split("; ")
        .find((c) => c.startsWith("token="))
        ?.split("=")[1]
      const res = await fetch("http://localhost:8080/api/v1/admin/stats", {
        headers: { Authorization: `Bearer ${token ?? ""}` },
      })
      if (!res.ok) return
      const data = await res.json()
      setUsuariosActivos(data.total_usuarios_activos ?? 0)
      setTotalConsultas(data.total_consultas ?? 0)
      setTotalPacientes(data.total_pacientes_activos ?? 0)
    } catch {
      // silencioso — las cards quedan en 0
    }
  }

  function openModal() {
    setForm(EMPTY_FORM);
    setError("");
    setSuccess(false);
    setShowModal(true);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const validationError = validateForm(form);
    if (validationError) {
      setError(validationError);
      return;
    }
    setSaving(true);
    setError("");
    try {
      const token = localStorage.getItem("token");
      const res = await fetch("http://localhost:8080/api/v1/entidades", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          nombre_entidad: form.nombre_entidad.trim(),
          tipo_entidad: form.tipo_entidad,
          nit: form.nit.trim(),
          ciudad: form.ciudad.trim() || undefined,
          direccion: form.direccion.trim() || undefined,
          telefono: form.telefono.trim() || undefined,
        }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error ?? "No se pudo registrar la entidad.");
        return;
      }
      setSuccess(true);
    } catch {
      setError("Error de conexión con el servidor.");
    } finally {
      setSaving(false);
    }
  }

  async function fetchEntidades(q: string) {
    setLoadingList(true);
    setListError(null);
    try {
      const token = document.cookie
        .split("; ")
        .find((c) => c.startsWith("token="))
        ?.split("=")[1];
      const params = new URLSearchParams();
      if (q) params.set("q", q);
      const res = await fetch(
        `http://localhost:8080/api/v1/admin/entidades?${params.toString()}`,
        { headers: { Authorization: `Bearer ${token ?? ""}` } }
      );
      if (res.status === 403) {
        window.location.href = "/login";
        return;
      }
      if (!res.ok) throw new Error(`Error ${res.status}`);
      const data: AdminEntidadListResponse = await res.json();
      setEntidades(data.entidades);
      setTotal(data.total);
    } catch {
      setListError("No se pudo cargar el listado de entidades.");
      setEntidades([]);
    } finally {
      setLoadingList(false);
    }
  }

  useEffect(() => {
    void fetchEntidades("");
    void fetchStats();
  }, []);

  function handleSearchChange(value: string) {
    setQuery(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => void fetchEntidades(value), 400);
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy/40 p-4">
          <Card className="flex w-full max-w-lg flex-col gap-5 p-6">
            {success ? (
              <div className="flex flex-col items-center gap-3 py-4 text-center">
                <div className="flex size-12 items-center justify-center rounded-full bg-success/10 text-success">
                  <CheckCircle className="size-6" />
                </div>
                <h3 className="font-display text-lg font-semibold text-ink">
                  Entidad registrada
                </h3>
                <p className="text-sm text-slate">
                  <span className="font-medium text-ink">{form.nombre_entidad}</span> fue
                  creada exitosamente.
                </p>
                <Button className="mt-2" onClick={() => setShowModal(false)}>
                  Cerrar
                </Button>
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between">
                  <h3 className="font-display text-lg font-semibold text-ink">
                    Registrar entidad de salud
                  </h3>
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="text-muted hover:text-ink"
                  >
                    <X className="size-4" />
                  </button>
                </div>

                <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <Field label="Nombre de la entidad">
                      <Input
                        required
                        maxLength={150}
                        value={form.nombre_entidad}
                        onChange={(e) =>
                          setForm({ ...form, nombre_entidad: e.target.value })
                        }
                        placeholder="Hospital Universitario..."
                      />
                    </Field>

                    <Field label="Tipo de entidad">
                      <select
                        className={selectClass}
                        value={form.tipo_entidad}
                        onChange={(e) =>
                          setForm({ ...form, tipo_entidad: e.target.value })
                        }
                      >
                        <option value="IPS">IPS</option>
                        <option value="EPS">EPS</option>
                        <option value="clinica">Clínica</option>
                        <option value="hospital">Hospital</option>
                        <option value="consultorio">Consultorio</option>
                      </select>
                    </Field>

                    <Field label="NIT">
                      <Input
                        required
                        maxLength={50}
                        value={form.nit}
                        onChange={(e) =>
                          setForm({ ...form, nit: e.target.value })
                        }
                        placeholder="900.123.456-7"
                      />
                    </Field>

                    <Field label="Ciudad">
                      <Input
                        maxLength={100}
                        value={form.ciudad}
                        onChange={(e) =>
                          setForm({ ...form, ciudad: e.target.value })
                        }
                        placeholder="Bogotá"
                      />
                    </Field>

                    <Field label="Teléfono">
                      <Input
                        maxLength={50}
                        value={form.telefono}
                        onChange={(e) =>
                          setForm({ ...form, telefono: e.target.value })
                        }
                        placeholder="6012000000"
                      />
                    </Field>
                  </div>

                  <Field label="Dirección">
                    <Input
                      maxLength={255}
                      value={form.direccion}
                      onChange={(e) =>
                        setForm({ ...form, direccion: e.target.value })
                      }
                      placeholder="Calle 119 # 7-75"
                    />
                  </Field>

                  {error && <p className="text-sm text-danger">{error}</p>}

                  <div className="flex justify-end gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setShowModal(false)}
                    >
                      Cancelar
                    </Button>
                    <Button type="submit" disabled={saving}>
                      {saving ? "Registrando…" : "Registrar entidad"}
                    </Button>
                  </div>
                </form>
              </>
            )}
          </Card>
        </div>
      )}

      {/* Encabezado */}
      <div className="flex items-center justify-between border-b border-line pb-5">
        <div className="flex items-center gap-3">
          <Building2 className="size-5 text-teal" />
          <h2 className="font-display text-2xl font-semibold text-ink">
            Gestión de Entidades de Salud
          </h2>
        </div>
        <button
          onClick={openModal}
          className="flex items-center gap-2 rounded bg-navy px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-navy-800"
        >
          <Plus className="size-4" />
          Registrar Entidad
        </button>
      </div>

      {/* Stats del sistema */}
      <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
        {[
          { label: "TOTAL ENTIDADES",       value: total.toLocaleString("es-CO"),          sub: "registradas" },
          { label: "USUARIOS ACTIVOS",      value: usuariosActivos.toLocaleString("es-CO"), sub: "en la plataforma" },
          { label: "CONSULTAS REGISTRADAS", value: totalConsultas.toLocaleString("es-CO"),  sub: "historial total" },
          { label: "PACIENTES ACTIVOS",     value: totalPacientes.toLocaleString("es-CO"),  sub: "en el sistema" },
        ].map(({ label, value, sub }) => (
          <Card key={label} className="p-5">
            <p className="text-[10px] uppercase tracking-[0.6px] text-muted">{label}</p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">{value}</p>
            <p className="text-xs text-label">{sub}</p>
          </Card>
        ))}
      </div>

      {/* Buscador */}
      <Card className="p-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted" />
          <input
            type="text"
            value={query}
            onChange={(e) => handleSearchChange(e.target.value)}
            placeholder="Buscar por NIT, nombre de entidad o ciudad..."
            className="h-10 w-full rounded border border-line bg-field pl-9 pr-4 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
          />
        </div>
      </Card>

      {/* Cards de entidades */}
      {listError && (
        <div className="rounded border border-danger bg-danger/10 p-4 text-sm text-danger">
          {listError}
        </div>
      )}
      {loadingList && !listError && (
        <div className="py-8 text-center text-sm text-muted">Cargando entidades…</div>
      )}
      {!loadingList && !listError && entidades.length === 0 && (
        <div className="py-8 text-center text-sm text-muted">No se encontraron entidades.</div>
      )}
      <div className="flex flex-col gap-4">
        {entidades.map((ent) => {
          const initials = ent.nombre_entidad
            .split(" ")
            .slice(0, 2)
            .map((w) => w[0])
            .join("")
            .toUpperCase();
          return (
            <Card key={ent.id} className="p-6">
              <div className="flex items-start gap-6">
                <div className="flex size-20 shrink-0 items-center justify-center rounded border border-line bg-field">
                  <span className="font-display text-xl font-bold text-teal">{initials}</span>
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="flex items-center gap-3">
                        <h3 className="font-display text-lg font-semibold text-ink">
                          {ent.nombre_entidad}
                        </h3>
                        <Badge tone={ent.estado ? "success" : "danger"}>
                          {ent.estado ? "Activa" : "Inactiva"}
                        </Badge>
                        <Badge tone="neutral">{ent.tipo_entidad}</Badge>
                      </div>
                      <div className="mt-1 flex items-center gap-1 text-sm text-slate">
                        <MapPin className="size-3" />
                        {ent.ciudad ?? "—"}
                      </div>
                    </div>
                    <Link
                      href={`/admin/entidades/${ent.id}`}
                      className="flex items-center gap-2 rounded border border-navy px-4 py-2 text-sm text-navy transition-colors hover:bg-navy hover:text-white"
                    >
                      Ver Detalle
                    </Link>
                  </div>
                  <div className="mt-4 grid grid-cols-3 gap-4 border-t border-line pt-4">
                    <div>
                      <p className="text-[10px] uppercase tracking-[0.6px] text-muted">NIT</p>
                      <p className="mt-0.5 text-sm text-navy-800">{ent.nit}</p>
                    </div>
                    <div>
                      <p className="text-[10px] uppercase tracking-[0.6px] text-muted">TIPO</p>
                      <p className="mt-0.5 text-sm text-navy-800">{ent.tipo_entidad}</p>
                    </div>
                    <div>
                      <p className="text-[10px] uppercase tracking-[0.6px] text-muted">CIUDAD</p>
                      <p className="mt-0.5 text-sm text-navy-800">{ent.ciudad ?? "—"}</p>
                    </div>
                  </div>
                </div>
              </div>
            </Card>
          );
        })}
      </div>

      {/* Paginación */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted">
          {total > 0 ? `Mostrando ${entidades.length} de ${total} entidades` : "Sin entidades"}
        </p>
        {total > entidades.length && (
          <div className="flex items-center gap-1">
            <button className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <ChevronLeft className="size-4" />
            </button>
            {[1, 2, 3].map((p) => (
              <button
                key={p}
                className={`flex size-8 items-center justify-center rounded border text-sm transition-colors ${
                  p === 1
                    ? "border-teal bg-teal text-white"
                    : "border-line text-slate hover:bg-field"
                }`}
              >
                {p}
              </button>
            ))}
            <button className="flex size-8 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <ChevronRight className="size-4" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
