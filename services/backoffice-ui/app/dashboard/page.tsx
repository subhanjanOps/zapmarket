"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Tag, Package, Layers, Users, Store, ClipboardList, ShieldCheck, ChevronRight } from "lucide-react";
import { getProducts, getCategories, getSkus } from "@/lib/api";

interface Stat { label: string; value: number | "—"; href: string; accent?: boolean }

interface EntityRow {
  label: string;
  href: string;
  count: number | "—";
  icon: React.ReactNode;
}

function fmt(n: number | "—") {
  if (n === "—") return "—";
  return n.toLocaleString();
}

export default function DashboardPage() {
  const [stats, setStats] = useState<Stat[]>([]);
  const [loading, setLoading] = useState(true);
  const [today, setToday] = useState("");

  useEffect(() => {
    setToday(new Date().toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" }));
  }, []);

  useEffect(() => {
    Promise.allSettled([
      getCategories(),
      getProducts({ limit: 1 }),
      getProducts({ status: "ACTIVE", limit: 1 }),
      getSkus({ limit: 1 }),
    ]).then(([cats, all, active, skus]) => {
      setStats([
        {
          label: "Categories",
          value: cats.status === "fulfilled" ? (cats.value.total ?? cats.value.data.length) : "—",
          href: "/dashboard/categories",
        },
        {
          label: "Products",
          value: all.status === "fulfilled" ? all.value.total : "—",
          href: "/dashboard/products",
        },
        {
          label: "Active",
          value: active.status === "fulfilled" ? active.value.total : "—",
          href: "/dashboard/products?status=ACTIVE",
          accent: true,
        },
        {
          label: "SKUs",
          value: skus.status === "fulfilled" ? skus.value.total : "—",
          href: "/dashboard/skus",
        },
      ]);
    }).finally(() => setLoading(false));
  }, []);

  const catStat  = stats.find((s) => s.label === "Categories");
  const prodStat = stats.find((s) => s.label === "Products");
  const activeStat = stats.find((s) => s.label === "Active");
  const skuStat  = stats.find((s) => s.label === "SKUs");

  const catalogRows: EntityRow[] = [
    { label: "Categories",  href: "/dashboard/categories", count: catStat?.value    ?? "—", icon: <Tag size={13} strokeWidth={1.75} /> },
    { label: "Products",    href: "/dashboard/products",   count: prodStat?.value   ?? "—", icon: <Package size={13} strokeWidth={1.75} /> },
    { label: "Active prods",href: "/dashboard/products?status=ACTIVE", count: activeStat?.value ?? "—", icon: <Package size={13} strokeWidth={1.75} /> },
    { label: "SKUs",        href: "/dashboard/skus",       count: skuStat?.value    ?? "—", icon: <Layers size={13} strokeWidth={1.75} /> },
  ];

  const operationsRows: EntityRow[] = [
    { label: "Users",        href: "/dashboard/users",      count: "—", icon: <Users size={13} strokeWidth={1.75} /> },
    { label: "Sellers",      href: "/dashboard/sellers",    count: "—", icon: <Store size={13} strokeWidth={1.75} /> },
    { label: "Orders",       href: "/dashboard/orders",     count: "—", icon: <ClipboardList size={13} strokeWidth={1.75} /> },
    { label: "Moderation",   href: "/dashboard/moderation", count: "—", icon: <ShieldCheck size={13} strokeWidth={1.75} /> },
  ];

  return (
    <div className="page-content">
      {/* Header */}
      <div className="page-header">
        <div>
          <h1 className="page-title">Overview</h1>
          <p className="page-subtitle">{today}</p>
        </div>
      </div>

      {/* Stat row */}
      {loading ? (
        <div className="stat-row">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="stat-cell" style={{ pointerEvents: "none" }}>
              <div className="skeleton" style={{ height: 28, width: 64, marginBottom: "0.5rem", borderRadius: 4 }} />
              <div className="skeleton" style={{ height: 10, width: 80, borderRadius: 3 }} />
            </div>
          ))}
        </div>
      ) : (
        <div className="stat-row">
          {stats.map((s) => (
            <Link key={s.label} href={s.href} className="stat-cell">
              <div className={`stat-value${s.accent ? " stat-value-accent" : ""}`}>
                {fmt(s.value)}
              </div>
              <div className="stat-label">{s.label}</div>
            </Link>
          ))}
        </div>
      )}

      {/* Lower section — two columns */}
      <div className="quick-access-grid">
        {/* Catalog */}
        <div>
          <p className="section-label">Catalog</p>
          {catalogRows.map((row) => (
            <Link key={row.label} href={row.href} className="entity-row">
              <span style={{ color: "var(--muted)" }}>{row.icon}</span>
              <span className="entity-row-name">{row.label}</span>
              <span className="entity-row-count">{fmt(row.count)}</span>
              <ChevronRight size={12} strokeWidth={2} className="entity-row-arrow" />
            </Link>
          ))}
        </div>

        {/* Operations */}
        <div>
          <p className="section-label">Operations</p>
          {operationsRows.map((row) => (
            <Link key={row.label} href={row.href} className="entity-row">
              <span style={{ color: "var(--muted)" }}>{row.icon}</span>
              <span className="entity-row-name">{row.label}</span>
              <span className="entity-row-count">{fmt(row.count)}</span>
              <ChevronRight size={12} strokeWidth={2} className="entity-row-arrow" />
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
