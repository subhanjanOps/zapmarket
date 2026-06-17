"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getRoutes, createRoute, updateRoute, deleteRoute, Route } from "@/lib/api";
import { Plus, Pencil, Trash2, RefreshCw } from "lucide-react";

const AUTH_MODES = ["required", "none", "method_split"] as const;

function AuthBadge({ mode }: { mode: string }) {
  const styles: Record<string, string> = {
    required: "badge badge-red",
    none: "badge badge-green",
    method_split: "badge badge-yellow",
  };
  return <span className={styles[mode] ?? "badge badge-gray"}>{mode}</span>;
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: "rgba(0,0,0,0.6)" }}>
      <div className="card w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-semibold">{title}</h2>
          <button onClick={onClose} style={{ color: "var(--muted)" }}>✕</button>
        </div>
        {children}
      </div>
    </div>
  );
}

type FormState = {
  path_prefix: string;
  upstream: string;
  auth_mode: "required" | "none" | "method_split";
  strip_prefix: boolean;
};

const EMPTY: FormState = { path_prefix: "", upstream: "", auth_mode: "required", strip_prefix: false };

export default function RoutesPage() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  const [showAdd, setShowAdd] = useState(false);
  const [editing, setEditing] = useState<Route | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      const token = getToken();
      if (!token) return;
      setLoading(true);
      try {
        const data = await getRoutes(token);
        if (!cancelled) {
          setRoutes(data);
          setError("");
        }
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load routes");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchData();
    return () => { cancelled = true; };
  }, [refreshKey]);

  function openEdit(r: Route) {
    setEditing(r);
    setForm({ path_prefix: r.path_prefix, upstream: r.upstream, auth_mode: r.auth_mode, strip_prefix: r.strip_prefix });
  }

  async function handleSave() {
    const token = getToken()!;
    setSaving(true);
    try {
      if (editing) {
        await updateRoute(token, editing.id, { upstream: form.upstream, auth_mode: form.auth_mode, strip_prefix: form.strip_prefix });
      } else {
        await createRoute(token, form);
      }
      setShowAdd(false);
      setEditing(null);
      setForm(EMPTY);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(r: Route) {
    if (!confirm(`Disable route ${r.path_prefix}?`)) return;
    try {
      await deleteRoute(getToken()!, r.id);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Delete failed");
    }
  }

  const modalOpen = showAdd || !!editing;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold">Routes</h1>
          <p className="text-sm mt-1" style={{ color: "var(--muted)" }}>
            {routes.length} active route{routes.length !== 1 ? "s" : ""}
          </p>
        </div>
        <div className="flex gap-2">
          <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>
            <RefreshCw size={14} />
          </button>
          <button className="btn btn-primary" onClick={() => { setForm(EMPTY); setShowAdd(true); }}>
            <Plus size={14} /> Add Route
          </button>
        </div>
      </div>

      {error && <p className="mb-4 text-sm" style={{ color: "var(--danger)" }}>{error}</p>}

      <div className="card p-0 overflow-hidden">
        {loading ? (
          <p className="p-6 text-sm" style={{ color: "var(--muted)" }}>Loading…</p>
        ) : routes.length === 0 ? (
          <p className="p-6 text-sm" style={{ color: "var(--muted)" }}>No routes found.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Path Prefix</th>
                <th>Upstream</th>
                <th>Auth Mode</th>
                <th>Strip Prefix</th>
                <th>Updated</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {routes.map((r) => (
                <tr key={r.id}>
                  <td><span className="mono" style={{ fontSize: "0.8125rem", color: "var(--accent-hover)" }}>{r.path_prefix}</span></td>
                  <td className="mono" style={{ fontSize: "0.8125rem" }}>{r.upstream}</td>
                  <td><AuthBadge mode={r.auth_mode} /></td>
                  <td>
                    <span className={`badge ${r.strip_prefix ? "badge-blue" : "badge-gray"}`}>
                      {r.strip_prefix ? "yes" : "no"}
                    </span>
                  </td>
                  <td style={{ color: "var(--muted)" }} className="text-xs">
                    {new Date(r.updated_at).toLocaleString()}
                  </td>
                  <td>
                    <div className="flex gap-2">
                      <button className="btn btn-ghost" style={{ padding: "0.3rem 0.6rem" }} onClick={() => openEdit(r)}>
                        <Pencil size={13} />
                      </button>
                      <button className="btn btn-danger" style={{ padding: "0.3rem 0.6rem" }} onClick={() => handleDelete(r)}>
                        <Trash2 size={13} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {modalOpen && (
        <Modal title={editing ? "Edit Route" : "Add Route"} onClose={() => { setShowAdd(false); setEditing(null); }}>
          <div className="flex flex-col gap-3">
            <div>
              <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>Path Prefix</label>
              <input
                className="input"
                value={form.path_prefix}
                onChange={(e) => setForm((f) => ({ ...f, path_prefix: e.target.value }))}
                placeholder="/v1/my-service"
                disabled={!!editing}
              />
            </div>
            <div>
              <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>Upstream</label>
              <input
                className="input"
                value={form.upstream}
                onChange={(e) => setForm((f) => ({ ...f, upstream: e.target.value }))}
                placeholder="my-service"
              />
            </div>
            <div>
              <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>Auth Mode</label>
              <select
                className="input"
                value={form.auth_mode}
                onChange={(e) => setForm((f) => ({ ...f, auth_mode: e.target.value as FormState["auth_mode"] }))}
              >
                {AUTH_MODES.map((m) => <option key={m} value={m}>{m}</option>)}
              </select>
            </div>
            <div className="flex items-center gap-2">
              <input
                id="strip"
                type="checkbox"
                checked={form.strip_prefix}
                onChange={(e) => setForm((f) => ({ ...f, strip_prefix: e.target.checked }))}
              />
              <label htmlFor="strip" className="text-sm">Strip prefix before forwarding</label>
            </div>
            <div className="flex gap-2 mt-2">
              <button className="btn btn-primary flex-1" onClick={handleSave} disabled={saving}>
                {saving ? "Saving…" : "Save"}
              </button>
              <button className="btn btn-ghost flex-1" onClick={() => { setShowAdd(false); setEditing(null); }}>
                Cancel
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
