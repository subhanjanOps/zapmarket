"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getRoutes, createRoute, updateRoute, deleteRoute, probeRoute, Route, ProbeResult } from "@/lib/api";
import { Plus, Pencil, Trash2, FlaskConical } from "lucide-react";

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
  const [probeToken, setProbeToken] = useState(getToken() ?? "");
  const [result, setResult] = useState<ProbeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");

  async function run() {
    const token = getToken();
    if (!token) return;
    setLoading(true);
    setErr("");
    setResult(null);
    try {
      const r = await probeRoute(token, { method, path, token: probeToken || undefined, body: body || undefined });
      setResult(r);
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.6)", display: "flex",
      alignItems: "center", justifyContent: "center", zIndex: 100,
    }} onClick={onClose}>
      <div style={{
        background: "var(--surface)", border: "1px solid var(--border)", borderRadius: "4px",
        padding: "1.5rem", width: "min(600px, 94vw)", maxHeight: "90vh", overflowY: "auto",
      }} onClick={(e) => e.stopPropagation()}>
        <h2 style={{ fontSize: "0.9375rem", fontWeight: 600, color: "var(--text)", margin: "0 0 1rem" }}>
          Route Tester — <span className="mono" style={{ color: "var(--accent)" }}>{route.path_prefix}</span>
        </h2>

        <div style={{ display: "grid", gridTemplateColumns: "6rem 1fr", gap: "0.5rem", marginBottom: "0.75rem" }}>
          <select className="input" value={method} onChange={(e) => setMethod(e.target.value)}>
            {["GET","POST","PUT","DELETE","PATCH"].map((m) => <option key={m}>{m}</option>)}
          </select>
          <input className="input mono" value={path} onChange={(e) => setPath(e.target.value)} placeholder="/path" />
        </div>
        <input
          className="input mono"
          style={{ marginBottom: "0.5rem" }}
          value={probeToken}
          onChange={(e) => setProbeToken(e.target.value)}
          placeholder="Bearer token (optional)"
        />
        <textarea
          className="input mono"
          rows={3}
          style={{ marginBottom: "0.75rem", resize: "vertical" }}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder='{"key": "value"} (optional body)'
        />

        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-primary" onClick={run} disabled={loading}>
            {loading ? "Sending…" : "Send"}
          </button>
          <button className="btn btn-ghost" onClick={onClose}>Close</button>
        </div>

        {err && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginTop: "0.75rem" }}>{err}</p>}

        {result && (
          <div style={{ marginTop: "1rem" }}>
            <div style={{ display: "flex", gap: "1rem", marginBottom: "0.5rem", fontSize: "0.8125rem" }}>
              <span>
                Status:{" "}
                <span className={`badge ${result.status < 300 ? "badge-green" : result.status < 500 ? "badge-yellow" : "badge-red"}`}>
                  {result.status}
                </span>
              </span>
              <span style={{ color: "var(--muted)" }}>{result.latency_ms}ms via <span className="mono">{result.upstream}</span></span>
            </div>
            <pre style={{
              background: "var(--surface2)", border: "1px solid var(--border)", borderRadius: "3px",
              padding: "0.75rem", fontSize: "0.75rem", overflowX: "auto", maxHeight: "12rem", color: "var(--text)",
            }}>
              {(() => { try { return JSON.stringify(JSON.parse(result.body), null, 2); } catch { return result.body; } })()}
            </pre>
          </div>
        )}
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
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.6)", display: "flex",
      alignItems: "center", justifyContent: "center", zIndex: 100,
    }} onClick={onClose}>
      <div style={{
        background: "var(--surface)", border: "1px solid var(--border)", borderRadius: "4px",
        padding: "1.5rem", width: "min(500px, 94vw)",
      }} onClick={(e) => e.stopPropagation()}>
        <h2 style={{ fontSize: "0.9375rem", fontWeight: 600, color: "var(--text)", margin: "0 0 1rem" }}>
          {initial?.path_prefix ? "Edit Route" : "New Route"}
        </h2>
        {[
          { label: "Path Prefix", key: "path_prefix", placeholder: "/v1/auth" },
          { label: "Upstream", key: "upstream", placeholder: "auth-service" },
        ].map(({ label, key, placeholder }) => (
          <div key={key} style={{ marginBottom: "0.75rem" }}>
            <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>{label}</label>
            <input
              className="input mono"
              value={(form as Record<string, unknown>)[key] as string}
              onChange={(e) => setForm({ ...form, [key]: e.target.value })}
              placeholder={placeholder}
            />
          </div>
        ))}
        <div style={{ marginBottom: "0.75rem" }}>
          <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>Auth Mode</label>
          <select className="input" value={form.auth_mode} onChange={(e) => setForm({ ...form, auth_mode: e.target.value as Route["auth_mode"] })}>
            <option value="required">required</option>
            <option value="none">none</option>
            <option value="method_split">method_split</option>
          </select>
        </div>
        <label style={{ display: "flex", alignItems: "center", gap: "0.5rem", fontSize: "0.8125rem", color: "var(--text)", marginBottom: "1rem", cursor: "pointer" }}>
          <input type="checkbox" checked={form.strip_prefix} onChange={(e) => setForm({ ...form, strip_prefix: e.target.checked })} />
          Strip path prefix before forwarding
        </label>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-primary" onClick={() => onSave(form)}>Save</button>
          <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  );
}

