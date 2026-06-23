import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export async function GET(req: NextRequest) {
  const token = req.cookies.get("buyer_token")?.value;
  if (!token) return NextResponse.json({ user: null });
  try {
    const upstream = await fetch(`${GW}/v1/users/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (!upstream.ok) return NextResponse.json({ user: null });
    const data = await upstream.json();
    return NextResponse.json({ user: { id: data.id, email: data.email, role: data.role } });
  } catch {
    return NextResponse.json({ user: null });
  }
}
