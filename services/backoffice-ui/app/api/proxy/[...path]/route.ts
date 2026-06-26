import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? "http://localhost:8000";

const ALLOWED_PREFIXES = [
  "v1/products",
  "v1/skus",
  "v1/categories",
  "v1/orders",
  "v1/users",
  "v1/inventory",
  "v1/payments",
  "v1/images",
  "v1/campaigns",
  "v1/currencies",
];

async function proxy(req: NextRequest, pathSegments: string[]) {
  // Reject path traversal attempts.
  if (pathSegments.some((s) => s === ".." || s === ".")) {
    return NextResponse.json({ error: "Invalid path" }, { status: 400 });
  }

  const joined = pathSegments.join("/");
  if (!ALLOWED_PREFIXES.some((p) => joined === p || joined.startsWith(p + "/"))) {
    return NextResponse.json({ error: "Forbidden" }, { status: 403 });
  }

  const token = req.cookies.get("bo_token")?.value;
  if (!token) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const upstreamUrl = `${GW}/${pathSegments.join("/")}${req.nextUrl.search}`;

  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const ct = req.headers.get("content-type");
  let body: ArrayBuffer | undefined;
  if (ct) {
    headers["Content-Type"] = ct;
    body = await req.arrayBuffer();
  }

  const options: RequestInit = { method: req.method, headers };
  if (body !== undefined && body.byteLength > 0) options.body = body;

  let upstream: Response;
  try {
    upstream = await fetch(upstreamUrl, options);
  } catch {
    return NextResponse.json({ error: "Gateway unreachable" }, { status: 502 });
  }

  if (upstream.status === 204) return new NextResponse(null, { status: 204 });

  const upstreamCt = upstream.headers.get("content-type") ?? "application/json";
  const responseBody = await upstream.arrayBuffer();
  return new NextResponse(responseBody, {
    status: upstream.status,
    headers: { "Content-Type": upstreamCt },
  });
}

export async function GET(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  return proxy(req, path);
}
export async function POST(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  return proxy(req, path);
}
export async function PUT(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  return proxy(req, path);
}
export async function PATCH(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  return proxy(req, path);
}
export async function DELETE(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  return proxy(req, path);
}
