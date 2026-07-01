"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ArrowRight, ChevronDown, Lock, User } from "lucide-react";
import { BrandMark } from "@/components/brand";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";

const ROLES = [
  { value: "medico", label: "Médico / Profesional de salud" },
  { value: "paciente", label: "Paciente" },
  { value: "admin_entidad", label: "Administrador de entidad" },
  { value: "admin_plataforma", label: "Administrador de plataforma" },
];

export default function LoginPage() {
  const router = useRouter();
  const [role, setRole] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    // Flujo: tras validar credenciales se exige verificación MFA.
    router.push("/mfa");
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
          <Field label="Rol de acceso" htmlFor="role">
            <div className="relative w-full">
              <select
                id="role"
                value={role}
                onChange={(e) => setRole(e.target.value)}
                className="h-11 w-full appearance-none rounded-[var(--radius)] border border-line bg-field px-4 pr-10 text-base text-navy-800 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20"
              >
                <option value="" disabled>
                  Seleccione su rol
                </option>
                {ROLES.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </select>
              <ChevronDown className="pointer-events-none absolute right-3.5 top-1/2 size-5 -translate-y-1/2 text-muted" />
            </div>
          </Field>

          <Field label="Correo institucional o cédula" htmlFor="user">
            <Input
              id="user"
              type="text"
              placeholder="ej: j.perez@clinica.com"
              icon={<User className="size-5" />}
              autoComplete="username"
            />
          </Field>

          <Field
            label="Contraseña"
            htmlFor="password"
            hint={
              <a href="#" className="text-xs tracking-[0.6px] text-teal">
                ¿Olvidó su contraseña?
              </a>
            }
          >
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              icon={<Lock className="size-5" />}
              autoComplete="current-password"
            />
          </Field>

          <Button type="submit" size="lg" className="mt-2 w-full">
            Iniciar Sesión
            <ArrowRight className="size-5" />
          </Button>
        </form>

        <div className="mt-6 border-t border-line pt-4 text-center text-sm text-slate">
          ¿Problemas para acceder?{" "}
          <a href="#" className="text-teal">
            Contacte a TI
          </a>
        </div>
      </Card>

      {/* Footer de cumplimiento */}
      <footer className="flex flex-col items-center gap-1 pt-8 opacity-60">
        <p className="text-xs tracking-[0.6px] text-label">
          © 2026 SINAPSIS Health Systems. Todos los derechos reservados.
        </p>
        <div className="flex gap-4">
          <a href="#" className="text-xs tracking-[0.6px] text-label underline">
            Privacidad
          </a>
          <a href="#" className="text-xs tracking-[0.6px] text-label underline">
            Términos
          </a>
        </div>
      </footer>
    </div>
  );
}
