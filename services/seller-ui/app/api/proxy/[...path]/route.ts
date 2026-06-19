import { NextRequest, NextResponse } from "next/server";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

async function proxy(req: NextRequest, pathSegments: string[]) {
  const token = req.cookies.get("seller_token")?.value;
  const upstreamUrl = `${GW}/${pathSegments.join("/")}${req.nextUrl.search}`;

  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  let body: ArrayBuffer | undefined;
  const ct = req.headers.get("content-type");
  if (ct) {
    headers["Content-Type"] = ct;
    body = await req.arrayBuffer();
  }

  const options: RequestInit = {
    method: req.method,
    headers,
    cache: "no-store",
  };

  if (body != undefined && body?.byteLength > 0) {
    options.body = body;
  }

  const upstream = await fetch(upstreamUrl, options);

  if (upstream.status === 204) {
    return new NextResponse(null, { status: 204 });
  }

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
