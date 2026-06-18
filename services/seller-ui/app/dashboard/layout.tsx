"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { Splash } from "@/app/components/Skeleton";
import { getToken, clearToken } from "@/lib/auth";
import { LayoutDashboard, Package, ShoppingBag, LogOut } from "lucide-react";

type ThemeId = "walnut" | "cream" | "slate" | "solarized-dark" | "solarized-light";

const THEMES: { id: ThemeId; label: string; dot: string }[] = [
  { id: "walnut",          label: "Walnut",          dot: "#e08c42" },
  { id: "cream",           label: "Cream",           dot: "#c45e18" },
  { id: "slate",           label: "Slate",           dot: "#6b9ef0" },
  { id: "solarized-dark",  label: "Solarized Dark",  dot: "#2aa198" },
  { id: "solarized-light", label: "Solarized Light", dot: "#268bd2" },
];

const NAV = [
  { href: "/dashboard",          label: "Overview",  Icon: LayoutDashboard },
  { href: "/dashboard/products", label: "Products",  Icon: Package         },
  { href: "/dashboard/orders",   label: "Orders",    Icon: ShoppingBag     },
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

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router   = useRouter();
  const pathname = usePathname();
  const [ready, setReady]         = useState(false);
  const [theme, setTheme]         = useState<ThemeId>("walnut");
  const [navigating, setNavigating] = useState(false);
  const prevPath = useRef(pathname);
  const clock    = useClock();

  useEffect(() => {
    if (!getToken()) { router.replace("/login"); return; }
    const saved = (localStorage.getItem("zap-theme") as ThemeId) ?? "walnut";
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
      const t = setTimeout(() => setNavigating(false), 500);
      return () => clearTimeout(t);
    }
  }, [pathname]);

  if (!ready) return <Splash />;

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      {navigating && <div className="nav-progress" key={pathname} />}

      {/* ── Sidebar ─────────────────────────────────────────────────────── */}
      <aside style={{
        width: "15rem",
        flexShrink: 0,
        display: "flex",
        flexDirection: "column",
        background: "var(--surface)",
        borderRight: "1px solid var(--border)",
        boxShadow: "2px 0 12px rgba(0,0,0,0.18)",
        position: "sticky",
        top: 0,
        height: "100vh",
        overflowY: "auto",
        zIndex: 10,
      }}>
        {/* Brand */}
        <div style={{ padding: "1.5rem 1.25rem 1.125rem", borderBottom: "1px solid var(--border)" }}>
          <div style={{ fontWeight: 700, fontSize: "1rem", color: "var(--text)", letterSpacing: "-0.01em", marginBottom: "0.375rem" }}>
            Zap<span style={{ color: "var(--accent)" }}>Market</span>
          </div>
          <span style={{ fontSize: "0.6875rem", color: "var(--muted)", background: "var(--accent-bg)", border: "1px solid color-mix(in srgb, var(--accent) 25%, transparent)", borderRadius: 4, padding: "1px 6px" }}>
            Seller Portal
          </span>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, padding: "0.75rem 0.625rem" }}>
          {NAV.map(({ href, label, Icon }) => {
            const active = pathname === href || (href !== "/dashboard" && pathname.startsWith(href));
            return (
              <Link
                key={href}
                href={href}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.625rem",
                  padding: "0.5625rem 0.875rem",
                  borderRadius: 8,
                  marginBottom: 2,
                  fontSize: "0.875rem",
                  fontWeight: active ? 500 : 400,
                  color: active ? "var(--accent)" : "var(--text-2)",
                  background: active ? "var(--accent-bg)" : "transparent",
                  textDecoration: "none",
                  transition: "background 0.1s, color 0.1s",
                }}
                onMouseEnter={(e) => { if (!active) { e.currentTarget.style.background = "var(--surface2)"; e.currentTarget.style.color = "var(--text)"; } }}
                onMouseLeave={(e) => { if (!active) { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--text-2)"; } }}
              >
                <Icon size={15} strokeWidth={active ? 2.2 : 1.8} />
                {label}
              </Link>
            );
          })}
        </nav>

        {/* Footer */}
        <div style={{ borderTop: "1px solid var(--border)", padding: "0.875rem 1.25rem 1.125rem" }}>
          <div style={{ fontFamily: '"Roboto Mono", monospace', fontSize: "0.6875rem", color: "var(--muted)", marginBottom: "0.875rem", letterSpacing: "0.04em" }}>
            {clock}
          </div>
          <div style={{ fontSize: "0.6875rem", color: "var(--muted)", marginBottom: "0.4rem", fontWeight: 500 }}>Theme</div>
          <div className="theme-picker" style={{ marginBottom: "0.625rem" }}>
            {THEMES.map((t) => (
              <button key={t.id} aria-pressed={theme === t.id} title={t.label} onClick={() => applyTheme(t.id)} className="theme-dot" style={{ background: t.dot }} />
            ))}
          </div>
          <div style={{ fontSize: "0.6875rem", color: "var(--muted)", marginBottom: "0.875rem" }}>
            {THEMES.find((t) => t.id === theme)?.label}
          </div>
          <button
            onClick={() => { clearToken(); router.push("/login"); }}
            style={{ display: "flex", alignItems: "center", gap: "0.5rem", width: "100%", padding: "0.5rem 0.625rem", border: "none", borderRadius: 7, background: "transparent", cursor: "pointer", fontSize: "0.8125rem", color: "var(--muted)", transition: "color 0.1s, background 0.1s", fontFamily: "inherit", textAlign: "left" }}
            onMouseEnter={(e) => { e.currentTarget.style.color = "var(--danger)"; e.currentTarget.style.background = "color-mix(in srgb, var(--danger) 8%, transparent)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.color = "var(--muted)"; e.currentTarget.style.background = "transparent"; }}
          >
            <LogOut size={14} strokeWidth={1.8} />
            Sign out
          </button>
        </div>
      </aside>

      {/* ── Main ──────────────────────────────────────────────────────────── */}
      <main style={{ flex: 1, padding: "2rem 2.5rem", overflowY: "auto", minHeight: "100vh", background: "var(--bg)" }}>
        {children}
      </main>
    </div>
  );
}
