"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { Splash } from "@/app/components/Skeleton";
import { getToken, clearToken } from "@/lib/auth";
import {
  LayoutDashboard, Route, Network, ScrollText,
  BarChart2, ShieldOff, LogOut, Menu, X,
} from "lucide-react";

type ThemeId = "terminal" | "phosphor" | "walnut" | "cream" | "slate" | "solarized-dark" | "solarized-light";

const THEMES: { id: ThemeId; label: string; dot: string }[] = [
  { id: "terminal",        label: "Terminal",        dot: "#06b6d4" },
  { id: "phosphor",        label: "Phosphor",        dot: "#00ff41" },
  { id: "walnut",          label: "Walnut",          dot: "#e08c42" },
  { id: "cream",           label: "Cream",           dot: "#c45e18" },
  { id: "slate",           label: "Slate",           dot: "#6b9ef0" },
  { id: "solarized-dark",  label: "Solarized Dark",  dot: "#2aa198" },
  { id: "solarized-light", label: "Solarized Light", dot: "#268bd2" },
];

const NAV = [
  { href: "/dashboard",           label: "Overview",  Icon: LayoutDashboard, idx: "01" },
  { href: "/dashboard/routes",    label: "Routes",    Icon: Route,           idx: "02" },
  { href: "/dashboard/registry",  label: "Registry",  Icon: Network,         idx: "03" },
  { href: "/dashboard/metrics",   label: "Metrics",   Icon: BarChart2,       idx: "04" },
  { href: "/dashboard/audit",     label: "Audit Log", Icon: ScrollText,      idx: "05" },
  { href: "/dashboard/blocklist", label: "Blocklist", Icon: ShieldOff,       idx: "06" },
];

