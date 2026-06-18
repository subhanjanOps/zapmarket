"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { getToken, clearToken } from "@/lib/auth";

const THEMES = [
  { name: "walnut",          color: "#e08c42" },
  { name: "cream",           color: "#c45e18" },
  { name: "slate",           color: "#6b9ef0" },
  { name: "solarized-dark",  color: "#2aa198" },
  { name: "solarized-light", color: "#268bd2" },
];

const NAV = [
  {
    section: "Overview",
    items: [
      { href: "/dashboard",            icon: "⊞", label: "Dashboard" },
    ],
  },
  {
    section: "Catalog",
    items: [
      { href: "/dashboard/categories", icon: "◉", label: "Categories" },
      { href: "/dashboard/products",   icon: "▣", label: "Products" },
      { href: "/dashboard/skus",       icon: "◈", label: "SKUs" },
    ],
  },
  {
    section: "Users & Orders",
    items: [
      { href: "/dashboard/users",      icon: "◎", label: "Users" },
      { href: "/dashboard/sellers",    icon: "⬡", label: "Sellers" },
      { href: "/dashboard/orders",     icon: "≡", label: "Orders" },
    ],
  },
  {
    section: "Compliance",
    items: [
      { href: "/dashboard/moderation", icon: "✓", label: "Moderation" },
    ],
  },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router   = useRouter();
  const pathname = usePathname();
  const [theme, setTheme] = useState("walnut");

  useEffect(() => {
    const saved = localStorage.getItem("bo_theme");
    if (saved) setTheme(saved);
  }, []);

  useEffect(() => {
    const token = getToken();
    if (!token) router.replace("/login");
  }, [router]);

  function applyTheme(t: string) {
    setTheme(t);
    localStorage.setItem("bo_theme", t);
    document.documentElement.setAttribute("data-theme", t);
  }

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  function signOut() {
    clearToken();
    router.push("/login");
  }

  function isActive(href: string) {
    if (href === "/dashboard") return pathname === "/dashboard";
    return pathname.startsWith(href);
  }

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      {/* ── Sidebar ──────────────────────────────────────────── */}
      <aside
        style={{
          width: 220,
          flexShrink: 0,
          background: "var(--surface)",
          borderRight: "1px solid var(--border)",
          display: "flex",
          flexDirection: "column",
          position: "sticky",
          top: 0,
          height: "100vh",
          overflowY: "auto",
        }}
      >
        {/* Wordmark */}
        <div
          style={{
            padding: "1rem 0.875rem 0.75rem",
            borderBottom: "1px solid var(--border)",
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
          }}
        >
          <span
            style={{
              width: 26,
              height: 26,
              borderRadius: 6,
              background: "var(--accent)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 13,
              fontWeight: 700,
              color: "var(--accent-text)",
              flexShrink: 0,
            }}
          >
            Z
          </span>
          <div>
            <div style={{ fontSize: "0.8125rem", fontWeight: 700, color: "var(--text)", lineHeight: 1.2 }}>
              ZapMarket
            </div>
            <div style={{ fontSize: "0.625rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.06em" }}>
              Backoffice
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, padding: "0.5rem 0" }}>
          {NAV.map((group) => (
            <div key={group.section}>
              <div className="nav-section">{group.section}</div>
              {group.items.map((item) => {
                const active = isActive(item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "0.5rem",
                      padding: "0.4375rem 0.875rem",
                      margin: "1px 0.375rem",
                      borderRadius: 6,
                      fontSize: "0.8125rem",
                      fontWeight: active ? 600 : 400,
                      color: active ? "var(--accent)" : "var(--text-2)",
                      background: active ? "var(--accent-bg)" : "transparent",
                      textDecoration: "none",
                      transition: "background 0.1s, color 0.1s",
                    }}
                    onMouseEnter={(e) => {
                      if (!active) {
                        (e.currentTarget as HTMLElement).style.background = "var(--surface2)";
                        (e.currentTarget as HTMLElement).style.color = "var(--text)";
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (!active) {
                        (e.currentTarget as HTMLElement).style.background = "transparent";
                        (e.currentTarget as HTMLElement).style.color = "var(--text-2)";
                      }
                    }}
                  >
                    <span style={{ fontSize: 14, opacity: 0.8 }}>{item.icon}</span>
                    {item.label}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>

        {/* Bottom: theme picker + sign out */}
        <div
          style={{
            padding: "0.75rem 0.875rem",
            borderTop: "1px solid var(--border)",
            display: "flex",
            flexDirection: "column",
            gap: "0.625rem",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "0.375rem" }}>
            <span style={{ fontSize: "0.6875rem", color: "var(--muted)", marginRight: "0.25rem" }}>Theme</span>
            {THEMES.map((t) => (
              <button
                key={t.name}
                className="theme-dot"
                aria-label={t.name}
                aria-pressed={theme === t.name}
                onClick={() => applyTheme(t.name)}
                style={{ background: t.color }}
              />
            ))}
          </div>
          <button
            className="btn btn-ghost"
            onClick={signOut}
            style={{ width: "100%", justifyContent: "flex-start", fontSize: "0.8rem", padding: "0.375rem 0.5rem" }}
          >
            <span>⏻</span> Sign out
          </button>
        </div>
      </aside>

      {/* ── Main ─────────────────────────────────────────────── */}
      <main style={{ flex: 1, minWidth: 0, overflowX: "hidden" }}>
        {children}
      </main>
    </div>
  );
}
