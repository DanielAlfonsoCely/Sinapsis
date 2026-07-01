# SINAPSIS — Handoff de implementación desde Figma

Estado del pase Figma → web. Retomar aquí cuando se reponga el cupo del MCP de
Figma (plan Starter llegó al límite de tool calls).

## Datos del archivo Figma

- **File key:** `XGII7Bpw3syFAp6ip6mcVS`
- **URL:** https://www.figma.com/design/XGII7Bpw3syFAp6ip6mcVS/Sinapsis
- Permisos: "cualquier persona con el enlace puede editar" (necesario para el MCP).
- Nota: el archivo original `MKpZwtjbJbixgknAh7SMWJ` NO da acceso al MCP (el plan
  Starter reporta seat=view aunque seas dueño del equipo).

## Herramienta a usar para retomar

Por cada pantalla pendiente:
1. `get_design_context({ nodeId, fileKey: "XGII7Bpw3syFAp6ip6mcVS" })`
   — devuelve código React+Tailwind de referencia + screenshot + assets.
2. Adaptar al design system de este proyecto (ver abajo), NO copiar el Tailwind
   crudo de Figma (usa posiciones absolutas y clases `content-stretch`, etc.).
3. Reemplazar la página placeholder correspondiente.

Ojo: `get_design_context` a veces trunca a 25k tokens. Si pasa, usar
`get_screenshot` para lo visual y `get_metadata` para la estructura del nodo.

## Pantallas — node IDs y ruta destino

| Pantalla Figma                     | node-id (API) | Ruta Next.js          | Estado |
|------------------------------------|---------------|-----------------------|--------|
| Login Principal - SINAPSIS         | `0:795`       | `/login`              | ✅ Hecho (pixel) |
| Verificación MFA - SINAPSIS        | `0:882`       | `/mfa`                | ✅ Hecho (pixel) |
| Dashboard Medico                   | `0:3`         | `/dashboard`          | ✅ Hecho (pixel) |
| Pestaña Pacientes Médico           | `0:338`       | `/pacientes`          | ⏳ Placeholder |
| Agenda Semanal                     | `0:529`       | `/agenda`             | ⏳ Placeholder |
| Agenda dia                         | `0:952`       | `/agenda/[dia]` o `/agenda?vista=dia` | ⏳ Placeholder |
| Pantalla Consulta                  | `0:1247`      | `/consulta`           | ⏳ Placeholder |
| Análisis IA                        | `0:1515`      | `/analisis-ia`        | ⏳ Placeholder |
| Gestión de Fórmulas Médicas        | `0:1803`      | `/formulas`           | ⏳ Placeholder |
| Historia Clinica                   | `0:2066`      | `/historia-clinica`   | ⏳ Placeholder |

(`/auditoria` existe como placeholder pero no tiene pantalla en Figma todavía.)

## Design system ya establecido (usar estos tokens)

Definidos en `src/app/globals.css` con `@theme` de Tailwind v4. Clases directas:

- Colores: `navy` `#001530`, `navy-800` `#001a42`, `teal` `#286580`,
  `slate` (cuerpo), `muted` (placeholder), `label` (mayúsculas), `line`
  (bordes), `field` (fondo input `#f1f3ff`), `surface`, `shell`, `canvas`.
- Estados: `success`, `warning`, `danger`, `info`.
- Fuentes: `font-display` (Sora, títulos), `font-sans` (DM Sans, cuerpo).
- Radio: `rounded-[var(--radius)]` = 8px.

Componentes reutilizables en `src/components/`:
- `ui/button.tsx` — `<Button variant size asChild>`
- `ui/input.tsx` — `<Field label hint>` + `<Input icon>`
- `ui/card.tsx` — `Card`, `CardHeader`, `CardTitle`, `CardContent`
- `ui/badge.tsx` — `<Badge tone>` (neutral/success/warning/danger/info/navy)
- `ui/stat-card.tsx` — `<StatCard label value hint icon>`
- `brand.tsx` — `BrandMark`, `Wordmark`
- `layout/sidebar.tsx` (nav con estado activo), `layout/topbar.tsx`
- `coming-soon.tsx` — placeholder a reemplazar

Iconos: `lucide-react` (no descargar SVGs de Figma).

## Convenciones del proyecto

- Next.js 16 + React 19 + Tailwind v4. Leer `AGENTS.md`: hay breaking changes;
  `params`/`searchParams` son Promises (hay que `await`).
- Rutas con datos/estado de cliente → `"use client"`.
- Layout de dashboard ya envuelve con sidebar+topbar; las páginas solo aportan
  el contenido central.

## Reglas de negocio a respetar en las pantallas pendientes

(De los PDFs en `../Documentos/`)
- **Consulta (`/consulta`):** el médico DEBE registrar pre-diagnóstico ANTES de
  habilitar las herramientas de IA (RN-007). Bloquear IA sin pre-diagnóstico.
- **Análisis IA (`/analisis-ia`):** mostrar modelo usado (ej. "MONAI v2.0") y el
  disclaimer obligatorio "Esta sugerencia NO es un diagnóstico. El médico es
  responsable de la interpretación final" (RN-008). Nunca emite diagnóstico.
- **Fórmulas (`/formulas`):** estados Vigente / Reclamado / Vencido / Atrasado.
- **Historia Clínica:** registros incrementales (no se sobrescribe), con fórmulas
  y exámenes/imágenes asociados; toda acción va a bitácora de auditoría inmutable.

## Cómo correr

```bash
cd frontend
npm run dev   # http://localhost:3000  (redirige / → /login)
```
