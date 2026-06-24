"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { Package, ShoppingBag, DollarSign, Archive, Plus, ArrowRight } from "lucide-react";
import { getProducts, getSellerOrders, Product, Order } from "@/lib/api";
import { useCurrency } from "@/lib/currency";
import { StatCard } from "@/app/components/StatCard";
import { StatusBadge } from "@/app/components/StatusBadge";
import { SkeletonStatCards, SkeletonTableCard } from "@/app/components/Skeleton";

export default function DashboardPage() {
  const [productTotal, setProductTotal] = useState(0);
  const [activeCount, setActiveCount]   = useState(0);
  const [draftCount, setDraftCount]     = useState(0);
  const [orders, setOrders]       = useState<Order[]>([]);
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState("");

  useEffect(() => {
    let cancelled = false;

    Promise.all([
      getProducts({ limit: 1 }),
      getProducts({ status: "ACTIVE", limit: 1 }),
      getProducts({ status: "DRAFT", limit: 1 }),
      getSellerOrders({ limit: 10 }),
    ])
      .then(([all, active, draft, or]) => {
        if (!cancelled) {
          setProductTotal(all.total);
          setActiveCount(active.total);
          setDraftCount(draft.total);
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

  const { format, formatFrom, rates } = useCurrency();
  const archivedCount = Math.max(0, productTotal - activeCount - draftCount);

  const thisMonth = new Date(new Date().getFullYear(), new Date().getMonth(), 1);
  const monthOrders = orders.filter((o) => new Date(o.created_at) >= thisMonth);

  const revenueUsdCents = monthOrders
    .filter((o) => o.status === "CONFIRMED")
    .reduce((sum, o) => sum + o.total_amount / (rates[o.currency] ?? 1), 0);
  const revenueDisplay = format(Math.round(revenueUsdCents));

  const now = new Date();
  const greeting =
    now.getHours() < 12 ? "Good morning" : now.getHours() < 18 ? "Good afternoon" : "Good evening";

  if (loading) {
    return (
      <>
        <div className="page-header">
          <div>
            <h1 className="page-title">Overview</h1>
            <p className="page-subtitle">Loading your dashboard…</p>
          </div>
        </div>
        <SkeletonStatCards count={4} />
        <SkeletonTableCard cols={5} rows={6} />
      </>
    );
  }

  return (
    <>
      {/* Header */}
      <div className="page-header">
        <div>
          <p style={{ fontSize: "0.8rem", color: "var(--muted)", margin: "0 0 0.2rem", fontWeight: 500, letterSpacing: "0.02em" }}>
            {greeting} ·{" "}
            {now.toLocaleDateString("en-US", { weekday: "long", month: "long", day: "numeric" })}
          </p>
          <h1 className="page-title">Dashboard</h1>
        </div>
        <Link
          href="/dashboard/products/new"
          className="btn btn-primary"
          style={{ textDecoration: "none", gap: "0.5rem" }}
        >
          <Plus size={14} />
          New Product
        </Link>
      </div>

      {error && (
        <p style={{ color: "var(--danger)", marginBottom: "1rem", fontSize: "0.8125rem" }}>{error}</p>
      )}

      {/* KPI stat cards */}
      <div
        className="stat-cards-grid"
        style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(12rem, 1fr))", gap: "1rem", marginBottom: "1.75rem" }}
      >
        <StatCard
          label="Total Products"
          value={productTotal}
          sub={`${activeCount} active · ${draftCount} draft`}
          icon={Package}
          iconColor="var(--info)"
        />
        <StatCard
          label="Orders This Month"
          value={monthOrders.length}
          sub="incoming orders"
          icon={ShoppingBag}
          iconColor="var(--warning)"
        />
        <StatCard
          label="Revenue This Month"
          value={revenueDisplay}
          sub="confirmed orders"
          accent
          icon={DollarSign}
        />
        <StatCard
          label="Archived Products"
          value={archivedCount}
          sub="not publicly listed"
          icon={Archive}
          iconColor="var(--muted)"
        />
      </div>

      {/* Main grid */}
      <div
        className="dash-overview-grid"
        style={{ display: "grid", gridTemplateColumns: "1fr 21rem", gap: "1.25rem", alignItems: "start" }}
      >
        {/* Recent orders table */}
        <div className="card" style={{ padding: 0, overflow: "hidden" }}>
          <div className="card-header">
            <div>
              <div className="card-title">Recent Orders</div>
              <div style={{ fontSize: "0.75rem", color: "var(--muted)", marginTop: 2 }}>
                {orders.length > 0 ? `Last ${orders.length} orders` : "No orders yet"}
              </div>
            </div>
            <Link
              href="/dashboard/orders"
              style={{
                fontSize: "0.8rem",
                color: "var(--accent)",
                textDecoration: "none",
                display: "inline-flex",
                alignItems: "center",
                gap: "0.25rem",
                fontWeight: 600,
              }}
            >
              View all <ArrowRight size={13} />
            </Link>
          </div>
          <div style={{ overflowX: "auto", WebkitOverflowScrolling: "touch" }}>
            <table>
              <thead>
                <tr>
                  <th>Order ID</th>
                  <th>Total</th>
                  <th>Status</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {orders.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="empty-state">
                      <p className="empty-state-title">No orders yet</p>
                      <p className="empty-state-body">Orders will appear here once buyers start purchasing.</p>
                    </td>
                  </tr>
                ) : (
                  orders.map((o) => (
                    <tr key={o.id}>
                      <td>
                        <Link
                          href={`/dashboard/orders/${o.id}`}
                          className="mono"
                          style={{ color: "var(--accent)", textDecoration: "none", fontWeight: 500 }}
                        >
                          {o.id.slice(0, 8)}…
                        </Link>
                      </td>
                      <td className="mono" style={{ fontWeight: 500 }}>
                        {formatFrom(o.total_amount, o.currency)}
                      </td>
                      <td>
                        <StatusBadge status={o.status} />
                      </td>
                      <td style={{ color: "var(--muted)" }}>
                        {new Date(o.created_at).toLocaleDateString("en-US", {
                          month: "short",
                          day: "numeric",
                        })}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right column */}
        <div style={{ display: "flex", flexDirection: "column", gap: "1.25rem" }}>
          {/* Product Status breakdown */}
          <div className="card">
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.125rem" }}>
              <span className="card-title">Product Status</span>
              <Link
                href="/dashboard/products"
                style={{ fontSize: "0.75rem", color: "var(--accent)", textDecoration: "none", fontWeight: 600, display: "inline-flex", alignItems: "center", gap: "0.2rem" }}
              >
                Manage <ArrowRight size={11} />
              </Link>
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: "0.875rem" }}>
              {[
                { label: "Active",   count: activeCount,   color: "var(--success)" },
                { label: "Draft",    count: draftCount,    color: "var(--warning)" },
                { label: "Archived", count: archivedCount, color: "var(--muted)"   },
              ].map((s) => (
                <div key={s.label}>
                  <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.375rem", alignItems: "center" }}>
                    <span style={{ fontSize: "0.8125rem", color: "var(--text-2)", display: "flex", alignItems: "center", gap: "0.5rem" }}>
                      <span style={{ display: "inline-block", width: 8, height: 8, borderRadius: "50%", background: s.color, flexShrink: 0 }} />
                      {s.label}
                    </span>
                    <span style={{ fontSize: "0.8125rem", fontWeight: 700, color: "var(--text)", fontVariantNumeric: "tabular-nums" }}>
                      {s.count}
                    </span>
                  </div>
                  <div style={{ height: 5, background: "var(--surface2)", borderRadius: 3, overflow: "hidden" }}>
                    <div
                      style={{
                        height: "100%",
                        width: productTotal ? `${(s.count / productTotal) * 100}%` : "0%",
                        background: s.color,
                        borderRadius: 3,
                        transition: "width 0.6s cubic-bezier(0.4,0,0.2,1)",
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>

            <div style={{ marginTop: "1.25rem", paddingTop: "1rem", borderTop: "1px solid var(--border)" }}>
              <Link
                href="/dashboard/products/new"
                className="btn btn-primary"
                style={{ width: "100%", justifyContent: "center", textDecoration: "none", display: "flex" }}
              >
                <Plus size={13} /> New Product
              </Link>
            </div>
          </div>

          {/* Quick links */}
          <div className="card" style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
            <span className="card-title" style={{ marginBottom: "0.25rem" }}>Quick Actions</span>
            {[
              { href: "/dashboard/products", label: "Browse all products", icon: Package },
              { href: "/dashboard/orders",   label: "View all orders",     icon: ShoppingBag },
            ].map(({ href, label, icon: Icon }) => (
              <Link
                key={href}
                href={href}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.75rem",
                  padding: "0.625rem 0.75rem",
                  borderRadius: 10,
                  textDecoration: "none",
                  background: "var(--surface2)",
                  color: "var(--text)",
                  fontSize: "0.8125rem",
                  fontWeight: 500,
                  transition: "background 0.15s",
                  border: "1px solid var(--border)",
                }}
                onMouseEnter={(e) => { e.currentTarget.style.background = "var(--surface3)"; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = "var(--surface2)"; }}
              >
                <Icon size={15} style={{ color: "var(--accent)", flexShrink: 0 }} />
                <span style={{ flex: 1 }}>{label}</span>
                <ArrowRight size={13} style={{ color: "var(--muted)" }} />
              </Link>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}
