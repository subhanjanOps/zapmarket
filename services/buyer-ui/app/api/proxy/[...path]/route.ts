import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

async function proxy(req: NextRequest, pathSegments: string[]) {
  // Reject path traversal attempts.
  if (pathSegments.some((s) => s === ".." || s === ".")) {
    return NextResponse.json({ error: "Invalid path" }, { status: 400 });
  }

  const token = req.cookies.get("buyer_token")?.value;
  const upstreamUrl = `${GW}/${pathSegments.join("/")}${req.nextUrl.search}`;
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const ct = req.headers.get("content-type") ?? "";
  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let body: BodyInit | undefined;
  if (hasBody) {
    if (ct.includes("multipart/form-data")) {
      body = await req.formData();
    } else {
      headers["Content-Type"] = ct;
      const buf = await req.arrayBuffer();
      if (buf.byteLength > 0) body = buf;
    }
  }
  const upstream = await fetch(upstreamUrl, {
    method: req.method,
    headers,
    ...(body !== undefined ? { body } : {}),
    cache: "no-store",
  });
  if (upstream.status === 204) return new NextResponse(null, { status: 204 });
  const upstreamCt = upstream.headers.get("content-type") ?? "application/json";
  const responseBody = await upstream.arrayBuffer();
  return new NextResponse(responseBody, {
    status: upstream.status,
    headers: { "Content-Type": upstreamCt },
  });
}

export async function GET(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params; return proxy(req, path);
}
export async function POST(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params; return proxy(req, path);
}
export async function PUT(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params; return proxy(req, path);
}
export async function PATCH(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params; return proxy(req, path);
}
export async function DELETE(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params; return proxy(req, path);
}
