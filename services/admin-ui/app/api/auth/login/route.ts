import { NextRequest, NextResponse } from "next/server";

const GATEWAY = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export async function POST(req: NextRequest) {
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let res: Response;
  try {
    res = await fetch(`${GATEWAY}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unreachable" }, { status: 502 });
  }

  const json = await res.json();
  if (!res.ok) {
    return NextResponse.json(
      { error: json.message ?? json.error?.message ?? json.error ?? "Login failed" },
      { status: res.status },
    );
  }

  const token = json.access_token as string;
  const isProd = process.env.NODE_ENV === "production";
  const maxAge = 60 * 60; // 1h — matches JWT_ACCESS_EXPIRY_HOURS

  const response = NextResponse.json({ ok: true });
  // httpOnly: browser JS cannot read this — XSS cannot steal the token
  response.cookies.set("gw_token", token, {
    httpOnly: true,
    secure: isProd,
    sameSite: "strict",
    path: "/",
    maxAge,
  });
  // Non-httpOnly hint so client JS can detect auth state without reading the token
  response.cookies.set("gw_auth_hint", "1", {
    httpOnly: false,
    secure: isProd,
    sameSite: "strict",
    path: "/",
    maxAge,
  });
  return response;
}
