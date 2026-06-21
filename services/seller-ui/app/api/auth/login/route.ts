import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export async function POST(req: NextRequest) {
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }
  let data: Record<string, unknown>;
  try {
    data = await upstream.json();
  } catch {
    return NextResponse.json({ error: "Invalid upstream response" }, { status: 502 });
  }
  if (!upstream.ok) {
    return NextResponse.json(
      { error: data.error ?? data.message ?? `HTTP ${upstream.status}` },
      { status: upstream.status },
    );
  }
  const token = data.access_token as string;
  const res = NextResponse.json({ ok: true });
  res.cookies.set("seller_token", token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "strict",
    path: "/",
    maxAge: 60 * 60, // 1 hour — matches JWT_ACCESS_EXPIRY_HOURS
  });
  // Non-httpOnly hint so client JS can detect login state without reading the token
  res.cookies.set("seller_auth_hint", "1", {
    httpOnly: false,
    secure: process.env.NODE_ENV === "production",
    sameSite: "strict",
    path: "/",
    maxAge: 60 * 60,
  });
  return res;
}
