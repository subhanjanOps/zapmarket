import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export async function POST(req: NextRequest) {
  let body: Record<string, unknown>;
  try { body = await req.json(); } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
  body = { ...body, role: "buyer" };
  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/register`, {
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
  return NextResponse.json({ ok: true }, { status: 201 });
}
