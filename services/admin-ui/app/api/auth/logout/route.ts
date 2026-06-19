import { NextResponse } from "next/server";

export async function POST() {
  const response = NextResponse.json({ ok: true });
  response.cookies.delete("gw_token");
  response.cookies.delete("gw_auth_hint");
  return response;
}
