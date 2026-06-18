"use client";

import { useEffect, useState, useCallback } from "react";
import { getToken } from "@/lib/auth";
import { adminListSellers, adminUpdateSellerStatus, type AdminUser } from "@/lib/api";
import { TableSkeleton } from "@/app/components/Skeleton";
import { showAlert } from "@/app/components/Dialog";

const STATUSES = ["", "PENDING", "APPROVED", "SUSPENDED"];
const PAGE_SIZE = 20;

function SellerStatusBadge({ status }: { status?: string }) {
  if (!status) return <span className="badge badge-gray">—</span>;
  const cls = status === "APPROVED" ? "badge-green" : status === "PENDING" ? "badge-yellow" : "badge-red";
  return <span className={`badge ${cls}`}>{status}</span>;
}

export default function SellersPage() {
  const [rows, setRows]         = useState<AdminUser[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(0);
  const [loading, setLoading]   = useState(true);
  const [filter, setFilter]     = useState("");
  const [updating, setUpdating] = useState<string | null>(null);
  const [pendingCount, setPendingCount] = useState(0);

  const load = useCallback(() => {
    const token = getToken();
    if (!token) return;
    setLoading(true);
    adminListSellers(token, filter || undefined, PAGE_SIZE, page * PAGE_SIZE)
      .then((r) => { setRows(r.data); setTotal(r.total); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [filter, page]);

  useEffect(() => { load(); }, [load]);

  // separate count for pending badge
  useEffect(() => {
    const token = getToken();
    if (!token) return;
    adminListSellers(token, "PENDING", 1, 0)
      .then((r) => setPendingCount(Number(r.total)))
      .catch(() => {});
  }, []);

  async function setStatus(u: AdminUser, status: string) {
    const token = getToken();
    if (!token) return;
    setUpdating(u.id);
    try {
      await adminUpdateSellerStatus(token, u.id, status);
      load();
      if (status === "APPROVED" || u.seller_status === "PENDING") {
        setPendingCount((n) => Math.max(0, n - 1));
      }
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Failed");
    } finally {
      setUpdating(null);
    }
  }

  const pages = Math.ceil(total / PAGE_SIZE);

  return (
    <div style={{ padding: "2rem" }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">
            Sellers
            {pendingCount > 0 && (
              <span className="badge badge-yellow" style={{ marginLeft: "0.625rem", fontSize: "0.6875rem" }}>
                {pendingCount} pending
              </span>
            )}
          </h1>
          <p className="page-subtitle">{total} seller{total === 1 ? "" : "s"}</p>
        </div>
      </div>

      {/* Status filter tabs */}
      <div style={{ display: "flex", gap: "0.375rem", marginBottom: "1.25rem" }}>
        {STATUSES.map((s) => (
          <button
            key={s}
            className={`btn ${filter === s ? "btn-primary" : "btn-ghost"}`}
            style={{ fontSize: "0.75rem", padding: "0.3rem 0.75rem" }}
            onClick={() => { setFilter(s); setPage(0); }}
          >
            {s || "All"}
          </button>
        ))}
      </div>

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Email</th>
              <th>Status</th>
              <th>Joined</th>
              <th style={{ width: 180 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={8} cols={5} />
            ) : rows.length === 0 ? (
              <tr><td colSpan={5}><div className="empty-state"><p className="empty-state-title">No sellers found</p></div></td></tr>
            ) : (
              rows.map((u) => (
                <tr key={u.id}>
                  <td style={{ fontWeight: 500 }}>{u.full_name || <span style={{ color: "var(--muted)" }}>—</span>}</td>
                  <td style={{ color: "var(--text-2)" }}>{u.email}</td>
                  <td><SellerStatusBadge status={u.seller_status} /></td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(u.created_at).toLocaleDateString()}</td>
                  <td>
                    <div style={{ display: "flex", gap: "0.375rem", flexWrap: "wrap" }}>
                      {u.seller_status !== "APPROVED" && (
                        <button
                          className="btn btn-success"
                          style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                          onClick={() => setStatus(u, "APPROVED")}
                          disabled={updating === u.id}
                        >
                          {updating === u.id ? "…" : "✓ Approve"}
                        </button>
                      )}
                      {u.seller_status !== "SUSPENDED" && (
                        <button
                          className="btn btn-danger"
                          style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                          onClick={() => setStatus(u, "SUSPENDED")}
                          disabled={updating === u.id}
                        >
                          {updating === u.id ? "…" : "✗ Suspend"}
                        </button>
                      )}
                      {u.seller_status === "SUSPENDED" && (
                        <button
                          className="btn btn-secondary"
                          style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                          onClick={() => setStatus(u, "APPROVED")}
                          disabled={updating === u.id}
                        >
                          {updating === u.id ? "…" : "Reinstate"}
                        </button>
                      )}
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
    </div>
  );
}