// ── Routes Page ───────────────────────────────────────────────────────────────

export default function RoutesPage() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [editRoute, setEditRoute] = useState<Route | null>(null);
  const [creating, setCreating] = useState(false);
  const [probing, setProbing] = useState<Route | null>(null);

  useEffect(() => {
    let cancelled = false;
    const token = getToken();
    if (!token) return;
    getRoutes(token)
      .then((r) => { if (!cancelled) { setRoutes(r); setError(""); } })
      .catch((e) => { if (!cancelled) setError(e.message); });
    return () => { cancelled = true; };
  }, [refreshKey]);

  const token = getToken()!;

  async function handleCreate(data: Omit<Route, "id" | "created_at" | "updated_at" | "enabled">) {
    try {
      await createRoute(token, data);
      setCreating(false);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleUpdate(id: string, data: Partial<Route>) {
    try {
      await updateRoute(token, id, data);
      setEditRoute(null);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Disable this route?")) return;
    try {
      await deleteRoute(token, id);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function toggleEnabled(route: Route) {
    try {
      await updateRoute(token, route.id, { enabled: !route.enabled });
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
        <div>
          <h1 style={{ fontSize: "1.125rem", fontWeight: 600, color: "var(--text)", margin: 0 }}>Routes</h1>
          <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0.25rem 0 0" }}>
            {routes.length} route{routes.length !== 1 ? "s" : ""} configured
          </p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>Refresh</button>
          <button className="btn btn-primary" style={{ gap: "0.4rem" }} onClick={() => setCreating(true)}>
            <Plus size={14} /> New Route
          </button>
        </div>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      <div className="card" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr>
              <th>Path Prefix</th>
              <th>Upstream</th>
              <th>Auth</th>
              <th>Strip</th>
              <th>Status</th>
              <th style={{ width: "8rem" }}></th>
            </tr>
          </thead>
          <tbody>
            {routes.length === 0 ? (
              <tr><td colSpan={6} style={{ textAlign: "center", color: "var(--muted)", padding: "2rem" }}>No routes yet</td></tr>
            ) : routes.map((rt) => (
              <tr key={rt.id}>
                <td className="mono">{rt.path_prefix}</td>
                <td className="mono" style={{ color: "var(--muted)" }}>{rt.upstream}</td>
                <td><span className="badge badge-gray">{rt.auth_mode}</span></td>
                <td><span className={`badge ${rt.strip_prefix ? "badge-teal" : "badge-gray"}`}>{rt.strip_prefix ? "yes" : "no"}</span></td>
                <td>
                  <button
                    onClick={() => toggleEnabled(rt)}
                    style={{ background: "none", border: "none", cursor: "pointer", padding: 0 }}
                    title={rt.enabled ? "Click to disable" : "Click to enable"}
                  >
                    <span className={`badge ${rt.enabled ? "badge-green" : "badge-red"}`}>
                      {rt.enabled ? "enabled" : "disabled"}
                    </span>
                  </button>
                </td>
                <td>
                  <div style={{ display: "flex", gap: "0.375rem" }}>
                    <button className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem" }} onClick={() => setProbing(rt)} title="Test route">
                      <FlaskConical size={13} />
                    </button>
                    <button className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem" }} onClick={() => setEditRoute(rt)} title="Edit">
                      <Pencil size={13} />
                    </button>
                    <button className="btn btn-danger" style={{ padding: "0.3rem 0.5rem" }} onClick={() => handleDelete(rt.id)} title="Disable">
                      <Trash2 size={13} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {creating && (
        <EditModal onSave={handleCreate} onClose={() => setCreating(false)} />
      )}
      {editRoute && (
        <EditModal
          initial={editRoute}
          onSave={(data) => handleUpdate(editRoute.id, data)}
          onClose={() => setEditRoute(null)}
        />
      )}
      {probing && (
        <ProbeModal route={probing} onClose={() => setProbing(null)} />
      )}
    </div>
  );
}
