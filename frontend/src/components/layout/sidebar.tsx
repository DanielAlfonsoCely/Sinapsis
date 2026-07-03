"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Users,
  CalendarDays,
  Pill,
  BrainCircuit,
  ShieldCheck,
  LogOut,
} from "lucide-react";
import { BrandMark } from "@/components/brand";
import { cn } from "@/lib/utils";

// Nota: "Nueva Consulta" e "Historia Clínica" no van en el menú porque dependen
// del paciente seleccionado. Se acceden desde la columna "Acción" de /pacientes.
const NAV = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/pacientes", label: "Pacientes", icon: Users },
  { href: "/agenda", label: "Agenda", icon: CalendarDays },
  { href: "/formulas", label: "Fórmulas Médicas", icon: Pill },
  { href: "/analisis-ia", label: "Análisis IA", icon: BrainCircuit },
  { href: "/auditoria", label: "Auditoría", icon: ShieldCheck },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 z-30 flex w-64 flex-col justify-between bg-gradient-to-b from-[#001e4b] to-navy">
      <div className="flex flex-col gap-6 p-4">
        {/* Marca */}
        <div className="flex items-center gap-2.5 px-2 py-4">
          <BrandMark className="size-7 text-[#76aecc]" />
          <div className="leading-none">
            <p className="font-display text-xl font-bold tracking-tight text-white">
              SINAPSIS
            </p>
            <p className="mt-1 text-[10px] uppercase tracking-[1px] text-[#91b9cf]">
              Sistemas Médicos
            </p>
          </div>
        </div>

        {/* Navegación */}
        <nav className="flex flex-col gap-1">
          {NAV.map(({ href, label, icon: Icon }) => {
            const active =
              pathname === href || pathname.startsWith(href + "/");
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
        <Link
          href="/login"
          className="flex items-center gap-3 rounded px-4 py-3 text-sm text-[#d9534f] transition-colors hover:bg-white/5"
        >
          <LogOut className="size-5" />
          Cerrar Sesión
        </Link>
      </div>
    </aside>
  );
}
