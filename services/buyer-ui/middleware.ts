import { NextRequest, NextResponse } from "next/server";

const PROTECTED = ["/checkout", "/account/"];
const PUBLIC_API = ["/api/auth/", "/api/proxy/v1/auth/"];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (PUBLIC_API.some((p) => pathname.startsWith(p))) return NextResponse.next();
  if (PROTECTED.some((p) => pathname.startsWith(p))) {
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
