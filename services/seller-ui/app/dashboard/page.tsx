"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import { getProducts, getSellerOrders, Product, Order } from "@/lib/api";
import { StatCard } from "@/app/components/StatCard";
import { StatusBadge } from "@/app/components/StatusBadge";
import { SkeletonStatCards, SkeletonTableCard } from "@/app/components/Skeleton";

export default function DashboardPage() {
  const [products, setProducts]   = useState<Product[]>([]);
  const [orders, setOrders]       = useState<Order[]>([]);
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState("");

  useEffect(() => {
    let cancelled = false;
    const token = getToken();
    if (!token) return;

    Promise.all([
      getProducts(token, { limit: 100 }),
      getSellerOrders(token, { limit: 10 }),
    ])
      .then(([pr, or]) => {
        if (!cancelled) {
          setProducts(pr.products ?? []);
          setOrders(or.orders ?? []);
          setLoading(false);
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setError(e.message);
          setLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, []);

  const active   = products.filter((p) => p.status === "ACTIVE").length;
  const draft    = products.filter((p) => p.status === "DRAFT").length;
  const archived = products.filter((p) => p.status === "ARCHIVED").length;

  const thisMonth = new Date(); thisMonth.setDate(1); thisMonth.setHours(0, 0, 0, 0);
  const monthOrders = orders.filter((o) => new Date(o.created_at) >= thisMonth);
  const revenue = monthOrders
    .filter((o) => o.status === "CONFIRMED")
    .reduce((sum, o) => sum + o.total_amount, 0);

  const fmtMoney = (n: number) =>
    new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 }).format(n / 100);

  if (loading) {
    return (
      <>
        <div className="page-header"><h1 className="page-title">Overview</h1></div>
        <SkeletonStatCards count={4} />
        <SkeletonTableCard cols={5} rows={6} />
      </>
    );
  }

  return (
    <>
      <div className="page-header">
        <div>
          <h1 className="page-title">Overview</h1>
          <p className="page-subtitle">Welcome back · {new Date().toLocaleDateString("en-US", { month: "long", year: "numeric" })}</p>
        </div>
      </div>

      {error && <p style={{ color: "var(--danger)", marginBottom: "1rem", fontSize: "0.8125rem" }}>{error}</p>}

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(11rem, 1fr))", gap: "1rem", marginBottom: "1.75rem" }}>
        <StatCard label="Total Products"    value={products.length} sub={`${active} active · ${draft} draft`} />
        <StatCard label="Orders This Month" value={monthOrders.length} sub="seller orders" />
        <StatCard label="Revenue This Month" value={fmtMoney(revenue)} sub="confirmed only" accent />
        <StatCard label="Archived Products" value={archived} sub="not publicly listed" />
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 22rem", gap: "1.25rem" }}>
        {/* Recent orders */}
        <div className="card" style={{ padding: 0, overflow: "hidden" }}>
          <div className="card-header">
            <span className="card-title">Recent Orders</span>
            <Link href="/dashboard/orders" style={{ fontSize: "0.8rem", color: "var(--accent)", textDecoration: "none" }}>View all →</Link>
          </div>
          <table>
            <thead>
              <tr><th>Order ID</th><th>Total</th><th>Status</th><th>Date</th></tr>
            </thead>
            <tbody>
              {orders.length === 0 ? (
                <tr><td colSpan={4} style={{ textAlign: "center", color: "var(--muted)", padding: "2rem" }}>No orders yet</td></tr>
              ) : orders.map((o) => (
                <tr key={o.id}>
                  <td><Link href={`/dashboard/orders/${o.id}`} className="mono" style={{ color: "var(--accent)", textDecoration: "none" }}>{o.id.slice(0, 12)}…</Link></td>
                  <td className="mono">{fmtMoney(o.total_amount)}</td>
                  <td><StatusBadge status={o.status} /></td>
                  <td style={{ color: "var(--muted)" }}>{new Date(o.created_at).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Product breakdown */}
        <div className="card" style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <span className="card-title" style={{ marginBottom: "0.25rem" }}>Product Status</span>
          {[
            { label: "Active",   count: active,   color: "var(--success)" },
            { label: "Draft",    count: draft,    color: "var(--warning)" },
            { label: "Archived", count: archived, color: "var(--muted)"   },
          ].map((s) => (
            <div key={s.label}>
              <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.375rem" }}>
                <span style={{ fontSize: "0.8125rem", color: "var(--text-2)", display: "flex", alignItems: "center", gap: "0.5rem" }}>
                  <span style={{ display: "inline-block", width: 8, height: 8, borderRadius: "50%", background: s.color }} />
                  {s.label}
                </span>
                <span style={{ fontSize: "0.8125rem", fontWeight: 500, color: "var(--text)" }}>{s.count}</span>
              </div>
              <div style={{ height: 4, background: "var(--surface2)", borderRadius: 2 }}>
                <div style={{ height: "100%", width: products.length ? `${(s.count / products.length) * 100}%` : "0%", background: s.color, borderRadius: 2, transition: "width 0.3s" }} />
              </div>
            </div>
          ))}
          <div style={{ marginTop: "0.5rem", borderTop: "1px solid var(--border)", paddingTop: "0.875rem" }}>
            <Link href="/dashboard/products/new" className="btn btn-primary" style={{ width: "100%", justifyContent: "center", textDecoration: "none", display: "flex" }}>
              + New Product
            </Link>
          </div>
        </div>
      </div>
    </>
  );
}
