import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

export async function POST(req: NextRequest) {
  let body: unknown;
  try { body = await req.json(); } catch {
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
  const data = await upstream.json().catch(() => ({})) as Record<string, unknown>;
  if (!upstream.ok) {
    return NextResponse.json(
      { error: data.error ?? data.message ?? `HTTP ${upstream.status}` },
      { status: upstream.status },
    );
  }
  const token = data.access_token as string;
  const res = NextResponse.json({ ok: true });
  res.cookies.set("buyer_token", token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "strict",
    path: "/",
    maxAge: 60 * 60 * 24,
  });
  return res;
}
