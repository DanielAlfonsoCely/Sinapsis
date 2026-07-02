"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import {
  Search,
  UserPlus,
  ArrowUpRight,
  Phone,
  Calendar,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

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

  useEffect(() => {
    const timeout = setTimeout(() => {
      fetchPacientes(query);
    }, 300); // debounce simple para no pegarle al backend en cada tecla

    return () => clearTimeout(timeout);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

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
        <Button size="sm" disabled>
          <UserPlus className="size-4" />
          Nuevo Paciente
        </Button>
      </div>

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
                        <p className="font-medium text-navy-800">
                          {p.nombre_paciente} {p.apellidos_paciente}
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
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Link
                        href={`/consulta?paciente=${p.id}`}
                        className="rounded border border-line px-2.5 py-1 text-xs text-slate hover:bg-field"
                      >
                        Consultar
                      </Link>
                      <Link
                        href={`/historia-clinica?paciente=${p.id}`}
                        className="inline-flex text-teal hover:text-teal-700"
                      >
                        <ArrowUpRight className="size-[18px]" />
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