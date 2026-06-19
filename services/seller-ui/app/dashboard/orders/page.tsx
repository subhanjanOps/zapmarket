"use client";
import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { RefreshCw } from "lucide-react";
import { getSellerOrders, Order } from "@/lib/api";
import { StatusBadge } from "@/app/components/StatusBadge";
import { SkeletonTableCard } from "@/app/components/Skeleton";

const STATUSES = ["All", "PENDING", "RESERVED", "CONFIRMED", "CANCELLED"] as const;
const PAGE_SIZE = 20;

export default function OrdersPage() {
  const [orders, setOrders]   = useState<Order[]>([]);
  const [total, setTotal]     = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState("");
  const [status, setStatus]   = useState<typeof STATUSES[number]>("All");
  const [offset, setOffset]   = useState(0);
  const [from, setFrom]       = useState("");
  const [to, setTo]           = useState("");

  const load = useCallback(() => {
    setLoading(true);
    getSellerOrders({
      status: status === "All" ? undefined : status,
      from:   from || undefined,
      to:     to   || undefined,
      limit:  PAGE_SIZE,
      offset,
    })
      .then((r) => { setOrders(r.orders ?? []); setTotal(r.total ?? 0); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [status, from, to, offset]);

  useEffect(() => { load(); }, [load]);

  const fmtMoney = (amount: number, currency: string) =>
    new Intl.NumberFormat("en-US", { style: "currency", currency }).format(amount / 100);

  const pages = Math.ceil(total / PAGE_SIZE);
  const page  = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <>
      <div className="page-header">
        <div>
          <h1 className="page-title">Orders</h1>
          <p className="page-subtitle">{total} total orders</p>
        </div>
        <button className="btn btn-ghost" onClick={load}><RefreshCw size={13} /></button>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "0.75rem", marginBottom: "1.25rem", alignItems: "flex-end", flexWrap: "wrap" }}>
        <div style={{ display: "flex", gap: "2px", background: "var(--surface2)", borderRadius: 8, padding: 2 }}>
          {STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => { setStatus(s); setOffset(0); }}
              style={{ padding: "0.3125rem 0.75rem", borderRadius: 6, border: "none", cursor: "pointer", fontSize: "0.8125rem", fontFamily: "inherit", background: status === s ? "var(--surface)" : "transparent", color: status === s ? "var(--text)" : "var(--muted)", fontWeight: status === s ? 500 : 400, boxShadow: status === s ? "var(--shadow-sm)" : "none", transition: "all 0.1s" }}
            >
              {s === "All" ? "All" : s.charAt(0) + s.slice(1).toLowerCase()}
            </button>
          ))}
        </div>
        <div>
          <label htmlFor="orders-from" style={{ display: "block", fontSize: "0.6875rem", color: "var(--muted)", fontWeight: 500, marginBottom: "0.25rem" }}>From</label>
          <input id="orders-from" className="input" type="date" style={{ width: "10rem" }} value={from} onChange={(e) => { setFrom(e.target.value); setOffset(0); }} />
        </div>
        <div>
          <label htmlFor="orders-to" style={{ display: "block", fontSize: "0.6875rem", color: "var(--muted)", fontWeight: 500, marginBottom: "0.25rem" }}>To</label>
          <input id="orders-to" className="input" type="date" style={{ width: "10rem" }} value={to} onChange={(e) => { setTo(e.target.value); setOffset(0); }} />
        </div>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {loading ? (
        <SkeletonTableCard cols={5} rows={8} />
      ) : (
        <div className="card" style={{ padding: 0, overflow: "hidden" }}>
          <div style={{ overflowX: "auto", WebkitOverflowScrolling: "touch" }}>
            <table>
              <thead>
                <tr><th>Order ID</th><th>Total</th><th>Status</th><th>Date</th><th></th></tr>
              </thead>
              <tbody>
                {orders.length === 0 ? (
                  <tr><td colSpan={5} className="empty-state"><p className="empty-state-title">No orders found</p></td></tr>
                ) : orders.map((o) => (
                  <tr key={o.id}>
                    <td><span className="mono" style={{ color: "var(--muted)" }}>{o.id.slice(0, 12)}…</span></td>
                    <td className="mono">{fmtMoney(o.total_amount, o.currency)}</td>
                    <td><StatusBadge status={o.status} /></td>
                    <td style={{ color: "var(--muted)" }}>{new Date(o.created_at).toLocaleDateString()}</td>
                    <td style={{ textAlign: "right" }}>
                      <Link href={`/dashboard/orders/${o.id}`} className="btn btn-ghost" style={{ fontSize: "0.8rem", padding: "0.3rem 0.625rem", textDecoration: "none" }}>
                        View →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {pages > 1 && (
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "0.75rem 1.125rem", borderTop: "1px solid var(--border)" }}>
              <span style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>Page {page} of {pages}</span>
              <div style={{ display: "flex", gap: "0.375rem" }}>
                <button className="btn btn-ghost" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>← Prev</button>
                <button className="btn btn-ghost" disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset(offset + PAGE_SIZE)}>Next →</button>
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}
