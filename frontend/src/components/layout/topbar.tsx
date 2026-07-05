"use client";

import { useEffect, useState } from "react";
import { Bell, HelpCircle } from "lucide-react";

export function Topbar() {
  const [usuario, setUsuario] = useState({
    nombre_usuario: "",
    apellidos: "",
    especialidad: null as string | null,
    entidad: null as string | null,
  });

  useEffect(() => {
    try {
      const raw = localStorage.getItem("usuario");
      if (raw) {
        const u = JSON.parse(raw);
        setUsuario({
          nombre_usuario: u.nombre_usuario ?? "",
          apellidos: u.apellidos ?? "",
          especialidad: u.especialidad ?? null,
          entidad: u.entidad ?? null,
        });
      }
    } catch {}
  }, []);

  const nombreCompleto = usuario.nombre_usuario
    ? `Dr. ${usuario.nombre_usuario} ${usuario.apellidos}`.trim()
    : "";

  const rolDisplay = usuario.especialidad
    ? `Médico ${usuario.especialidad}`
    : "Médico";

  const entidadDisplay = usuario.entidad ?? "Hospital Universitario Fundación Santa Fe";

  const initials = [usuario.nombre_usuario[0], usuario.apellidos[0]]
    .filter(Boolean)
    .join("")
    .toUpperCase() || "ME";

  return (
    <header className="sticky top-0 z-20 flex h-[70px] items-center justify-between border-b border-line bg-shell px-8">
      <div>
        <h1 className="font-display text-xl font-bold text-ink">{entidadDisplay}</h1>
        <p className="text-sm text-slate">— Sede Principal, Bogotá</p>
      </div>

      <div className="flex items-center gap-8">
        <div className="text-right">
          <p className="text-sm text-navy-800">{nombreCompleto}</p>
          <p className="text-xs uppercase tracking-[0.6px] text-muted">{rolDisplay}</p>
        </div>
        <div className="flex items-center gap-3">
          <button className="flex size-10 items-center justify-center rounded-xl text-slate transition-colors hover:bg-field">
            <Bell className="size-5" />
          </button>
          <button className="flex size-10 items-center justify-center rounded-xl text-slate transition-colors hover:bg-field">
            <HelpCircle className="size-5" />
          </button>
          <div className="flex size-10 items-center justify-center rounded-xl border border-line bg-navy font-display text-sm font-semibold text-white">
            {initials}
          </div>
        </div>
      </div>
    </header>
  );
}