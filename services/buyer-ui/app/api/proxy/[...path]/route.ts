import { NextRequest, NextResponse } from "next/server";
import { GW } from "@/lib/gateway";

const MAX_BODY_BYTES = 10 * 1024 * 1024; // 10 MB

async function proxy(req: NextRequest, pathSegments: string[]) {
  // Reject path traversal attempts.
  if (pathSegments.some((s) => s === ".." || s === ".")) {
    return NextResponse.json({ error: "Invalid path" }, { status: 400 });
  }

  const joinedPath = "/" + pathSegments.join("/");
  const ALLOWED_PREFIXES = [
    "/v1/orders",
    "/v1/products",
    "/v1/skus",
    "/v1/categories",
    "/v1/campaigns",
    "/v1/users",
    "/v1/cart",
    "/v1/images",
    "/v1/reviews",
    "/v1/wishlist",
    "/v1/search",
  ];
  if (!ALLOWED_PREFIXES.some(p => joinedPath.startsWith(p))) {
    return NextResponse.json({ error: "Forbidden" }, { status: 403 });
  }

  const token = req.cookies.get("buyer_token")?.value;
  const upstreamUrl = `${GW}/${pathSegments.join("/")}${req.nextUrl.search}`;
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const FORWARDED_HEADERS = ["idempotency-key", "x-request-id", "accept-language", "x-correlation-id"];
  for (const h of FORWARDED_HEADERS) {
    const v = req.headers.get(h);
    if (v) headers[h] = v;
  }
  const ct = req.headers.get("content-type") ?? "";
  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let body: BodyInit | undefined;
  if (hasBody) {
    if (ct.includes("multipart/form-data")) {
      body = await req.formData();
    } else {
      headers["Content-Type"] = ct;
      const buf = await req.arrayBuffer();
      if (buf.byteLength > MAX_BODY_BYTES) {
        return NextResponse.json({ error: "Request body too large" }, { status: 413 });
      }
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
