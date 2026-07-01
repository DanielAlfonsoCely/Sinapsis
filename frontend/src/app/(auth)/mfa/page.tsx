"use client";

import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { Lock, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

export default function MfaPage() {
  const router = useRouter();
  const [code, setCode] = useState<string[]>(Array(6).fill(""));
  const inputs = useRef<(HTMLInputElement | null)[]>([]);

  function handleChange(index: number, value: string) {
    const digit = value.replace(/\D/g, "").slice(-1);
    const next = [...code];
    next[index] = digit;
    setCode(next);
    if (digit && index < 5) inputs.current[index + 1]?.focus();
  }

  function handleKeyDown(index: number, e: React.KeyboardEvent) {
    if (e.key === "Backspace" && !code[index] && index > 0) {
      inputs.current[index - 1]?.focus();
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    router.push("/dashboard");
  }

  return (
    <div className="flex w-full max-w-lg flex-col gap-6">
      <Card className="p-8">
        {/* Encabezado */}
        <div className="mb-6 flex flex-col gap-2">
          <div className="flex items-center gap-2 text-teal">
            <ShieldCheck className="size-5" />
            <span className="text-sm uppercase tracking-[0.7px]">
              Acceso protegido
            </span>
          </div>
          <h1 className="font-display text-2xl font-semibold text-ink">
            Verificación de Seguridad
          </h1>
          <p className="text-base text-slate">
            Ingresa el código de 6 dígitos enviado a tu correo institucional.
          </p>
        </div>

        <form className="flex flex-col gap-8" onSubmit={handleSubmit}>
          <div className="flex justify-center gap-4">
            {code.map((digit, i) => (
              <input
                key={i}
                ref={(el) => {
                  inputs.current[i] = el;
                }}
                inputMode="numeric"
                maxLength={1}
                value={digit}
                onChange={(e) => handleChange(i, e.target.value)}
                onKeyDown={(e) => handleKeyDown(i, e)}
                className="h-14 w-full max-w-16 rounded-[var(--radius)] border border-line bg-white text-center font-display text-2xl font-semibold text-ink outline-none transition-colors focus:border-teal focus:ring-2 focus:ring-teal/20"
              />
            ))}
          </div>

          <Button type="submit" size="lg" className="w-full">
            Verificar e Ingresar
          </Button>
        </form>

        {/* Acciones de pie */}
        <div className="mt-6 flex flex-col gap-4 border-t border-line pt-4">
          <div className="flex items-center justify-between">
            <p className="flex items-center gap-1.5 text-sm text-slate">
              <Lock className="size-4 text-muted" />
              Reenviar código en{" "}
              <span className="font-mono tabular-nums">00:56</span>
            </p>
            <button
              type="button"
              className="text-xs uppercase tracking-[0.6px] text-navy"
            >
              Reenviar ahora
            </button>
          </div>
          <p className="text-center text-sm text-slate">
            ¿No tienes acceso a tu correo?{" "}
            <a href="#" className="text-teal">
              Contactar Soporte IT
            </a>
          </p>
        </div>
      </Card>

      {/* Sello de confianza */}
      <div className="flex items-center justify-center gap-4 text-[10px] uppercase tracking-[1px] text-label">
        <span className="flex items-center gap-1">
          <Lock className="size-3" /> Encriptación AES-256
        </span>
        <span className="h-3 w-px bg-line" />
        <span className="flex items-center gap-1">
          <ShieldCheck className="size-3" /> Cumple normativa Habeas Data
        </span>
      </div>
    </div>
  );
}
