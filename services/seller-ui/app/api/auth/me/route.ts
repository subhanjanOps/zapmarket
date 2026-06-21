import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export async function GET(req: NextRequest) {
  const token = req.cookies.get("seller_token")?.value;
  if (!token) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
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
      { error: data.error ?? `HTTP ${upstream.status}` },
      { status: upstream.status },
    );
  }
  return NextResponse.json(data);
}
