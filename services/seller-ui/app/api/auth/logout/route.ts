import { NextResponse } from "next/server";

export async function POST() {
  const res = NextResponse.json({ ok: true });
  res.cookies.delete("seller_token");
  res.cookies.delete("seller_auth_hint");
  return res;
}
