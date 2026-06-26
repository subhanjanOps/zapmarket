"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import {
  adminListUsers, adminUpdateUserRole, adminDeactivateUser,
  type AdminUser,
} from "@/lib/api";
import StatusBadge from "@/app/components/StatusBadge";
import { TableSkeleton } from "@/app/components/Skeleton";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const ROLES = ["", "buyer", "seller", "admin"];
const PAGE_SIZE = 20;

export default function UsersPage() {
  const [rows, setRows]         = useState<AdminUser[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(0);
  const [loading, setLoading]   = useState(true);
  const [search, setSearch]     = useState("");
  const [role, setRole]         = useState("");
  const [editing, setEditing]   = useState<AdminUser | null>(null);
  const [newRole, setNewRole]   = useState("");
  const [saving, setSaving]     = useState(false);
  const [roleErr, setRoleErr]   = useState("");
  const [error, setError]       = useState<string | null>(null);
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => { setDebouncedSearch(search); setPage(0); }, 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [search]);

  const load = useCallback(() => {
    setLoading(true);
    adminListUsers({ search: debouncedSearch || undefined, role: role || undefined, limit: PAGE_SIZE, offset: page * PAGE_SIZE })
      .then((r) => { setRows(r.data); setTotal(r.total); setError(null); })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "Failed to load users"))
      .finally(() => setLoading(false));
  }, [debouncedSearch, role, page]);

  useEffect(() => { load(); }, [load]);

  function openEditRole(u: AdminUser) {
    setEditing(u);
    setNewRole(u.role);
    setRoleErr("");
  }

  async function saveRole() {
    if (!editing) return;
    setSaving(true);
    setRoleErr("");
    try {
      await adminUpdateUserRole(editing.id, newRole);
      setEditing(null);
      load();
    } catch (e: unknown) {
      setRoleErr(e instanceof Error ? e.message : "Failed");
    } finally {
      setSaving(false);
    }
  }

  async function deactivate(u: AdminUser) {
    if (!await showConfirm(`Deactivate ${u.email}? This will immediately revoke their access.`)) return;
    try { await adminDeactivateUser(u.id); load(); }
    catch (e: unknown) { await showAlert(e instanceof Error ? e.message : "Failed"); }
  }

  const pages = Math.ceil(total / PAGE_SIZE);

  return (
    <div className="page-content">
      {error && (
        <div style={{ marginBottom: "1rem", padding: "0.75rem 1rem", background: "#FEF2F2", border: "1px solid #FCA5A5", borderRadius: "0.5rem", color: "#DC2626", fontSize: "0.875rem" }}>
          {error}
        </div>
      )}
      <div className="page-header">
        <div>
          <h1 className="page-title">Users</h1>
          <p className="page-subtitle">{total} user{total === 1 ? "" : "s"}</p>
        </div>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "0.625rem", marginBottom: "1.25rem", flexWrap: "wrap" }}>
        <input
          className="input"
          style={{ maxWidth: 280 }}
          placeholder="Search email or name…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select className="input" style={{ maxWidth: 160 }} value={role} onChange={(e) => { setRole(e.target.value); setPage(0); }}>
          {ROLES.map((r) => <option key={r} value={r}>{r || "All roles"}</option>)}
        </select>
      </div>

      <div style={{ border: "1px solid var(--border)", borderRadius: 7, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Email</th>
              <th>Role</th>
              <th>Email verified</th>
              <th>Seller status</th>
              <th>Joined</th>
              <th style={{ width: 160 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={8} cols={7} />
            ) : rows.length === 0 ? (
              <tr><td colSpan={7}><div className="empty-state"><p className="empty-state-title">No users found</p></div></td></tr>
            ) : (
              rows.map((u) => (
                <tr key={u.id}>
                  <td style={{ fontWeight: 500 }}>{u.full_name || <span style={{ color: "var(--muted)" }}>—</span>}</td>
                  <td style={{ color: "var(--text-2)" }}>{u.email}</td>
                  <td>
                    <span className={`badge ${u.role === "admin" ? "badge-teal" : u.role === "seller" ? "badge-blue" : "badge-gray"}`}>
                      {u.role}
                    </span>
                  </td>
                  <td>
                    <span className={`status-dot ${u.is_verified ? "status-dot-green" : "status-dot-amber"}`} style={{ marginRight: 6 }} />
                    {u.is_verified ? "Yes" : "No"}
                  </td>
                  <td>
                    {u.role === "seller" && u.seller_status ? (
                      <span className={`badge ${u.seller_status === "APPROVED" ? "badge-green" : u.seller_status === "PENDING" ? "badge-yellow" : "badge-red"}`}>
                        {u.seller_status}
                      </span>
                    ) : (
                      <span style={{ color: "var(--muted)" }}>—</span>
                    )}
                  </td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(u.created_at).toLocaleDateString()}</td>
                  <td>
                    <div style={{ display: "flex", gap: "0.375rem" }}>
                      <button className="btn btn-ghost" style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }} onClick={() => openEditRole(u)}>
                        Edit role
                      </button>
                      <button className="btn btn-danger" style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }} onClick={() => deactivate(u)}>
                        Deactivate
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {pages > 1 && (
        <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginTop: "1rem", justifyContent: "flex-end" }}>
          <button className="btn btn-ghost" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>← Prev</button>
          <span style={{ fontSize: "0.8125rem", color: "var(--text-2)" }}>Page {page + 1} / {pages}</span>
          <button className="btn btn-ghost" disabled={page >= pages - 1} onClick={() => setPage((p) => p + 1)}>Next →</button>
        </div>
      )}

      {/* Edit Role Modal */}
      {editing && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="modal-title">Change role</h2>
              <p className="modal-subtitle">{editing.email}</p>
            </div>
            <div className="modal-body">
              <div style={{ display: "flex", flexDirection: "column", gap: "0.875rem" }}>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label className="form-label">New role</label>
                  <select className="input" value={newRole} onChange={(e) => setNewRole(e.target.value)}>
                    <option value="buyer">buyer</option>
                    <option value="seller">seller</option>
                    <option value="admin">admin</option>
                  </select>
                </div>
                {roleErr && <p style={{ color: "var(--danger)", fontSize: "0.8rem", margin: 0 }}>{roleErr}</p>}
                <div style={{ display: "flex", gap: "0.625rem", justifyContent: "flex-end" }}>
                  <button className="btn btn-ghost" onClick={() => setEditing(null)}>Cancel</button>
                  <button className="btn btn-primary" onClick={saveRole} disabled={saving || newRole === editing.role}>
                    {saving ? "Saving…" : "Update role"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
