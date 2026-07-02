import { Bell, Search, ShieldCheck } from "lucide-react";

interface AdminTopbarProps {
  systemStatus?: "OPTIMAL" | "DEGRADED" | "OFFLINE";
  entityContext?: string;
}

export function AdminTopbar({
  systemStatus = "OPTIMAL",
  entityContext = "ST. MARINA IPS — ACTIVA",
}: AdminTopbarProps) {
  const statusColors = {
    OPTIMAL: "bg-success/10 border-success/30 text-success",
    DEGRADED: "bg-warning/10 border-warning/30 text-warning",
    OFFLINE: "bg-danger/10 border-danger/30 text-danger",
  };

  return (
    <header className="sticky top-0 z-20 flex h-[70px] items-center justify-between border-b border-line bg-shell px-8">
      {/* Buscador */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted" />
        <input
          type="text"
          placeholder="Buscar recursos del sistema..."
          className="h-10 w-72 rounded border border-line bg-field pl-9 pr-4 text-sm text-slate placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-teal"
        />
      </div>

      {/* Derecha */}
      <div className="flex items-center gap-6">
        {/* Estado del sistema */}
        <div
          className={`flex items-center gap-2 rounded-xl border px-4 py-1.5 text-xs font-medium uppercase tracking-[0.6px] ${statusColors[systemStatus]}`}
        >
          <ShieldCheck className="size-3.5" />
          {entityContext}
        </div>

        {/* Acciones */}
        <div className="flex items-center gap-2">
          <button className="relative flex size-10 items-center justify-center rounded-xl text-slate transition-colors hover:bg-field">
            <Bell className="size-5" />
            {/* Badge de alerta */}
            <span className="absolute right-2 top-2 size-2 rounded-full bg-danger" />
          </button>
        </div>
      </div>
    </header>
  );
}
