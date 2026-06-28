import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

export async function POST(req: NextRequest) {
  let formData: FormData;
  try {
    formData = await req.formData();
  } catch {
    return NextResponse.json({ error: "Invalid form data" }, { status: 400 });
  }
  let upstream: Response;
  try {
    upstream = await fetch(`${GW}/v1/auth/profile/picture`, {
      method: "POST",
      body: formData,
    });
  } catch {
    return NextResponse.json({ error: "Gateway unavailable" }, { status: 502 });
  }
  const data = await upstream.json().catch(() => ({})) as Record<string, unknown>;
  if (!upstream.ok) {
    return NextResponse.json({ error: (data.error ?? `HTTP ${upstream.status}`) as string }, { status: upstream.status });
  }
  return NextResponse.json({ pfp_url: data.pfp_url });
}
