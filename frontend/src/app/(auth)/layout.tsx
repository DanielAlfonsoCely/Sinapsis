import { HelpCircle } from "lucide-react";
import { Wordmark } from "@/components/brand";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-full flex-col bg-canvas">
      {/* Header de identidad institucional */}
      <header className="flex h-16 items-center justify-center border-b border-line bg-shell px-8">
        <div className="flex w-full max-w-[1440px] items-center justify-between">
          <div className="flex items-center gap-4">
            <Wordmark />
            <span className="h-6 w-px bg-line" />
            <span className="font-display text-xl font-semibold text-navy-800">
              Centro Médico Central
            </span>
          </div>
          <button className="flex items-center gap-1.5 text-sm text-slate transition-colors hover:text-teal">
            <HelpCircle className="size-5" />
            Soporte Técnico
          </button>
        </div>
      </header>

      {/* Lienzo con decoración difuminada */}
      <main className="relative flex flex-1 items-center justify-center overflow-hidden p-8">
        <div className="pointer-events-none absolute inset-0 opacity-20">
          <div className="absolute -left-20 top-52 size-96 rounded-xl bg-glow-1 blur-[50px]" />
          <div className="absolute -right-20 bottom-52 size-96 rounded-xl bg-glow-2 blur-[50px]" />
        </div>
        <div className="relative z-10 w-full max-w-md">{children}</div>
      </main>
    </div>
  );
}
