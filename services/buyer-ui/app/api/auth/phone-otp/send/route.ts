import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

export async function POST(req: NextRequest) {
  let body: unknown;
  try { body = await req.json(); } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/phone-otp/send`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }
  const data = await upstream.json().catch(() => ({})) as Record<string, unknown>;
  if (!upstream.ok) {
    return NextResponse.json({ error: (data.error ?? `HTTP ${upstream.status}`) as string }, { status: upstream.status });
  }
  return NextResponse.json({ ok: true });
}
