const GATEWAY = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

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
};

async function apiFetch<T>(path: string, token: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${GATEWAY}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...(init?.headers ?? {}),
    },
  });
  const json = await res.json();
  if (!res.ok || !json.success) {
    throw new Error(json.error?.message ?? `HTTP ${res.status}`);
  }
  return json.data as T;
}

export async function login(email: string, password: string): Promise<string> {
  const res = await fetch(`${GATEWAY}/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const json = await res.json();
  if (!res.ok) throw new Error(json.error?.message ?? "Login failed");
  return json.access_token as string;
}

// ── Routes ────────────────────────────────────────────────────────────────────

export const getRoutes = (token: string) =>
  apiFetch<{ routes: Route[] }>("/gateway/v1/routes", token).then((d) => d.routes ?? []);

export const createRoute = (token: string, body: Omit<Route, "id" | "created_at" | "updated_at" | "enabled">) =>
  apiFetch<{ id: string }>("/gateway/v1/routes", token, {
    method: "POST",
    body: JSON.stringify(body),
  });

export const updateRoute = (token: string, id: string, body: Partial<Route>) =>
  apiFetch<{ updated: boolean }>(`/gateway/v1/routes/${id}`, token, {
    method: "PUT",
    body: JSON.stringify(body),
  });

export const deleteRoute = (token: string, id: string) =>
  apiFetch<{ disabled: boolean }>(`/gateway/v1/routes/${id}`, token, {
    method: "DELETE",
  });

// ── Audit ─────────────────────────────────────────────────────────────────────

export type AuditParams = {
  event?: string;
  user_id?: string;
  from?: string;
  to?: string;
  limit?: number;
};

export const getAudit = (token: string, params: AuditParams = {}) => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => v !== undefined && q.set(k, String(v)));
  return apiFetch<{ entries: AuditEntry[]; count: number }>(
    `/gateway/v1/audit?${q}`,
    token
  ).then((d) => d.entries ?? []);
};

// ── Registry ──────────────────────────────────────────────────────────────────

export const getRegistry = (token: string) =>
  apiFetch<{ instances: RegistryInstance[]; count: number }>("/gateway/v1/registry", token).then(
    (d) => d.instances ?? []
  );
