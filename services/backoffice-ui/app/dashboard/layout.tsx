"use client";

import { useEffect, useRef, useState } from "react";
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
  Menu,
  X,
  type LucideIcon,
} from "lucide-react";
import { isAuthenticated, clearToken } from "@/lib/auth";
import { CommandPalette } from "@/app/components/CommandPalette";

const THEMES = [
  { name: "enterprise-dark",  color: "#7c3aed", label: "Dark"           },
  { name: "enterprise-light", color: "#6d28d9", label: "Light"          },
  { name: "slate",            color: "#6b9ef0", label: "Slate"          },
  { name: "walnut",           color: "#e08c42", label: "Walnut"         },
  { name: "cream",            color: "#c45e18", label: "Cream"          },
  { name: "solarized-dark",   color: "#2aa198", label: "Solarized Dark" },
  { name: "solarized-light",  color: "#268bd2", label: "Solarized Light"},
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

const BOTTOM_NAV: NavItem[] = [
  { href: "/dashboard",            icon: LayoutDashboard, label: "Dashboard" },
  { href: "/dashboard/products",   icon: Package,         label: "Products" },
  { href: "/dashboard/users",      icon: Users,           label: "Users" },
  { href: "/dashboard/orders",     icon: ClipboardList,   label: "Orders" },
  { href: "/dashboard/moderation", icon: ShieldCheck,     label: "Moderate" },
];

interface SidebarProps {
  theme: string;
  isActive: (href: string) => boolean;
  applyTheme: (t: string) => void;
  signOut: () => void;
}

function SidebarContent({ theme, isActive, applyTheme, signOut }: SidebarProps) {
  return (
    <>
      {/* Wordmark */}
      <div style={{
        padding: "1.125rem 1rem 1rem",
        borderBottom: "1px solid var(--border)",
        display: "flex",
        alignItems: "center",
        gap: "0.5625rem",
      }}>
        <div style={{
          width: 24, height: 24, borderRadius: 5,
          background: "var(--accent)",
          display: "flex", alignItems: "center", justifyContent: "center",
          fontSize: 11, fontWeight: 800, color: "var(--accent-text)",
          flexShrink: 0, letterSpacing: "-0.03em",
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

      {/* Bottom: command palette + theme + sign out */}
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
            display: "flex", alignItems: "center", gap: "0.375rem",
            background: "none", border: "none", cursor: "pointer",
            padding: "0.25rem 0", width: "100%",
          }}
          title="Open command palette"
        >
          <span style={{ fontSize: "0.6875rem", color: "var(--muted)", flex: 1, textAlign: "left" }}>Command palette</span>
          <span className="kbd">⌘K</span>
        </button>

        <div>
          <label
            htmlFor="theme-select"
            style={{ display: "block", fontSize: "0.625rem", fontWeight: 600, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: "0.25rem" }}
          >
            Theme
          </label>
          <div style={{ position: "relative" }}>
            <div style={{
              position: "absolute", left: "0.5rem", top: "50%", transform: "translateY(-50%)",
              width: 8, height: 8, borderRadius: "50%",
              background: THEMES.find((t) => t.name === theme)?.color ?? "var(--accent)",
              pointerEvents: "none",
            }} />
            <select
              id="theme-select"
              className="input"
              value={theme}
              onChange={(e) => applyTheme(e.target.value)}
              style={{ paddingLeft: "1.5rem", fontSize: "0.75rem", height: 30, paddingTop: 0, paddingBottom: 0, cursor: "pointer" }}
              aria-label="Select theme"
            >
              {THEMES.map((t) => (
                <option key={t.name} value={t.name}>{t.label}</option>
              ))}
            </select>
          </div>
        </div>

        <button
          className="btn btn-ghost"
          onClick={signOut}
          style={{
            width: "100%", justifyContent: "flex-start",
            fontSize: "0.75rem", padding: "0.375rem 0.375rem",
            gap: "0.5rem", color: "var(--muted)", borderColor: "transparent",
          }}
        >
          <LogOut size={13} strokeWidth={1.75} />
          Sign out
        </button>
      </div>
    </>
  );
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router   = useRouter();
  const pathname = usePathname();
  const [theme, setTheme] = useState("enterprise-dark");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const prevPath = useRef(pathname);

  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    const saved = localStorage.getItem("bo_theme") ?? "enterprise-dark";
    setTheme(saved);
    document.documentElement.setAttribute("data-theme", saved);
  }, [router]);

  useEffect(() => {
    if (prevPath.current !== pathname) {
      prevPath.current = pathname;
      setDrawerOpen(false);
    }
  }, [pathname]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") setDrawerOpen(false); };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    document.body.style.overflow = drawerOpen ? "hidden" : "";
    return () => { document.body.style.overflow = ""; };
  }, [drawerOpen]);

  function applyTheme(t: string) {
    setTheme(t);
    localStorage.setItem("bo_theme", t);
    document.documentElement.setAttribute("data-theme", t);
  }

  async function signOut() {
    await clearToken();
    router.push("/login");
  }

  function isActive(href: string) {
    if (href === "/dashboard") return pathname === "/dashboard";
    return pathname.startsWith(href);
  }

  const sidebarProps: SidebarProps = { theme, isActive, applyTheme, signOut };

  return (
    <div style={{ display: "flex", minHeight: "100dvh" }}>

      {/* ── Desktop Sidebar ────────────────────────────────────────────────── */}
      <aside className="dash-sidebar" style={{
        width: 216,
        flexShrink: 0,
        background: "var(--surface)",
        borderRight: "1px solid var(--border)",
        display: "flex",
        flexDirection: "column",
        position: "sticky",
        top: 0,
        height: "100dvh",
        overflowY: "auto",
        zIndex: 10,
      }}>
        <SidebarContent {...sidebarProps} />
      </aside>

      {/* ── Mobile Top Bar ─────────────────────────────────────────────────── */}
      <div className="dash-topbar" style={{
        display: "none",
        position: "fixed",
        top: 0, left: 0, right: 0,
        height: "3.25rem",
        background: "var(--surface)",
        borderBottom: "1px solid var(--border)",
        alignItems: "center",
        padding: "0 1rem",
        gap: "0.75rem",
        zIndex: 50,
        boxShadow: "0 2px 8px rgba(0,0,0,0.12)",
      }}>
        <button
          onClick={() => setDrawerOpen(true)}
          style={{
            display: "flex", alignItems: "center", justifyContent: "center",
            width: "2rem", height: "2rem",
            background: "transparent", border: "none", cursor: "pointer",
            color: "var(--text-2)", borderRadius: "6px", flexShrink: 0,
          }}
          aria-label="Open navigation"
        >
          <Menu size={18} />
        </button>
        <div style={{ display: "flex", alignItems: "center", gap: "0.5625rem" }}>
          <div style={{
            width: 20, height: 20, borderRadius: 4,
            background: "var(--accent)",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 9, fontWeight: 800, color: "var(--accent-text)", letterSpacing: "-0.03em",
          }}>ZM</div>
          <span style={{ fontSize: "0.875rem", fontWeight: 600, color: "var(--text)", letterSpacing: "-0.01em" }}>
            ZapMarket
          </span>
        </div>
      </div>

      {/* ── Mobile Drawer Overlay ──────────────────────────────────────────── */}
      {drawerOpen && (
        <div
          className="dash-overlay"
          onClick={() => setDrawerOpen(false)}
          style={{
            position: "fixed", inset: 0,
            background: "rgba(0,0,0,0.55)",
            backdropFilter: "blur(2px)",
            zIndex: 60,
          }}
        />
      )}

      {/* ── Mobile Slide-out Drawer ────────────────────────────────────────── */}
      <div
        className="dash-drawer"
        style={{
          position: "fixed",
          top: 0, left: 0, bottom: 0,
          width: "16rem",
          background: "var(--surface)",
          borderRight: "1px solid var(--border)",
          display: "flex",
          flexDirection: "column",
          zIndex: 70,
          transform: drawerOpen ? "translateX(0)" : "translateX(-100%)",
          transition: "transform 0.22s ease",
          overflowY: "auto",
        }}
      >
        <div style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "flex-end",
          padding: "0.75rem 0.875rem",
          borderBottom: "1px solid var(--border)",
        }}>
          <button
            className="sidebar-close-btn"
            onClick={() => setDrawerOpen(false)}
            style={{
              display: "flex", alignItems: "center", justifyContent: "center",
              width: "2rem", height: "2rem",
              background: "var(--surface2)", border: "1px solid var(--border)",
              borderRadius: "6px", cursor: "pointer", color: "var(--text-2)",
            }}
            aria-label="Close navigation"
          >
            <X size={16} />
          </button>
        </div>
        <SidebarContent {...sidebarProps} />
      </div>

      {/* ── Main ───────────────────────────────────────────────────────────── */}
      <main style={{ flex: 1, minWidth: 0, overflowX: "hidden" }}>
        {children}
      </main>

      {/* ── Mobile Bottom Navigation ───────────────────────────────────────── */}
      <nav className="dash-bottom-nav" style={{
        display: "none",
        position: "fixed",
        bottom: 0, left: 0, right: 0,
        background: "var(--surface)",
        borderTop: "1px solid var(--border)",
        zIndex: 50,
        paddingBottom: "env(safe-area-inset-bottom)",
      }}>
        <div style={{
          display: "grid",
          gridTemplateColumns: `repeat(${BOTTOM_NAV.length}, 1fr)`,
        }}>
          {BOTTOM_NAV.map(({ href, label, icon: Icon }) => {
            const active = isActive(href);
            return (
              <Link
                key={href}
                href={href}
                style={{
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                  justifyContent: "center",
                  gap: "0.2rem",
                  padding: "0.5rem 0.25rem",
                  color: active ? "var(--accent)" : "var(--muted)",
                  textDecoration: "none",
                  transition: "color 0.1s",
                  minHeight: "3.25rem",
                }}
              >
                <Icon size={18} strokeWidth={active ? 2 : 1.75} />
                <span style={{ fontSize: "0.5625rem", letterSpacing: "0.02em", lineHeight: 1 }}>
                  {label}
                </span>
              </Link>
            );
          })}
        </div>
      </nav>

      {/* ── Command palette ────────────────────────────────────────────────── */}
      <CommandPalette />
    </div>
  );
}
