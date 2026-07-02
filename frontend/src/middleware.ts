import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const token = request.cookies.get("token")?.value;

  const isDashboardRoute = request.nextUrl.pathname.startsWith("/dashboard") ||
    request.nextUrl.pathname.startsWith("/agenda") ||
    request.nextUrl.pathname.startsWith("/analisis-ia") ||
    request.nextUrl.pathname.startsWith("/auditoria") ||
    request.nextUrl.pathname.startsWith("/consulta") ||
    request.nextUrl.pathname.startsWith("/formulas") ||
    request.nextUrl.pathname.startsWith("/historia-clinica") ||
    request.nextUrl.pathname.startsWith("/pacientes");

  if (isDashboardRoute && !token) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/agenda/:path*",
    "/analisis-ia/:path*",
    "/auditoria/:path*",
    "/consulta/:path*",
    "/formulas/:path*",
    "/historia-clinica/:path*",
    "/pacientes/:path*",
  ],
};