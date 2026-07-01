import type { Metadata } from "next";
import { Sora, DM_Sans } from "next/font/google";
import "./globals.css";

const sora = Sora({
  variable: "--font-sora",
  subsets: ["latin"],
  display: "swap",
});

const dmSans = DM_Sans({
  variable: "--font-dm-sans",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "SINAPSIS — Sistema Inteligente de Análisis de Patrones de Salud",
  description:
    "Plataforma de historia clínica electrónica unificada con apoyo de IA para profesionales de la salud.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="es" className={`${sora.variable} ${dmSans.variable} h-full`}>
      <body className="min-h-full">{children}</body>
    </html>
  );
}
