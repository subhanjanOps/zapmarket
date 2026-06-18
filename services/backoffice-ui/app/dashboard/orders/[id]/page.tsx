"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import { adminGetOrder, adminCancelOrder, type AdminOrder, type OrderItem } from "@/lib/api";
import StatusBadge from "@/app/components/StatusBadge";
import Skeleton from "@/app/components/Skeleton";

function fmt(cents: number, currency = "USD") {
  return new Intl.NumberFormat("en-US", { style: "currency", currency }).format(cents / 100);
}

const CANCELLABLE = new Set(["PENDING", "RESERVED"]);

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [order, setOrder] = useState<(AdminOrder & { items: OrderItem[] }) | null>(null);
  const [loading, setLoading] = useState(true);
  const [cancelling, setCancelling] = useState(false);

  useEffect(() => {
    const token = getToken();
    if (!token) return;
    setLoading(true);
    adminGetOrder(token, id)
      .then(setOrder)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [id]);

  async function cancel() {
    if (!order || !confirm("Force-cancel this order?")) return;
    const token = getToken();
    if (!token) return;
    setCancelling(true);
    try {
      await adminCancelOrder(token, order.id);
      const token2 = getToken()!;
      const refreshed = await adminGetOrder(token2, id);
      setOrder(refreshed);
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "Failed");
    } finally {
      setCancelling(false);
    }
  }

  return (
    <div style={{ padding: "2rem", maxWidth: 860 }}>
      {/* Breadcrumb */}
      <div style={{ display: "flex", alignItems: "center", gap: "0.375rem", marginBottom: "1.25rem", fontSize: "0.8125rem", color: "var(--muted)" }}>
        <Link href="/dashboard/orders" style={{ color: "var(--accent)", textDecoration: "none" }}>Orders</Link>
        <span>/</span>
        <span className="mono">{loading ? "…" : id.slice(0, 8) + "…"}</span>
      </div>

      {loading ? (
        <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
          {[120, 80, 200, 160].map((w, i) => <Skeleton key={i} w={w} h={20} />)}
        </div>
      ) : !order ? (
        <div className="empty-state">
          <p className="empty-state-title">Order not found</p>
          <button className="btn btn-ghost" onClick={() => router.push("/dashboard/orders")}>← Back</button>
        </div>
      ) : (
        <>
          {/* Header */}
          <div className="page-header">
            <div>
              <h1 className="page-title" style={{ display: "flex", alignItems: "center", gap: "0.625rem" }}>
                Order
                <span className="mono" style={{ fontSize: "1rem", fontWeight: 400, color: "var(--muted)" }}>
                  {order.id}
                </span>
              </h1>
              <p className="page-subtitle">Placed {new Date(order.created_at).toLocaleString()}</p>
            </div>
            <div style={{ display: "flex", gap: "0.625rem", alignItems: "center" }}>
              <StatusBadge status={order.status} />
              {CANCELLABLE.has(order.status) && (
                <button className="btn btn-danger" onClick={cancel} disabled={cancelling}>
                  {cancelling ? "Cancelling…" : "Cancel order"}
                </button>
              )}
            </div>
          </div>

          {/* Meta cards */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: "0.75rem", marginBottom: "1.5rem" }}>
            <div className="card" style={{ padding: "1rem" }}>
              <div style={{ fontSize: "0.6875rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.06em", marginBottom: "0.375rem" }}>Total</div>
              <div style={{ fontSize: "1.375rem", fontWeight: 700, color: "var(--text)" }}>{fmt(order.total_amount, order.currency)}</div>
            </div>
            <div className="card" style={{ padding: "1rem" }}>
              <div style={{ fontSize: "0.6875rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.06em", marginBottom: "0.375rem" }}>User ID</div>
              <div className="mono" style={{ fontSize: "0.8125rem", color: "var(--text-2)", wordBreak: "break-all" }}>{order.user_id}</div>
            </div>
            <div className="card" style={{ padding: "1rem" }}>
              <div style={{ fontSize: "0.6875rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.06em", marginBottom: "0.375rem" }}>Idempotency Key</div>
              <div className="mono" style={{ fontSize: "0.8125rem", color: "var(--text-2)", wordBreak: "break-all" }}>{order.idempotency_key}</div>
            </div>
          </div>

          {order.payment_id && (
            <div className="card" style={{ padding: "0.875rem 1rem", marginBottom: "1.5rem", display: "flex", alignItems: "center", gap: "0.75rem" }}>
              <span style={{ fontSize: "0.75rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.06em" }}>Payment ID</span>
              <span className="mono" style={{ fontSize: "0.8125rem", color: "var(--text-2)" }}>{order.payment_id}</span>
            </div>
          )}

          {/* Items */}
          <h2 style={{ fontSize: "0.9375rem", fontWeight: 600, color: "var(--text)", marginBottom: "0.75rem" }}>
            Items <span style={{ color: "var(--muted)", fontWeight: 400 }}>({order.items?.length ?? 0})</span>
          </h2>
          <div className="card" style={{ padding: 0, overflow: "hidden", marginBottom: "1.5rem" }}>
            <table>
              <thead>
                <tr>
                  <th>SKU ID</th>
                  <th>Seller</th>
                  <th>Qty</th>
                  <th>Unit Price</th>
                  <th>Subtotal</th>
                </tr>
              </thead>
              <tbody>
                {!order.items?.length ? (
                  <tr><td colSpan={5} style={{ textAlign: "center", color: "var(--muted)", padding: "1.5rem" }}>No items</td></tr>
                ) : (
                  order.items.map((item) => (
                    <tr key={item.id}>
                      <td><span className="mono" style={{ fontSize: "0.75rem" }}>{item.sku_id}</span></td>
                      <td><span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{item.seller_id?.slice(0, 8) ?? "—"}…</span></td>
                      <td style={{ fontWeight: 500 }}>{item.quantity}</td>
                      <td>{fmt(item.unit_price, order.currency)}</td>
                      <td style={{ fontWeight: 500 }}>{fmt(item.unit_price * item.quantity, order.currency)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Timestamps */}
          <div style={{ display: "flex", gap: "1.5rem", fontSize: "0.75rem", color: "var(--muted)" }}>
            <span>Created: {new Date(order.created_at).toLocaleString()}</span>
            <span>Updated: {new Date(order.updated_at).toLocaleString()}</span>
          </div>
        </>
      )}
    </div>
  );
}