function useClock() {
  const [t, setT] = useState("");
  useEffect(() => {
    const tick = () => setT(new Date().toLocaleTimeString("en-GB", { hour12: false }));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, []);
  return t;
}

function isActive(href: string, pathname: string) {
  if (href === "/dashboard") return pathname === href;
  return pathname.startsWith(href);
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router   = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [theme, setTheme] = useState<ThemeId>("terminal");
  const [navigating, setNavigating] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const prevPath = useRef(pathname);
  const clock = useClock();

  useEffect(() => {
    if (!getToken()) { router.replace("/login"); return; }
    const saved = (localStorage.getItem("zap-theme") as ThemeId) ?? "terminal";
    setTheme(saved);
    document.documentElement.setAttribute("data-theme", saved);
    setReady(true);
  }, [router]);

  function applyTheme(id: ThemeId) {
    setTheme(id);
    localStorage.setItem("zap-theme", id);
    document.documentElement.setAttribute("data-theme", id);
  }

  useEffect(() => {
    if (prevPath.current !== pathname) {
      prevPath.current = pathname;
      setNavigating(true);
      setDrawerOpen(false);
      const t = setTimeout(() => setNavigating(false), 500);
      return () => clearTimeout(t);
    }
  }, [pathname]);

  if (!ready) return <Splash />;

  const SidebarContent = () => (
    <>
      {/* Brand */}
      <div style={{
        padding: "1.375rem 1.25rem 1.125rem",
        borderBottom: "1px solid var(--border)",
      }}>
        <div style={{
          fontFamily: '"JetBrains Mono", monospace',
          fontWeight: 600,
          fontSize: "0.9375rem",
          color: "var(--text)",
          letterSpacing: "-0.01em",
          marginBottom: "0.25rem",
          display: "flex",
          alignItems: "center",
          gap: "0.375rem",
        }}>
          <span style={{ color: "var(--accent)", fontWeight: 700 }}>{">"}_</span>
          <span>Zap<span style={{ color: "var(--accent)" }}>Market</span></span>
        </div>
        <div style={{
          fontFamily: '"JetBrains Mono", monospace',
          fontSize: "0.625rem",
          color: "var(--muted)",
          letterSpacing: "0.04em",
          marginBottom: "0.625rem",
          textTransform: "uppercase",
        }}>
          API Gateway
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: "0.45rem" }}>
          <span className="status-dot status-dot-green status-dot-pulse" />
          <span style={{
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: "0.6875rem",
            color: "var(--muted)",
            letterSpacing: "0.02em",
          }}>
            gateway · online
          </span>
        </div>
      </div>

      {/* Nav */}
      <nav style={{ flex: 1, padding: "0.75rem 0.625rem" }}>
        {NAV.map(({ href, label, Icon, idx }) => {
          const active = isActive(href, pathname);
          return (
            <Link
              key={href}
              href={href}
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.625rem",
                padding: "0.5625rem 0.875rem",
                borderRadius: "8px",
                marginBottom: "2px",
                fontSize: "0.875rem",
                fontWeight: active ? 700 : 400,
                color: active ? "var(--accent)" : "var(--text-2)",
                background: active ? "var(--accent-bg)" : "transparent",
                textDecoration: "none",
                transition: "background 0.1s, color 0.1s",
                borderLeft: active ? "3px solid var(--accent)" : "3px solid transparent",
                position: "relative",
              }}
              onMouseEnter={(e) => {
                if (!active) {
                  e.currentTarget.style.background = "var(--surface2)";
                  e.currentTarget.style.color = "var(--text)";
                }
              }}
              onMouseLeave={(e) => {
                if (!active) {
                  e.currentTarget.style.background = "transparent";
                  e.currentTarget.style.color = "var(--text-2)";
                }
              }}
            >
              <span style={{
                fontFamily: '"JetBrains Mono", monospace',
                fontSize: "0.625rem",
                color: active ? "var(--accent)" : "var(--muted)",
                letterSpacing: "0.04em",
                minWidth: "1.375rem",
                flexShrink: 0,
                lineHeight: 1,
              }}>
                {idx}
              </span>
              <Icon size={15} strokeWidth={active ? 2.2 : 1.8} style={{ flexShrink: 0 }} />
              {label}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div style={{
        borderTop: "1px solid var(--border)",
        padding: "0.875rem 1.25rem 1.125rem",
      }}>
        {/* Clock */}
        <div style={{
          fontFamily: '"JetBrains Mono", monospace',
          fontSize: "0.6875rem",
          color: "var(--muted)",
          marginBottom: "0.875rem",
          letterSpacing: "0.04em",
        }}>
          {clock}
        </div>

        {/* Theme select */}
        <div style={{
          fontSize: "0.6875rem",
          color: "var(--muted)",
          marginBottom: "0.4rem",
          fontWeight: 500,
        }}>
          Theme
        </div>
        <div className="theme-picker" style={{ marginBottom: "0.625rem" }}>
          {THEMES.map((t) => (
            <button
              key={t.id}
              aria-pressed={theme === t.id}
              title={t.label}
              onClick={() => applyTheme(t.id)}
              className="theme-dot"
              style={{ background: t.dot }}
            />
          ))}
        </div>
        <div style={{
          fontSize: "0.6875rem",
          color: "var(--muted)",
          marginBottom: "0.875rem",
        }}>
          {THEMES.find((t) => t.id === theme)?.label}
        </div>

        {/* Sign out */}
        <button
          onClick={() => { clearToken(); router.push("/login"); }}
          style={{
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
            width: "100%",
            padding: "0.5rem 0.625rem",
            border: "none",
            borderRadius: "7px",
            background: "transparent",
            cursor: "pointer",
            fontSize: "0.8125rem",
            color: "var(--muted)",
            transition: "color 0.1s, background 0.1s",
            fontFamily: '"Space Grotesk", system-ui, sans-serif',
            textAlign: "left",
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.color = "var(--danger)";
            e.currentTarget.style.background = "color-mix(in srgb, var(--danger) 8%, transparent)";
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.color = "var(--muted)";
            e.currentTarget.style.background = "transparent";
          }}
        >
          <LogOut size={14} strokeWidth={1.8} />
          Sign out
        </button>
      </div>
    </>
  );

  return (
    <div style={{ display: "flex", minHeight: "100dvh" }}>
      {navigating && <div className="nav-progress" key={pathname} />}

      {/* ── Desktop Sidebar ───────────────────────────────────────────────── */}
      <aside className="dash-sidebar" style={{
        width: "15rem",
        flexShrink: 0,
        display: "flex",
        flexDirection: "column",
        background: "var(--surface)",
        borderRight: "1px solid var(--border)",
        boxShadow: "2px 0 12px rgba(0,0,0,0.18)",
        position: "sticky",
        top: 0,
        height: "100dvh",
        overflowY: "auto",
        zIndex: 10,
      }}>
        <SidebarContent />
      </aside>

      {/* ── Mobile Top Bar ────────────────────────────────────────────────── */}
      <div className="dash-topbar" style={{
        display: "none",
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        height: "3.25rem",
        background: "var(--surface)",
        borderBottom: "1px solid var(--border)",
        alignItems: "center",
        padding: "0 1rem",
        gap: "0.75rem",
        zIndex: 50,
        boxShadow: "0 2px 12px rgba(0,0,0,0.18)",
      }}>
        <button
          onClick={() => setDrawerOpen(true)}
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            width: "2rem",
            height: "2rem",
            background: "transparent",
            border: "none",
            cursor: "pointer",
            color: "var(--text-2)",
            borderRadius: "6px",
            flexShrink: 0,
          }}
          aria-label="Open navigation"
        >
          <Menu size={18} />
        </button>
        <div style={{
          fontFamily: '"JetBrains Mono", monospace',
          fontWeight: 600,
          fontSize: "0.875rem",
          color: "var(--text)",
          display: "flex",
          alignItems: "center",
          gap: "0.375rem",
        }}>
          <span style={{ color: "var(--accent)" }}>{">"}_</span>
          Zap<span style={{ color: "var(--accent)" }}>Market</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: "0.5rem" }}>
          <span className="status-dot status-dot-green status-dot-pulse" />
          <span style={{
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: "0.625rem",
            color: "var(--muted)",
          }}>
            {clock}
          </span>
        </div>
      </div>

      {/* ── Mobile Drawer Overlay ─────────────────────────────────────────── */}
      {drawerOpen && (
        <div
          className="dash-overlay"
          onClick={() => setDrawerOpen(false)}
          style={{
            position: "fixed",
            inset: 0,
            background: "rgba(0,0,0,0.6)",
            backdropFilter: "blur(2px)",
            zIndex: 60,
          }}
        />
      )}

      {/* ── Mobile Slide-out Drawer ───────────────────────────────────────── */}
      <div
        className="dash-drawer"
        style={{
          position: "fixed",
          top: 0,
          left: 0,
          bottom: 0,
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
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              width: "2rem",
              height: "2rem",
              background: "var(--surface2)",
              border: "1px solid var(--border)",
              borderRadius: "6px",
              cursor: "pointer",
              color: "var(--text-2)",
            }}
            aria-label="Close navigation"
          >
            <X size={16} />
          </button>
        </div>
        <SidebarContent />
      </div>

      {/* ── Main Content ─────────────────────────────────────────────────── */}
      <main style={{
        flex: 1,
        padding: "clamp(1rem, 4vw, 2rem)",
        overflowY: "auto",
        minHeight: "100dvh",
        background: "var(--bg)",
        paddingTop: "clamp(1rem, 4vw, 2rem)",
      }}>
        {children}
      </main>

      {/* ── Mobile Bottom Navigation ──────────────────────────────────────── */}
      <nav className="dash-bottom-nav" style={{
        display: "none",
        position: "fixed",
        bottom: 0,
        left: 0,
        right: 0,
        background: "var(--surface)",
        borderTop: "1px solid var(--border)",
        zIndex: 50,
        padding: "0.25rem 0 0.5rem",
      }}>
        <div style={{
          display: "grid",
          gridTemplateColumns: "repeat(6, 1fr)",
          gap: 0,
        }}>
          {NAV.map(({ href, label, Icon }) => {
            const active = isActive(href, pathname);
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
                  minHeight: "3rem",
                }}
              >
                <Icon size={18} strokeWidth={active ? 2.2 : 1.8} />
                <span style={{
                  fontFamily: '"JetBrains Mono", monospace',
                  fontSize: "0.5rem",
                  letterSpacing: "0.03em",
                  lineHeight: 1,
                }}>
                  {label.split(" ")[0]}
                </span>
              </Link>
            );
          })}
        </div>
      </nav>
    </div>
  );
}
