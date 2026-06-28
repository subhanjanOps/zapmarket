import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";
import { cookies } from "next/headers";

export async function POST(req: NextRequest) {
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;
  if (!token) {
    return NextResponse.json({ error: "Not authenticated" }, { status: 401 });
  }

  let body: unknown;
  try { body = await req.json(); } catch { body = {}; }

  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/otp/send`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }

  return NextResponse.json({ ok: upstream.ok }, { status: upstream.status });
}
