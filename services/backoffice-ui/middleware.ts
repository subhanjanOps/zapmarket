import { NextRequest, NextResponse } from "next/server";

const PUBLIC_PREFIXES = ["/login", "/api/auth/"];

function isTokenExpired(jwt: string): boolean {
  try {
    const payload = JSON.parse(atob(jwt.split(".")[1])) as { exp?: number };
    if (typeof payload.exp !== "number") return false;
    return Date.now() / 1000 > payload.exp;
  } catch {
    return true;
  }
}

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;

  if (PUBLIC_PREFIXES.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  const token = req.cookies.get("bo_token");
  if (!token || isTokenExpired(token.value)) {
    if (pathname.startsWith("/api/")) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }
    return NextResponse.redirect(new URL("/login", req.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
