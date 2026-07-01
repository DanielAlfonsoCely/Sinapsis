import { cn } from "@/lib/utils";

/** Marca SINAPSIS: ícono de red neuronal + wordmark. */
export function BrandMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      className={cn("size-7", className)}
      aria-hidden
    >
      <path
        d="M16 3c-3.6 0-6.5 2.6-6.9 6C6.2 9.6 4 12.2 4 15.4c0 2.3 1.2 4.4 3 5.6.2 3.3 3 6 6.4 6 1.2 0 2.3-.3 3.2-.9.9.6 2 .9 3.2.9 3.4 0 6.2-2.7 6.4-6 1.8-1.2 3-3.3 3-5.6 0-3.2-2.2-5.8-5.1-6.4C23.5 5.6 20.6 3 17 3h-1Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <circle cx="12" cy="13" r="1.6" fill="currentColor" />
      <circle cx="20" cy="11" r="1.6" fill="currentColor" />
      <circle cx="21" cy="19" r="1.6" fill="currentColor" />
      <circle cx="13" cy="21" r="1.6" fill="currentColor" />
      <path
        d="M12 13l8-2M20 11l1 8M21 19l-8 2M13 21l-1-8"
        stroke="currentColor"
        strokeWidth="1.2"
        opacity="0.6"
      />
    </svg>
  );
}

export function Wordmark({
  className,
  subtitle = false,
}: {
  className?: string;
  subtitle?: boolean;
}) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <BrandMark className="text-teal" />
      <div className="leading-none">
        <span className="font-display text-xl font-extrabold tracking-tight text-ink">
          SINAPSIS
        </span>
        {subtitle && (
          <p className="mt-0.5 text-[10px] uppercase tracking-[0.2px] text-label">
            Análisis Inteligente de Patrones de Salud
          </p>
        )}
      </div>
    </div>
  );
}
