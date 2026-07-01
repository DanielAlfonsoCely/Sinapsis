"use client";

import { cn } from "@/lib/utils";

interface FieldProps {
  label?: string;
  htmlFor?: string;
  hint?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

/** Envoltorio label + control, con la etiqueta en mayúsculas del diseño. */
export function Field({ label, htmlFor, hint, className, children }: FieldProps) {
  return (
    <div className={cn("flex w-full flex-col gap-1", className)}>
      {(label || hint) && (
        <div className="flex items-center justify-between">
          {label && (
            <label
              htmlFor={htmlFor}
              className="text-xs uppercase tracking-[0.6px] text-label"
            >
              {label}
            </label>
          )}
          {hint}
        </div>
      )}
      {children}
    </div>
  );
}

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  icon?: React.ReactNode;
}

export function Input({ className, icon, ...props }: InputProps) {
  return (
    <div className="relative w-full">
      {icon && (
        <span className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-muted">
          {icon}
        </span>
      )}
      <input
        className={cn(
          "h-11 w-full rounded-[var(--radius)] border border-line bg-field text-base text-navy-800 placeholder:text-muted",
          "px-4 outline-none transition-colors focus:border-teal focus:bg-white focus:ring-2 focus:ring-teal/20",
          icon && "pl-11",
          className,
        )}
        {...props}
      />
    </div>
  );
}
