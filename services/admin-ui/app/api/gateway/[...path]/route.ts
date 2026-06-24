import { NextRequest, NextResponse } from "next/server";

const GATEWAY = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

type Params = { params: Promise<{ path: string[] }> };

async function proxy(req: NextRequest, { params }: Params) {
  const token = req.cookies.get("gw_token")?.value;
  if (!token) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { path } = await params;

  // Reject path traversal attempts.
  if (path.some((s) => s === ".." || s === ".")) {
    return NextResponse.json({ error: "Invalid path" }, { status: 400 });
  }

  const target = `${GATEWAY}/${path.join("/")}${req.nextUrl.search}`;

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };

  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let body: string | undefined;
  if (hasBody) {
    body = await req.text();
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, { method: req.method, headers, body });
  } catch {
    return NextResponse.json({ error: "Gateway unreachable" }, { status: 502 });
  }

  const contentType = upstream.headers.get("Content-Type") ?? "application/json";
  const data = await upstream.text();
  return new NextResponse(data, {
    status: upstream.status,
    headers: { "Content-Type": contentType },
  });
}

export { proxy as GET, proxy as POST, proxy as PUT, proxy as DELETE, proxy as PATCH };
