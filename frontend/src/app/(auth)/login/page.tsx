"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ArrowRight, Lock, User } from "lucide-react";
import { BrandMark } from "@/components/brand";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch("http://localhost:8080/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, contrasena: password }),
      });

      const data = await res.json();

      if (!res.ok) {
        setError(data.error || "Credenciales incorrectas");
        return;
      }

      localStorage.setItem("token", data.token);
      document.cookie = `token=${data.token}; path=/; max-age=86400`;
      localStorage.setItem("usuario", JSON.stringify(data.usuario));

      const tipo = data.usuario.tipo_usuario;

      if (tipo === "medico") {
        router.push("/dashboard");
      } else if (tipo === "paciente") {
        router.push("/paciente");
      } else if (tipo === "admin_plataforma") {
        router.push("/admin/dashboard_admin");
      } else if (tipo === "admin_entidad") {
        setError("El panel de administración estará disponible próximamente.");
      } else {
        setError("Rol no soportado en esta versión de la plataforma.");
      }
    } catch {
      setError("Error de conexión con el servidor");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col items-center">
      {/* Branding */}
      <div className="flex flex-col items-center gap-4 pb-8">
        <div className="flex size-28 items-center justify-center rounded-2xl bg-white shadow-[0px_12px_24px_-10px_rgba(0,21,48,0.08)]">
          <BrandMark className="size-16 text-teal" />
        </div>
        <p className="max-w-xs text-center text-sm tracking-[0.35px] text-slate">
          SISTEMA INTELIGENTE DE ANÁLISIS DE PATRONES DE SALUD INTEGRADOS
        </p>
      </div>

      {/* Tarjeta de acceso */}
      <Card className="w-full p-6">
        <div className="mb-6 flex flex-col gap-1">
          <h1 className="font-display text-2xl font-semibold text-ink">
            Bienvenido
          </h1>
          <p className="text-sm text-slate">
            Ingrese sus credenciales institucionales para continuar.
          </p>
        </div>

        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          <Field label="Correo institucional o cédula" htmlFor="user">
            <Input
              id="user"
              type="email"
              placeholder="ej: j.perez@clinica.com"
              icon={<User className="size-5" />}
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </Field>

          <Field label="Contraseña" htmlFor="password">
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              icon={<Lock className="size-5" />}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Field>

          {error && <p className="text-sm text-red-500">{error}</p>}

          <Button type="submit" size="lg" className="mt-2 w-full" disabled={loading}>
            {loading ? "Iniciando sesión..." : "Iniciar Sesión"}
            {!loading && <ArrowRight className="size-5" />}
          </Button>
        </form>
      </Card>
    </div>
  );
}