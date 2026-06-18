"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import { getProducts, getCategories, getSkus } from "@/lib/api";

interface StatCard { label: string; value: string | number; href: string; accent?: boolean }

export default function DashboardPage() {
  const [stats, setStats] = useState<StatCard[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = getToken() ?? undefined;
    Promise.allSettled([
      getCategories(),
      getProducts({ limit: 1 }, token),
      getProducts({ status: "ACTIVE", limit: 1 }, token),
      getSkus({ limit: 1 }, token),
    ]).then(([cats, all, active, skus]) => {
      setStats([
        {
          label: "Categories",
          value: cats.status === "fulfilled" ? cats.value.total ?? cats.value.data.length : "—",
          href: "/dashboard/categories",
        },
        {
          label: "Total Products",
          value: all.status === "fulfilled" ? all.value.total : "—",
          href: "/dashboard/products",
        },
        {
          label: "Active Products",
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

  return (
    <div style={{ padding: "2rem" }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-subtitle">Catalog overview</p>
        </div>
      </div>

      {/* Stat cards */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
          gap: "1rem",
          marginBottom: "2rem",
        }}
      >
        {loading
          ? [1, 2, 3, 4].map((i) => (
              <div key={i} className="card" style={{ padding: "1.25rem" }}>
                <div className="skeleton" style={{ height: 11, width: 80, marginBottom: "0.75rem" }} />
                <div className="skeleton" style={{ height: 28, width: 56 }} />
              </div>
            ))
          : stats.map((s) => (
              <Link key={s.label} href={s.href} style={{ textDecoration: "none" }}>
                <div
                  className="card"
                  style={{
                    padding: "1.25rem",
                    cursor: "pointer",
                    transition: "border-color 0.12s",
                    borderColor: s.accent ? "var(--accent)" : undefined,
                  }}
                >
                  <div
                    style={{
                      fontSize: "0.6875rem",
                      fontWeight: 500,
                      color: "var(--muted)",
                      textTransform: "uppercase",
                      letterSpacing: "0.07em",
                      marginBottom: "0.5rem",
                    }}
                  >
                    {s.label}
                  </div>
                  <div
                    style={{
                      fontSize: "1.625rem",
                      fontWeight: 700,
                      color: s.accent ? "var(--accent)" : "var(--text)",
                      lineHeight: 1,
                    }}
                  >
                    {s.value}
                  </div>
                </div>
              </Link>
            ))}
      </div>

      {/* Quick links */}
      <div className="card" style={{ padding: "1.25rem 1.5rem" }}>
        <div className="card-title" style={{ marginBottom: "1rem" }}>Quick actions</div>
        <div style={{ display: "flex", gap: "0.625rem", flexWrap: "wrap" }}>
          <Link href="/dashboard/categories" className="btn btn-secondary">
            Manage categories
          </Link>
          <Link href="/dashboard/products" className="btn btn-secondary">
            Browse products
          </Link>
          <Link href="/dashboard/moderation" className="btn btn-secondary">
            Review moderation queue
          </Link>
        </div>
      </div>
    </div>
  );
}
