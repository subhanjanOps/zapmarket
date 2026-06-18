"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { getToken } from "@/lib/auth";
import { getSellerOrder, cancelOrder, Order, OrderItem } from "@/lib/api";
import { StatusBadge } from "@/app/components/StatusBadge";
import { StatusTimeline } from "@/app/components/StatusTimeline";
import { SkeletonTableCard, Skel } from "@/app/components/Skeleton";

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();

  const [order, setOrder]     = useState<Order | null>(null);
  const [items, setItems]     = useState<OrderItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState("");
  const [cancelling, setCancelling] = useState(false);

  useEffect(() => {
    const token = getToken();
    if (!token) return;
    getSellerOrder(token, id)
      .then((r) => { setOrder(r.order); setItems(r.items ?? []); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [id]);

  async function handleCancel() {
    if (!confirm("Cancel this order? This cannot be undone.")) return;
    const token = getToken();
    if (!token) return;
    setCancelling(true);
    try {
      const updated = await cancelOrder(token, id);
      setOrder(updated);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Cancel failed");
    } finally {
      setCancelling(false);
    }
  }

  const fmtMoney = (n: number) =>
    new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(n / 100);

  if (loading) {
    return (
      <>
        <div style={{ height: "2.5rem", marginBottom: "1.75rem" }} />
        <div className="card" style={{ display: "flex", flexDirection: "column", gap: "1rem", marginBottom: "1.5rem" }}>
          <Skel w="12rem" h="1.25rem" /><Skel w="6rem" /><Skel w="8rem" />
        </div>
        <SkeletonTableCard cols={5} rows={4} />
      </>
    );
  }

  if (!order) {
    return (
      <div className="card empty-state">
        <p className="empty-state-title">Order not found</p>
        <Link href="/dashboard/orders" style={{ color: "var(--accent)" }}>← Back to orders</Link>
      </div>
    );
  }

  const canCancel = order.status === "PENDING" || order.status === "RESERVED";
  const subtotal  = items.reduce((s, i) => s + i.unit_price * i.quantity, 0);

  return (
    <>
      <div className="page-header">
        <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
          <Link href="/dashboard/orders" className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem", textDecoration: "none" }}>
            <ArrowLeft size={15} />
          </Link>
          <div>
            <h1 className="page-title" style={{ fontFamily: "\"Roboto Mono\", monospace", fontSize: "0.9375rem" }}>
              {order.id.slice(0, 18)}…
            </h1>
            <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginTop: "0.25rem" }}>
              <StatusBadge status={order.status} />
              <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>
                {new Date(order.created_at).toLocaleString()}
              </span>
            </div>
          </div>
        </div>
        {canCancel && (
          <button className="btn btn-danger" onClick={handleCancel} disabled={cancelling}>
            {cancelling ? "Cancelling…" : "Cancel Order"}
          </button>
        )}
      </div>

      {error && (
        <div style={{ background: "color-mix(in srgb, var(--danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)", borderRadius: 7, padding: "0.625rem 0.875rem", fontSize: "0.8125rem", color: "var(--danger)", marginBottom: "1rem" }}>
          {error}
        </div>
      )}

      <div style={{ display: "flex", flexDirection: "column", gap: "1.25rem" }}>
        {/* Status timeline */}
        <div className="card">
          <h3 style={{ margin: "0 0 1.25rem", fontSize: "0.875rem", fontWeight: 600 }}>Status</h3>
          <StatusTimeline status={order.status} />
        </div>

        {/* Buyer info */}
        <div className="card">
          <h3 style={{ margin: "0 0 0.875rem", fontSize: "0.875rem", fontWeight: 600 }}>Buyer</h3>
          <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
            <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>User ID</span>
            <span className="mono" style={{ fontSize: "0.8125rem", color: "var(--text-2)" }}>{order.user_id}</span>
          </div>
          <p style={{ margin: "0.5rem 0 0", fontSize: "0.75rem", color: "var(--muted)" }}>
            Buyer PII is not available in the seller view.
          </p>
        </div>

        {/* Line items */}
        <div className="card" style={{ padding: 0, overflow: "hidden" }}>
          <div className="card-header">
            <span className="card-title">Line Items</span>
          </div>
          <table>
            <thead>
              <tr><th>SKU</th><th>Product</th><th>Qty</th><th>Unit Price</th><th style={{ textAlign: "right" }}>Subtotal</th></tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr><td colSpan={5} style={{ textAlign: "center", color: "var(--muted)", padding: "1.5rem" }}>No line items</td></tr>
              ) : items.map((item) => (
                <tr key={item.id}>
                  <td><span className="mono" style={{ fontSize: "0.8125rem" }}>{item.sku_code ?? item.sku_id.slice(0, 10)}</span></td>
                  <td style={{ color: "var(--text-2)" }}>{item.product_name ?? "—"}</td>
                  <td>{item.quantity}</td>
                  <td className="mono">{fmtMoney(item.unit_price)}</td>
                  <td className="mono" style={{ textAlign: "right", fontWeight: 500 }}>{fmtMoney(item.unit_price * item.quantity)}</td>
                </tr>
              ))}
            </tbody>
          </table>

          {/* Totals footer */}
          <div style={{ padding: "0.875rem 1.125rem", borderTop: "1px solid var(--border)", display: "flex", flexDirection: "column", gap: "0.375rem", alignItems: "flex-end" }}>
            <div style={{ display: "flex", gap: "3rem", fontSize: "0.8125rem", color: "var(--muted)" }}>
              <span>Subtotal</span>
              <span className="mono">{fmtMoney(subtotal)}</span>
            </div>
            <div style={{ display: "flex", gap: "3rem", fontSize: "0.9375rem", fontWeight: 700, color: "var(--text)" }}>
              <span>Total</span>
              <span className="mono">{fmtMoney(order.total_amount)}</span>
            </div>
          </div>
        </div>

        {/* Raw IDs */}
        <div className="card" style={{ padding: "0.875rem 1.125rem" }}>
          <div style={{ display: "flex", gap: "2rem", flexWrap: "wrap" }}>
            {[["Order ID", order.id], ["Idempotency Key", order.idempotency_key], ["Updated", new Date(order.updated_at).toLocaleString()]].map(([label, val]) => (
              <div key={label}>
                <div style={{ fontSize: "0.6875rem", color: "var(--muted)", fontWeight: 500, marginBottom: "0.2rem" }}>{label}</div>
                <div className="mono" style={{ fontSize: "0.75rem", color: "var(--text-2)" }}>{val}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}
