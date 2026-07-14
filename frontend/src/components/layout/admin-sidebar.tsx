"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import {
  LayoutDashboard,
  Building2,
  Users,
  ScrollText,
  Settings,
  LogOut,
} from "lucide-react";
import { BrandMark } from "@/components/brand";
import { cn } from "@/lib/utils";

const NAV = [
  { href: "/admin/dashboard_admin", label: "Dashboard", icon: LayoutDashboard },
  { href: "/admin/entidades", label: "Entidades de Salud", icon: Building2 },
  { href: "/admin/usuarios", label: "Gestión de Usuarios", icon: Users },
  { href: "/admin/registros", label: "Registros del Sistema", icon: ScrollText },
  { href: "/admin/configuracion", label: "Configuración", icon: Settings },
];

function getInitials(name: string): string {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0].toUpperCase())
    .join("");
}

export function AdminSidebar() {
  const pathname = usePathname();
  const [userName, setUserName] = useState("Administrador");
  const [userInitials, setUserInitials] = useState("AD");

  useEffect(() => {
    try {
      const raw = localStorage.getItem("usuario");
      if (raw) {
        const u = JSON.parse(raw) as { nombre?: string; nombre_completo?: string };
        const name = u.nombre_completo ?? u.nombre ?? "";
        if (name) {
          setUserName(name);
          setUserInitials(getInitials(name));
        }
      }
    } catch {
      // silently ignore
    }
  }, []);

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
              Administrador
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

      {/* Perfil + Cerrar sesión */}
      <div className="border-t border-white/10 p-4">
        <div className="mb-2 flex items-center gap-3 rounded bg-white/5 px-4 py-3">
          <div className="flex size-8 items-center justify-center rounded-lg bg-[#76aecc]/20 font-display text-xs font-semibold text-[#76aecc]">
            {userInitials}
          </div>
          <div className="min-w-0 leading-none">
            <p className="truncate text-sm font-medium text-white">{userName}</p>
            <p className="mt-0.5 text-[10px] uppercase tracking-[0.6px] text-[#91b9cf]">
              System Admin
            </p>
          </div>
        </div>
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
