import { NextRequest, NextResponse } from "next/server";

// Routes that require a valid session cookie.
const PROTECTED = ["/checkout", "/account"];
// Routes that are always public (no auth check).
const PUBLIC_PREFIXES = ["/login", "/register", "/api/auth/", "/api/proxy/v1/auth/", "/products", "/blog", "/deals", "/glossary", "/cart"];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (PUBLIC_PREFIXES.some((p) => pathname === p || pathname.startsWith(p + "/") || pathname === p)) return NextResponse.next();
  if (PROTECTED.some((p) => pathname === p || pathname.startsWith(p + "/") || pathname.startsWith(p))) {
    const token = req.cookies.get("buyer_token");
    if (!token) {
      if (pathname.startsWith("/api/")) {
        return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
      }
      const url = req.nextUrl.clone();
      url.pathname = "/login";
      url.searchParams.set("next", pathname);
      return NextResponse.redirect(url);
    }
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
