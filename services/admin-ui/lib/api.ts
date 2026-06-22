// All requests go through the Next.js BFF proxy at /api/gateway/...
// The proxy reads the httpOnly gw_token cookie and attaches Authorization header.

export type Route = {
  id: string;
  path_prefix: string;
  upstream: string;
  auth_mode: "none" | "required" | "method_split";
  strip_prefix: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type AuditEntry = {
  id: number;
  ts: string;
  request_id: string;
  user_id?: string;
  ip: string;
  method: string;
  path: string;
  upstream: string;
  status_code: number;
  event: string;
  detail: string;
};

export type RegistryInstance = {
  service: string;
  instance_id: string;
  addr: string;
  started_at: string;
  healthy: boolean;
};

export type UpstreamMetric = {
  name: string;
  total: number;
  errors: number;
  req_per_min: number;
  error_rate: number;
};

export type BlocklistEntry = {
  ip: string;
  reason: string;
  blocked_at: string;
};

export type Stats = {
  active_routes: number;
  total_routes: number;
  live_instances: number;
  live_services: number;
  recent_audit: AuditEntry[];
  upstreams: UpstreamMetric[];
};

export type ProbeResult = {
  status: number;
  headers: Record<string, string>;
  body: string;
  latency_ms: number;
  upstream: string;
  addr: string;
};

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/gateway${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (!res.ok) {
    const ct = res.headers.get("Content-Type") ?? "";
    if (ct.includes("application/json")) {
      const json = await res.json();
      throw new Error(json.error?.message ?? json.error ?? `HTTP ${res.status}`);
    }
    throw new Error(`HTTP ${res.status}`);
  }

  const json = await res.json();
  if (!json.success) {
    throw new Error(json.error?.message ?? "Unknown error");
  }
  return json.data as T;
}

// ── Auth ──────────────────────────────────────────────────────────────────────

/** Calls the BFF login route, which sets httpOnly cookies on success. */
export async function login(email: string, password: string): Promise<void> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const json = await res.json();
  if (!res.ok) throw new Error(json.error ?? "Login failed");
}

// ── Stats ─────────────────────────────────────────────────────────────────────

export const getStats = () =>
  apiFetch<Stats>("/gateway/v1/stats");

// ── Routes ────────────────────────────────────────────────────────────────────

export const getRoutes = () =>
  apiFetch<{ routes: Route[] }>("/gateway/v1/routes").then((d) => d.routes ?? []);

export const createRoute = (body: Omit<Route, "id" | "created_at" | "updated_at" | "enabled">) =>
  apiFetch<{ id: string }>("/gateway/v1/routes", {
    method: "POST",
    body: JSON.stringify(body),
  });

export const updateRoute = (id: string, body: Partial<Route>) =>
  apiFetch<{ updated: boolean }>(`/gateway/v1/routes/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

export const deleteRoute = (id: string) =>
  apiFetch<{ disabled: boolean }>(`/gateway/v1/routes/${id}`, {
    method: "DELETE",
  });

// ── Audit ─────────────────────────────────────────────────────────────────────

export type AuditParams = {
  event?: string;
  user_id?: string;
  from?: string;
  to?: string;
  limit?: number;
  after_id?: number;
};

export const getAudit = (params: AuditParams = {}) => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => v !== undefined && q.set(k, String(v)));
  return apiFetch<{ entries: AuditEntry[]; count: number }>(
    `/gateway/v1/audit?${q}`,
  ).then((d) => d.entries ?? []);
};

// ── Registry ──────────────────────────────────────────────────────────────────

export const getRegistry = () =>
  apiFetch<{ instances: RegistryInstance[]; count: number }>("/gateway/v1/registry").then(
    (d) => d.instances ?? [],
  );

// ── Metrics ───────────────────────────────────────────────────────────────────

export const getMetrics = () =>
  apiFetch<{ upstreams: UpstreamMetric[] }>("/gateway/v1/metrics").then(
    (d) => d.upstreams ?? [],
  );

// ── Probe ─────────────────────────────────────────────────────────────────────

export const probeRoute = (
  req: { method: string; path: string; token?: string; headers?: Record<string, string>; body?: string },
) =>
  apiFetch<ProbeResult>("/gateway/v1/probe", {
    method: "POST",
    body: JSON.stringify(req),
  });

// ── Blocklist ─────────────────────────────────────────────────────────────────

export const getBlocklist = () =>
  apiFetch<{ blocked: BlocklistEntry[]; count: number }>("/gateway/v1/blocklist").then(
    (d) => d.blocked ?? [],
  );

export const blockIP = (ip: string, reason?: string) =>
  apiFetch<{ blocked: boolean }>("/gateway/v1/blocklist", {
    method: "POST",
    body: JSON.stringify({ ip, reason: reason ?? "" }),
  });

export const unblockIP = (ip: string) =>
  apiFetch<{ removed: boolean }>(`/gateway/v1/blocklist/${encodeURIComponent(ip)}`, {
    method: "DELETE",
  });

// ── Currencies ────────────────────────────────────────────────────────────────

export type Currency = {
  code: string;
  name: string;
  flag: string;
  decimals: number;
  enabled: boolean;
};

export async function getCurrencies(): Promise<Currency[]> {
  const res = await fetch("/api/gateway/api/v1/currencies");
  if (!res.ok) throw new Error(`getCurrencies: ${res.status}`);
  return res.json();
}

export async function toggleCurrency(code: string, enabled: boolean): Promise<void> {
  const res = await fetch(`/api/gateway/api/v1/admin/currencies/${code}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) throw new Error(`toggleCurrency: ${res.status}`);
}
