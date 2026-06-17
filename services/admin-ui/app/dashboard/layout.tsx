"use client";
import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { getToken, clearToken } from "@/lib/auth";
import { Route, Network, ScrollText, LogOut } from "lucide-react";

const NAV = [
  { href: "/dashboard",          label: "routes",   icon: Route      },
  { href: "/dashboard/registry", label: "registry", icon: Network    },
  { href: "/dashboard/audit",    label: "audit",    icon: ScrollText },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!getToken()) router.replace("/login");
    else setReady(true);
  }, [router]);

  if (!ready) return null;

  function logout() {
    clearToken();
    router.push("/login");
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
        {/* Logo mark */}
        <div style={{ padding: "0 1.25rem", marginBottom: "2rem" }}>
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

        {/* Separator + sign out */}
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
