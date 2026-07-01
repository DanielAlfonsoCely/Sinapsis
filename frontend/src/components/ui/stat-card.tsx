import { cn } from "@/lib/utils";

interface StatCardProps {
  label: string;
  value: string | number;
  hint?: string;
  icon?: React.ReactNode;
  valueClassName?: string;
}

export function StatCard({
  label,
  value,
  hint,
  icon,
  valueClassName,
}: StatCardProps) {
  return (
    <div className="flex items-center gap-6 rounded-[var(--radius)] border border-line bg-[#e1f0f3] p-6">
      {icon && (
        <div className="flex size-12 items-center justify-center rounded border border-line bg-white text-teal">
          {icon}
        </div>
      )}
      <div>
        <p className="text-xs uppercase tracking-[0.6px] text-muted">{label}</p>
        <p
          className={cn(
            "font-display text-3xl font-semibold tracking-tight text-ink",
            valueClassName,
          )}
        >
          {value}
        </p>
        {hint && (
          <p className="text-xs tracking-[0.6px] text-label">{hint}</p>
        )}
      </div>
    </div>
  );
}
