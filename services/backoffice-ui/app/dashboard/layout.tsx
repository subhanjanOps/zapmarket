"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import {
  LayoutDashboard,
  Tag,
  Package,
  Layers,
  Users,
  Store,
  ClipboardList,
  ShieldCheck,
  LogOut,
  type LucideIcon,
} from "lucide-react";
import { getToken, clearToken } from "@/lib/auth";
import { CommandPalette } from "@/app/components/CommandPalette";

const THEMES = [
  { name: "walnut",          color: "#e08c42", label: "Walnut" },
  { name: "cream",           color: "#c45e18", label: "Cream" },
  { name: "slate",           color: "#6b9ef0", label: "Slate" },
  { name: "solarized-dark",  color: "#2aa198", label: "Solarized Dark" },
  { name: "solarized-light", color: "#268bd2", label: "Solarized Light" },
];

interface NavItem { href: string; icon: LucideIcon; label: string }
interface NavGroup { section: string; items: NavItem[] }

const NAV: NavGroup[] = [
  {
    section: "Overview",
    items: [
      { href: "/dashboard",            icon: LayoutDashboard, label: "Dashboard" },
    ],
  },
  {
    section: "Catalog",
    items: [
      { href: "/dashboard/categories", icon: Tag,     label: "Categories" },
      { href: "/dashboard/products",   icon: Package, label: "Products" },
      { href: "/dashboard/skus",       icon: Layers,  label: "SKUs" },
    ],
  },
  {
    section: "Users & Orders",
    items: [
      { href: "/dashboard/users",   icon: Users,         label: "Users" },
      { href: "/dashboard/sellers", icon: Store,         label: "Sellers" },
      { href: "/dashboard/orders",  icon: ClipboardList, label: "Orders" },
    ],
  },
  {
    section: "Compliance",
    items: [
      { href: "/dashboard/moderation", icon: ShieldCheck, label: "Moderation" },
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
      {/* ── Sidebar ──────────────────────────────────────────────────── */}
      <aside style={{
        width: 216,
        flexShrink: 0,
        background: "var(--surface)",
        borderRight: "1px solid var(--border)",
        display: "flex",
        flexDirection: "column",
        position: "sticky",
        top: 0,
        height: "100vh",
        overflowY: "auto",
      }}>

        {/* Wordmark */}
        <div style={{
          padding: "1.125rem 1rem 1rem",
          borderBottom: "1px solid var(--border)",
          display: "flex",
          alignItems: "center",
          gap: "0.5625rem",
        }}>
          <div style={{
            width: 24,
            height: 24,
            borderRadius: 5,
            background: "var(--accent)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: 11,
            fontWeight: 800,
            color: "var(--accent-text)",
            flexShrink: 0,
            letterSpacing: "-0.03em",
          }}>
            ZM
          </div>
          <div>
            <div style={{ fontSize: "0.8125rem", fontWeight: 600, color: "var(--text)", lineHeight: 1.25, letterSpacing: "-0.01em" }}>
              ZapMarket
            </div>
            <div style={{ fontSize: "0.5625rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.1em", marginTop: 1 }}>
              Admin Console
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, paddingBottom: "0.5rem" }}>
          {NAV.map((group) => (
            <div key={group.section}>
              <div className="nav-section">{group.section}</div>
              {group.items.map((item) => {
                const active = isActive(item.href);
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`nav-item${active ? " nav-item-active" : ""}`}
                  >
                    <Icon size={14} strokeWidth={active ? 2 : 1.75} className="nav-icon" />
                    {item.label}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>

        {/* Bottom: theme picker + sign out */}
        <div style={{
          padding: "0.75rem 1rem",
          borderTop: "1px solid var(--border)",
          display: "flex",
          flexDirection: "column",
          gap: "0.5rem",
        }}>
          {/* ⌘K hint */}
          <button
            onClick={() => {
              window.dispatchEvent(
                Object.assign(new KeyboardEvent("keydown", { key: "k", bubbles: true }), { metaKey: true })
              );
            }}
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.375rem",
              background: "none",
              border: "none",
              cursor: "pointer",
              padding: "0.25rem 0",
              width: "100%",
            }}
            title="Open command palette"
          >
            <span style={{ fontSize: "0.6875rem", color: "var(--muted)", flex: 1, textAlign: "left" }}>Command palette</span>
            <span className="kbd">⌘K</span>
          </button>

          <div style={{ display: "flex", alignItems: "center", gap: "0.375rem" }}>
            <span style={{ fontSize: "0.625rem", fontWeight: 500, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", marginRight: "0.125rem" }}>
              Theme
            </span>
            {THEMES.map((t) => (
              <button
                key={t.name}
                className="theme-dot"
                aria-label={t.label}
                aria-pressed={theme === t.name}
                title={t.label}
                onClick={() => applyTheme(t.name)}
                style={{ background: t.color }}
              />
            ))}
          </div>
          <button
            className="btn btn-ghost"
            onClick={signOut}
            style={{
              width: "100%",
              justifyContent: "flex-start",
              fontSize: "0.75rem",
              padding: "0.375rem 0.375rem",
              gap: "0.5rem",
              color: "var(--muted)",
              borderColor: "transparent",
            }}
          >
            <LogOut size={13} strokeWidth={1.75} />
            Sign out
          </button>
        </div>
      </aside>

      {/* ── Main ─────────────────────────────────────────────────────── */}
      <main style={{ flex: 1, minWidth: 0, overflowX: "hidden" }}>
        {children}
      </main>

      {/* ── Command palette ──────────────────────────────────────────── */}
      <CommandPalette />
    </div>
  );
}
