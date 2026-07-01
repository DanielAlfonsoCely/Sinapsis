import { Bell, HelpCircle } from "lucide-react";

interface TopbarProps {
  entity?: string;
  location?: string;
  userName?: string;
  userRole?: string;
}

export function Topbar({
  entity = "Hospital Universitario Fundación Santa Fe",
  location = "Sede Principal, Bogotá",
  userName = "Dr. Camilo Pineda",
  userRole = "Médico General",
}: TopbarProps) {
  return (
    <header className="sticky top-0 z-20 flex h-[70px] items-center justify-between border-b border-line bg-shell px-8">
      <div>
        <h1 className="font-display text-xl font-bold text-ink">{entity}</h1>
        <p className="text-sm text-slate">— {location}</p>
      </div>

      <div className="flex items-center gap-8">
        <div className="text-right">
          <p className="text-sm text-navy-800">{userName}</p>
          <p className="text-xs uppercase tracking-[0.6px] text-muted">
            {userRole}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button className="flex size-10 items-center justify-center rounded-xl text-slate transition-colors hover:bg-field">
            <Bell className="size-5" />
          </button>
          <button className="flex size-10 items-center justify-center rounded-xl text-slate transition-colors hover:bg-field">
            <HelpCircle className="size-5" />
          </button>
          <div className="flex size-10 items-center justify-center rounded-xl border border-line bg-navy font-display text-sm font-semibold text-white">
            CP
          </div>
        </div>
      </div>
    </header>
  );
}
