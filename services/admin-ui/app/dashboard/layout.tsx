"use client";
import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { getToken, clearToken } from "@/lib/auth";
import { LayoutDashboard, Route, Network, ScrollText, BarChart2, ShieldOff, LogOut, Sun, Moon } from "lucide-react";

const NAV = [
  { href: "/dashboard",             label: "overview",   icon: LayoutDashboard },
  { href: "/dashboard/routes",      label: "routes",     icon: Route           },
  { href: "/dashboard/registry",    label: "registry",   icon: Network         },
  { href: "/dashboard/audit",       label: "audit",      icon: ScrollText      },
  { href: "/dashboard/metrics",     label: "metrics",    icon: BarChart2       },
  { href: "/dashboard/blocklist",   label: "blocklist",  icon: ShieldOff       },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [theme, setTheme] = useState<"dark" | "light">("dark");

  useEffect(() => {
    if (!getToken()) router.replace("/login");
    else setReady(true);
  }, [router]);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  if (!ready) return null;

  function logout() {
    clearToken();
    router.push("/login");
  }

  function toggleTheme() {
    setTheme((t) => (t === "dark" ? "light" : "dark"));
  }

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside style={{
        width: "13rem",
        flexShrink: 0,
        display: "flex",
        flexDirection: "column",
        paddingTop: "1.5rem",
        paddingBottom: "1.25rem",
        background: "var(--surface)",
        borderRight: "1px solid var(--border)",
      }}>
        {/* Logo + theme toggle */}
        <div style={{ padding: "0 1.25rem", marginBottom: "2rem", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div>
            <span className="mono" style={{ fontSize: "0.9375rem", fontWeight: 600, letterSpacing: "-0.01em", color: "var(--text)" }}>
              ZAP
            </span>
            <span className="mono" style={{ fontSize: "0.9375rem", fontWeight: 600, color: "var(--accent)" }}>
              /
            </span>
            <span className="mono" style={{ fontSize: "0.9375rem", fontWeight: 600, letterSpacing: "-0.01em", color: "var(--muted)" }}>
              GATEWAY
            </span>
          </div>
          <button
            onClick={toggleTheme}
            title={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
            style={{
              background: "transparent",
              border: "none",
              cursor: "pointer",
              color: "var(--muted)",
              padding: "0.25rem",
              borderRadius: "3px",
              display: "flex",
              alignItems: "center",
            }}
            onMouseEnter={(e) => (e.currentTarget.style.color = "var(--accent)")}
            onMouseLeave={(e) => (e.currentTarget.style.color = "var(--muted)")}
          >
            {theme === "dark" ? <Sun size={14} /> : <Moon size={14} />}
          </button>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, display: "flex", flexDirection: "column", gap: "2px", padding: "0 0.5rem" }}>
          {NAV.map(({ href, label, icon: Icon }) => {
            const active = pathname === href;
            return (
              <Link
                key={href}
                href={href}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.625rem",
                  padding: "0.5rem 0.75rem",
                  borderRadius: "3px",
                  fontSize: "0.8125rem",
                  fontFamily: '"JetBrains Mono", monospace',
                  fontWeight: active ? 500 : 400,
                  color: active ? "var(--accent)" : "var(--muted)",
                  background: active ? "rgba(0, 200, 150, 0.07)" : "transparent",
                  borderLeft: active ? "2px solid var(--accent)" : "2px solid transparent",
                  transition: "color 0.12s, background 0.12s",
                  textDecoration: "none",
                }}
              >
                <Icon size={14} />
                {label}
              </Link>
            );
          })}
        </nav>

        {/* Sign out */}
        <div style={{ padding: "0 0.5rem", borderTop: "1px solid var(--border)", paddingTop: "0.75rem", marginTop: "0.5rem" }}>
          <button
            onClick={logout}
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.625rem",
              padding: "0.5rem 0.75rem",
              borderRadius: "3px",
              fontSize: "0.8125rem",
              fontFamily: '"JetBrains Mono", monospace',
              color: "var(--muted)",
              background: "transparent",
              border: "none",
              cursor: "pointer",
              width: "100%",
              transition: "color 0.12s",
            }}
            onMouseEnter={(e) => (e.currentTarget.style.color = "var(--danger)")}
            onMouseLeave={(e) => (e.currentTarget.style.color = "var(--muted)")}
          >
            <LogOut size={14} />
            sign out
          </button>
        </div>
      </aside>

      {/* Main */}
      <main style={{ flex: 1, padding: "2rem 2.5rem", overflowY: "auto", minHeight: "100vh" }}>
        {children}
      </main>
    </div>
  );
}
