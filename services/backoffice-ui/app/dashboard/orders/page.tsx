"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import { adminListOrders, adminCancelOrder, type AdminOrder } from "@/lib/api";
import StatusBadge from "@/app/components/StatusBadge";
import { TableSkeleton } from "@/app/components/Skeleton";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const STATUSES = ["", "PENDING", "RESERVED", "CONFIRMED", "CANCELLED"];
const PAGE_SIZE = 20;
const CANCELLABLE = new Set(["PENDING", "RESERVED"]);

function fmt(cents: number, currency = "USD") {
  return new Intl.NumberFormat("en-US", { style: "currency", currency }).format(cents / 100);
}

export default function OrdersPage() {
  const [rows, setRows]         = useState<AdminOrder[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(0);
  const [loading, setLoading]   = useState(true);
  const [status, setStatus]     = useState("");
  const [userId, setUserId]     = useState("");
  const [from, setFrom]         = useState("");
  const [to, setTo]             = useState("");
  const [cancelling, setCancelling] = useState<string | null>(null);

  const load = useCallback(() => {
    const token = getToken();
    if (!token) return;
    setLoading(true);
    adminListOrders(token, {
      status: status || undefined,
      user_id: userId || undefined,
      from: from || undefined,
      to: to || undefined,
      limit: PAGE_SIZE,
      offset: page * PAGE_SIZE,
    })
      .then((r) => { setRows(r.data); setTotal(r.total); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [status, userId, from, to, page]);

  useEffect(() => { load(); }, [load]);

  async function cancel(id: string) {
    if (!await showConfirm("Force-cancel this order?")) return;
    const token = getToken();
    if (!token) return;
    setCancelling(id);
    try { await adminCancelOrder(token, id); load(); }
    catch (e: unknown) { await showAlert(e instanceof Error ? e.message : "Failed"); }
    finally { setCancelling(null); }
  }

  const pages = Math.ceil(total / PAGE_SIZE);

  return (
    <div style={{ padding: "2rem" }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">Orders</h1>
          <p className="page-subtitle">{total} order{total === 1 ? "" : "s"}</p>
        </div>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "0.625rem", marginBottom: "0.75rem", flexWrap: "wrap" }}>
        <select className="input" style={{ maxWidth: 160 }} value={status} onChange={(e) => { setStatus(e.target.value); setPage(0); }}>
          {STATUSES.map((s) => <option key={s} value={s}>{s || "All statuses"}</option>)}
        </select>
        <input
          className="input"
          style={{ maxWidth: 280 }}
          placeholder="Filter by User ID…"
          value={userId}
          onChange={(e) => { setUserId(e.target.value); setPage(0); }}
        />
      </div>
      <div style={{ display: "flex", gap: "0.625rem", marginBottom: "1.25rem", flexWrap: "wrap", alignItems: "center" }}>
        <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>Date range:</span>
        <input className="input" type="date" style={{ maxWidth: 160 }} value={from} onChange={(e) => { setFrom(e.target.value); setPage(0); }} />
        <span style={{ color: "var(--muted)" }}>→</span>
        <input className="input" type="date" style={{ maxWidth: 160 }} value={to} onChange={(e) => { setTo(e.target.value); setPage(0); }} />
        {(from || to) && (
          <button className="btn btn-ghost" style={{ fontSize: "0.75rem" }} onClick={() => { setFrom(""); setTo(""); }}>
            Clear
          </button>
        )}
      </div>

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>Order ID</th>
              <th>User</th>
              <th>Total</th>
              <th>Status</th>
              <th>Date</th>
              <th style={{ width: 140 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={8} cols={6} />
            ) : rows.length === 0 ? (
              <tr><td colSpan={6}><div className="empty-state"><p className="empty-state-title">No orders found</p></div></td></tr>
            ) : (
              rows.map((o) => (
                <tr key={o.id}>
                  <td>
                    <Link href={`/dashboard/orders/${o.id}`} style={{ color: "var(--accent)", textDecoration: "none" }} className="mono">
                      {o.id.slice(0, 8)}…
                    </Link>
                  </td>
                  <td><span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{o.user_id.slice(0, 8)}…</span></td>
                  <td style={{ fontWeight: 500 }}>{fmt(o.total_amount, o.currency)}</td>
                  <td><StatusBadge status={o.status} /></td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(o.created_at).toLocaleDateString()}</td>
                  <td>
                    <div style={{ display: "flex", gap: "0.375rem" }}>
                      <Link href={`/dashboard/orders/${o.id}`} className="btn btn-ghost" style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}>
                        View
                      </Link>
                      {CANCELLABLE.has(o.status) && (
                        <button
                          className="btn btn-danger"
                          style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                          onClick={() => cancel(o.id)}
                          disabled={cancelling === o.id}
                        >
                          {cancelling === o.id ? "…" : "Cancel"}
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
