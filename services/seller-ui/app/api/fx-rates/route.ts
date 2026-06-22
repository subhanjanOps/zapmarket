import { NextResponse } from "next/server";

const CACHE_DURATION = 3600; // 1 hour

export async function GET() {
  const res = await fetch("https://api.frankfurter.app/latest?from=USD", {
    next: { revalidate: CACHE_DURATION },
  });
  if (!res.ok) {
    return NextResponse.json({ error: "Failed to fetch rates" }, { status: 502 });
  }
  const data = await res.json();
  return NextResponse.json(data, {
    headers: { "Cache-Control": `public, max-age=${CACHE_DURATION}, stale-while-revalidate=600` },
  });
}
