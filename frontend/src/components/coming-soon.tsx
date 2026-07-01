import { Construction } from "lucide-react";
import { Card } from "@/components/ui/card";

/** Placeholder para rutas cuyo diseño aún no se ha extraído de Figma. */
export function ComingSoon({ title }: { title: string }) {
  return (
    <Card className="flex flex-col items-center gap-3 p-12 text-center">
      <div className="flex size-14 items-center justify-center rounded-full bg-field text-teal">
        <Construction className="size-7" />
      </div>
      <h2 className="font-display text-xl font-semibold text-ink">{title}</h2>
      <p className="max-w-md text-sm text-slate">
        Pantalla pendiente de implementar desde el diseño de Figma. La base
        (layout, navegación y design system) ya está lista.
      </p>
    </Card>
  );
}
