import { ShieldCheck } from "lucide-react";

interface AdminTopbarProps {
  systemStatus?: "OPTIMAL" | "DEGRADED" | "OFFLINE";
  entityContext?: string;
}

export function AdminTopbar({
  systemStatus = "OPTIMAL",
  entityContext = "ADMINISTRADOR SINAPSIS",
}: AdminTopbarProps) {
  const statusColors = {
    OPTIMAL: "bg-success/10 border-success/30 text-success",
    DEGRADED: "bg-warning/10 border-warning/30 text-warning",
    OFFLINE: "bg-danger/10 border-danger/30 text-danger",
  };

  return (
    <header className="sticky top-0 z-20 flex h-[70px] items-center justify-end border-b border-line bg-shell px-8">

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
      </div>
    </header>
  );
}
