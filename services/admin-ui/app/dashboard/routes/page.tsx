"use client";
import { useState } from "react";
import { getRoutes, createRoute, updateRoute, deleteRoute, probeRoute, Route, ProbeResult } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { Plus, Pencil, Trash2, FlaskConical } from "lucide-react";
import { SkeletonTableCard } from "@/app/components/Skeleton";

const EMPTY: Omit<Route, "id" | "created_at" | "updated_at" | "enabled"> = {
  path_prefix: "",
  upstream: "",
  auth_mode: "required",
  strip_prefix: false,
};

// ── Route Tester Modal ────────────────────────────────────────────────────────

function ProbeModal({ route, onClose }: { route: Route; onClose: () => void }) {
  const [method, setMethod] = useState("GET");
  const [path, setPath] = useState(route.path_prefix);
  const [body, setBody] = useState("");
  const [probeToken, setProbeToken] = useState("");
  const [result, setResult] = useState<ProbeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");

  async function run() {
    setLoading(true);
    setErr("");
    setResult(null);
    try {
      const r = await probeRoute({ method, path, token: probeToken || undefined, body: body || undefined });
      setResult(r);
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ width: "min(620px, 100%)" }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">
            Route Tester —{" "}
            <span className="mono" style={{ color: "var(--accent)" }}>{route.path_prefix}</span>
          </h2>
          <p className="modal-subtitle">Send a request through the gateway to this route</p>
        </div>
        <div className="modal-body">
          <div className="form-group">
            <div style={{ display: "grid", gridTemplateColumns: "6.5rem 1fr", gap: "0.5rem" }}>
              <div>
                <label className="form-label">Method</label>
                <select className="input" value={method} onChange={(e) => setMethod(e.target.value)}>
                  {["GET","POST","PUT","DELETE","PATCH"].map((m) => <option key={m}>{m}</option>)}
                </select>
              </div>
              <div>
                <label className="form-label">Path</label>
                <input className="input mono" value={path} onChange={(e) => setPath(e.target.value)} placeholder="/path/to/resource" />
              </div>
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Bearer Token (optional)</label>
            <input
              className="input mono"
              value={probeToken}
              onChange={(e) => setProbeToken(e.target.value)}
              placeholder="Paste JWT or leave blank to use current session"
            />
          </div>
          <div className="form-group">
            <label className="form-label">Request Body (optional)</label>
            <textarea className="input mono" rows={3} value={body} onChange={(e) => setBody(e.target.value)} placeholder='{"key": "value"}' />
          </div>

          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button className="btn btn-primary" onClick={run} disabled={loading}>
              {loading ? "Sending…" : "Send Request"}
            </button>
            <button className="btn btn-ghost" onClick={onClose}>Close</button>
          </div>

          {err && (
            <div style={{
              marginTop: "0.875rem", padding: "0.5rem 0.75rem", borderRadius: "4px",
              background: "color-mix(in srgb, var(--danger) 10%, transparent)",
              border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)",
              fontSize: "0.8rem", color: "var(--danger)",
            }}>
              {err}
            </div>
          )}

          {result && (
            <div style={{ marginTop: "1rem", borderTop: "1px solid var(--border)", paddingTop: "1rem" }}>
              <div style={{ display: "flex", alignItems: "center", gap: "1rem", marginBottom: "0.625rem", fontSize: "0.8125rem" }}>
                <span className={`badge ${result.status < 300 ? "badge-green" : result.status < 500 ? "badge-yellow" : "badge-red"}`}>
                  {result.status}
                </span>
                <span style={{ color: "var(--text-2)" }}>
                  <span className="mono">{result.latency_ms}ms</span>
                  {" · "}
                  <span className="mono" style={{ color: "var(--muted)" }}>{result.upstream}</span>
                </span>
              </div>
              <pre style={{
                background: "var(--surface2)", border: "1px solid var(--border)", borderRadius: "4px",
                padding: "0.875rem", fontSize: "0.75rem", overflowX: "auto", maxHeight: "14rem", color: "var(--text)", margin: 0,
              }}>
                {(() => { try { return JSON.stringify(JSON.parse(result.body), null, 2); } catch { return result.body; } })()}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Edit/Create Modal ─────────────────────────────────────────────────────────

type EditModalProps = {
  initial?: Partial<Route>;
  onSave: (data: Omit<Route, "id" | "created_at" | "updated_at" | "enabled">) => void;
  onClose: () => void;
};

function EditModal({ initial, onSave, onClose }: EditModalProps) {
  const [form, setForm] = useState({ ...EMPTY, ...initial });
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">{initial?.path_prefix ? "Edit Route" : "New Route"}</h2>
          <p className="modal-subtitle">Configure routing rules for this path prefix</p>
        </div>
        <div className="modal-body">
          {[
            { label: "Path Prefix", key: "path_prefix", placeholder: "/v1/auth" },
            { label: "Upstream Service", key: "upstream", placeholder: "auth-service" },
          ].map(({ label, key, placeholder }) => (
            <div key={key} className="form-group">
              <label className="form-label">{label}</label>
              <input
                className="input mono"
                value={(form as Record<string, unknown>)[key] as string}
                onChange={(e) => setForm({ ...form, [key]: e.target.value })}
                placeholder={placeholder}
              />
            </div>
          ))}
          <div className="form-group">
            <label className="form-label">Auth Mode</label>
            <select className="input" value={form.auth_mode} onChange={(e) => setForm({ ...form, auth_mode: e.target.value as Route["auth_mode"] })}>
              <option value="required">required — JWT required on all methods</option>
              <option value="none">none — public, no auth</option>
              <option value="method_split">method_split — GET/HEAD public, mutations require JWT</option>
            </select>
          </div>
          <label className="form-group" style={{ display: "flex", alignItems: "center", gap: "0.5rem", fontSize: "0.8125rem", color: "var(--text)", cursor: "pointer" }}>
            <input type="checkbox" checked={form.strip_prefix} onChange={(e) => setForm({ ...form, strip_prefix: e.target.checked })} />
            Strip path prefix before forwarding to upstream
          </label>
          <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.25rem" }}>
            <button className="btn btn-primary" onClick={() => onSave(form)}>Save Route</button>
            <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Routes Page ───────────────────────────────────────────────────────────────

export default function RoutesPage() {
  const { data, loading, error: fetchError, refresh } = useDataFetch<Route[]>(getRoutes);
  const routes = data ?? [];

  const [actionError, setActionError] = useState("");
  const [editRoute, setEditRoute] = useState<Route | null>(null);
  const [creating, setCreating] = useState(false);
  const [probing, setProbing] = useState<Route | null>(null);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);

  const error = actionError || fetchError;

  async function handleCreate(data: Omit<Route, "id" | "created_at" | "updated_at" | "enabled">) {
    try {
      await createRoute(data);
      setCreating(false);
      setActionError("");
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleUpdate(id: string, data: Partial<Route>) {
    try {
      await updateRoute(id, data);
      setEditRoute(null);
      setActionError("");
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteRoute(id);
      setConfirmingId(null);
      setActionError("");
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    }
  }

  async function toggleEnabled(route: Route) {
    try {
      await updateRoute(route.id, { enabled: !route.enabled });
      setActionError("");
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Routes</h1>
          <p className="page-subtitle">{routes.length} route{routes.length !== 1 ? "s" : ""} configured</p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-ghost" onClick={refresh}>Refresh</button>
          <button className="btn btn-primary" onClick={() => setCreating(true)}>
            <Plus size={14} /> New Route
          </button>
        </div>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {loading && routes.length === 0 ? (
        <SkeletonTableCard cols={6} rows={5} />
      ) : null}

      <div className="card" style={{ padding: 0, display: loading && routes.length === 0 ? "none" : undefined, overflowX: "auto" }}>
        <table style={{ minWidth: "42rem" }}>
          <thead>
            <tr>
              <th style={{ width: "22rem" }}>Path Prefix</th>
              <th style={{ width: "14rem" }}>Upstream</th>
              <th style={{ width: "9rem" }}>Auth</th>
              <th style={{ width: "5rem" }}>Strip</th>
              <th style={{ width: "7rem" }}>Status</th>
              <th style={{ width: "11rem" }}></th>
            </tr>
          </thead>
          <tbody>
            {routes.length === 0 ? (
              <tr><td colSpan={6}><div className="empty-state"><p className="empty-state-title">No routes configured</p><p className="empty-state-body">Add a route to start proxying traffic</p></div></td></tr>
            ) : routes.map((rt) => (
              <tr key={rt.id} data-status={rt.enabled ? "ok" : "off"}>
                <td className="mono" style={{ whiteSpace: "nowrap" }}>{rt.path_prefix}</td>
                <td className="mono" style={{ color: "var(--muted)", whiteSpace: "nowrap" }}>{rt.upstream}</td>
                <td><span className="badge badge-gray" style={{ whiteSpace: "nowrap" }}>{rt.auth_mode}</span></td>
                <td><span className={`badge ${rt.strip_prefix ? "badge-teal" : "badge-gray"}`}>{rt.strip_prefix ? "yes" : "no"}</span></td>
                <td>
                  <button onClick={() => toggleEnabled(rt)} style={{ background: "none", border: "none", cursor: "pointer", padding: 0 }} title={rt.enabled ? "Click to disable" : "Click to enable"}>
                    <span className={`badge ${rt.enabled ? "badge-green" : "badge-red"}`}>{rt.enabled ? "enabled" : "disabled"}</span>
                  </button>
                </td>
                <td>
                  {confirmingId === rt.id ? (
                    <div style={{ display: "flex", gap: "0.375rem", alignItems: "center" }}>
                      <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>Disable?</span>
                      <button className="btn btn-danger" style={{ padding: "0.25rem 0.5rem", fontSize: "0.75rem" }} onClick={() => handleDelete(rt.id)}>Yes</button>
                      <button className="btn btn-ghost" style={{ padding: "0.25rem 0.5rem", fontSize: "0.75rem" }} onClick={() => setConfirmingId(null)}>No</button>
                    </div>
                  ) : (
                    <div style={{ display: "flex", gap: "0.375rem" }}>
                      <button className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem" }} onClick={() => setProbing(rt)} title="Test route">
                        <FlaskConical size={13} />
                      </button>
                      <button className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem" }} onClick={() => setEditRoute(rt)} title="Edit">
                        <Pencil size={13} />
                      </button>
                      <button className="btn btn-danger" style={{ padding: "0.3rem 0.5rem" }} onClick={() => setConfirmingId(rt.id)} title="Disable">
                        <Trash2 size={13} />
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {creating && <EditModal onSave={handleCreate} onClose={() => setCreating(false)} />}
      {editRoute && (
        <EditModal
          initial={editRoute}
          onSave={(data) => handleUpdate(editRoute.id, data)}
          onClose={() => setEditRoute(null)}
        />
      )}
      {probing && <ProbeModal route={probing} onClose={() => setProbing(null)} />}
    </div>
  );
}
