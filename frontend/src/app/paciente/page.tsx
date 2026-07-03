"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { LogOut, User, Phone, Mail, Calendar, MapPin } from "lucide-react";
import { Wordmark } from "@/components/brand";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

type PacienteDetalle = {
  nombre_paciente: string;
  apellidos_paciente: string;
  numero_documento: string;
  tipo_documento: string;
  fecha_nacimiento: string;
  sexo: string | null;
  telefono: string | null;
  email: string | null;
  direccion: string | null;
};

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("es-CO", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  });
}

export default function PacienteHomePage() {
  const router = useRouter();
  const [paciente, setPaciente] = useState<PacienteDetalle | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }

    fetch("http://localhost:8080/api/v1/pacientes/me", {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(async (res) => {
        if (!res.ok) {
          setError("No se pudieron cargar sus datos.");
          return;
        }
        setPaciente(await res.json());
      })
      .catch(() => setError("Error de conexión con el servidor"))
      .finally(() => setLoading(false));
  }, [router]);

  function handleLogout() {
    localStorage.removeItem("token");
    localStorage.removeItem("usuario");
    document.cookie = "token=; path=/; max-age=0";
    router.push("/login");
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

        <Card className="flex flex-col gap-5 p-6">
          {loading && (
            <p className="text-center text-sm text-slate">Cargando…</p>
          )}
          {error && <p className="text-center text-sm text-danger">{error}</p>}

          {paciente && (
            <>
              <div className="flex flex-col items-center gap-2 text-center">
                <div className="flex size-16 items-center justify-center rounded-full bg-[#91b9cf]/20 text-teal">
                  <User className="size-8" />
                </div>
                <h1 className="font-display text-xl font-semibold text-ink">
                  {paciente.nombre_paciente} {paciente.apellidos_paciente}
                </h1>
                <p className="font-mono text-sm text-muted">
                  {paciente.tipo_documento} {paciente.numero_documento}
                </p>
              </div>

              <div className="flex flex-col gap-3 border-t border-line pt-4">
                <div className="flex items-center gap-3 text-sm">
                  <Calendar className="size-4 shrink-0 text-teal" />
                  <span className="text-slate">
                    Nacido el {formatDate(paciente.fecha_nacimiento)}
                  </span>
                </div>
                {paciente.email && (
                  <div className="flex items-center gap-3 text-sm">
                    <Mail className="size-4 shrink-0 text-teal" />
                    <span className="text-slate">{paciente.email}</span>
                  </div>
                )}
                {paciente.telefono && (
                  <div className="flex items-center gap-3 text-sm">
                    <Phone className="size-4 shrink-0 text-teal" />
                    <span className="text-slate">{paciente.telefono}</span>
                  </div>
                )}
                {paciente.direccion && (
                  <div className="flex items-center gap-3 text-sm">
                    <MapPin className="size-4 shrink-0 text-teal" />
                    <span className="text-slate">{paciente.direccion}</span>
                  </div>
                )}
              </div>
            </>
          )}
        </Card>
      </div>
    </div>
  );
}
