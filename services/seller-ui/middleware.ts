import { NextRequest, NextResponse } from "next/server";

const PUBLIC_PREFIXES = ["/login", "/register", "/api/auth/", "/api/proxy/v1/auth/"];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;

  if (PUBLIC_PREFIXES.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  // Only check for cookie presence — signature verification happens at the gateway
  // via ValidateToken gRPC. Client-side JWT decoding is not a security check.
  const token = req.cookies.get("seller_token");
  if (!token) {
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
