"use client";

import { useState } from "react";
import {
  Building2,
  Users,
  Activity,
  BarChart2,
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

const ENTIDADES = [
  {
    initials: "CG",
    name: "Central General IPS",
    nit: "900.123.456-7",
    location: "Medellín, CO",
    admin: "Marc Jacobs",
    patients: "12,402",
    apiUsage: 75,
    status: "Activa" as const,
    plan: "Enterprise",
  },
  {
    initials: "ND",
    name: "Norte Diagnostic Lab",
    nit: "800.987.654-3",
    location: "Bogotá, CO",
    admin: "Sarah Connor",
    patients: "5,120",
    apiUsage: 42,
    status: "Activa" as const,
    plan: "Pro",
  },
  {
    initials: "PP",
    name: "Pacific Pediatric Clinic",
    nit: "700.456.789-1",
    location: "Cali, CO",
    admin: "Roberto Gomez",
    patients: "2,880",
    apiUsage: 31,
    status: "Offline" as const,
    plan: "Basic",
  },
];

const STATUS_TONE = {
  Activa: "success" as const,
  Offline: "danger" as const,
  Suspendida: "warning" as const,
};

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

export default function EntidadesPage() {
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

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
          { label: "TOTAL ENTIDADES", value: "160", sub: "registradas" },
          { label: "USUARIOS ACTIVOS GLOBALES", value: "42,891", sub: "en línea" },
          { label: "SALUD DEL SISTEMA", value: "Óptima", sub: "todos los nodos" },
          { label: "CARGA PROMEDIO API", value: "68%", sub: "uso actual" },
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
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted" />
            <input
              type="text"
              placeholder="Buscar por NIT, nombre de entidad o ciudad..."
              className="h-10 w-full rounded border border-line bg-field pl-9 pr-4 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
            />
          </div>
          <button className="flex items-center gap-2 rounded border border-navy px-4 py-2 text-sm text-navy transition-colors hover:bg-navy hover:text-white">
            <BarChart2 className="size-4" />
            Exportar Reporte
          </button>
          <div className="flex items-center gap-2 border-l border-line pl-3">
            <button className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <BarChart2 className="size-4" />
            </button>
            <button className="flex size-9 items-center justify-center rounded border border-line text-slate transition-colors hover:bg-field">
              <Users className="size-4" />
            </button>
          </div>
        </div>
      </Card>

      {/* Cards de entidades */}
      <div className="flex flex-col gap-4">
        {ENTIDADES.map((ent) => (
          <Card key={ent.nit} className="p-6">
            <div className="flex items-start gap-6">
              <div className="flex size-20 shrink-0 items-center justify-center rounded border border-line bg-field">
                <span className="font-display text-xl font-bold text-teal">
                  {ent.initials}
                </span>
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="flex items-center gap-3">
                      <h3 className="font-display text-lg font-semibold text-ink">
                        {ent.name}
                      </h3>
                      <Badge tone={STATUS_TONE[ent.status]}>{ent.status}</Badge>
                      <Badge tone="neutral">{ent.plan}</Badge>
                    </div>
                    <div className="mt-1 flex items-center gap-1 text-sm text-slate">
                      <MapPin className="size-3" />
                      {ent.location}
                    </div>
                  </div>
                </div>
                <div className="mt-4 grid grid-cols-4 gap-4 border-t border-line pt-4">
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">UBICACIÓN</p>
                    <p className="mt-0.5 text-sm text-navy-800">{ent.location}</p>
                  </div>
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">ADMINISTRADOR</p>
                    <p className="mt-0.5 text-sm text-navy-800">{ent.admin}</p>
                  </div>
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">PACIENTES ACTIVOS</p>
                    <p className="mt-0.5 text-sm text-navy-800">{ent.patients}</p>
                  </div>
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.6px] text-muted">USO DE API</p>
                    <div className="mt-1 flex items-center gap-2">
                      <span className="text-sm text-navy-800">{ent.apiUsage}%</span>
                      <div className="h-1.5 flex-1 rounded-full bg-field">
                        <div
                          className="h-full rounded-full bg-teal"
                          style={{ width: `${ent.apiUsage}%` }}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Paginación */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted">Mostrando 1–3 de 160 entidades</p>
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
      </div>
    </div>
  );
}