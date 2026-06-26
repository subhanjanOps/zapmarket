import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

export async function POST(req: NextRequest) {
  const token = req.cookies.get("buyer_token")?.value;

  // Best-effort server-side revocation
  if (token) {
    try {
      await fetch(`${GW}/v1/auth/logout`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        signal: AbortSignal.timeout(3000),
      });
    } catch {
      // Non-fatal — client cookie will be cleared regardless
    }
  }

  const res = NextResponse.json({ ok: true });
  res.cookies.set("buyer_token", "", {
    maxAge: 0,
    path: "/",
    httpOnly: true,
    sameSite: "strict",
  });
  return res;
}
