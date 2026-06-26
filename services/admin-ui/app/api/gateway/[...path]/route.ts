import { NextRequest, NextResponse } from "next/server";

const GATEWAY = process.env.GATEWAY_URL ?? "http://localhost:8000";
const MAX_BODY_BYTES = 10 * 1024 * 1024; // 10 MB

const ALLOWED_PREFIXES = [
  "gateway/v1/stats",
  "gateway/v1/routes",
  "gateway/v1/audit",
  "gateway/v1/registry",
  "gateway/v1/metrics",
  "gateway/v1/probe",
  "gateway/v1/blocklist",
  "v1/currencies",
  "v1/admin/currencies",
];

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

  const joined = path.join("/");
  if (!ALLOWED_PREFIXES.some((p) => joined === p || joined.startsWith(p + "/"))) {
    return NextResponse.json({ error: "Forbidden" }, { status: 403 });
  }

  const target = `${GATEWAY}/${path.join("/")}${req.nextUrl.search}`;

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
  };

  const ct = req.headers.get("content-type") ?? "";
  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let body: BodyInit | undefined;
  if (hasBody) {
    if (ct.includes("multipart/form-data")) {
      body = await req.formData();
    } else {
      if (ct) headers["Content-Type"] = ct;
      const text = await req.text();
      if (text.length > MAX_BODY_BYTES) {
        return NextResponse.json({ error: "Request body too large" }, { status: 413 });
      }
      body = text;
    }
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
