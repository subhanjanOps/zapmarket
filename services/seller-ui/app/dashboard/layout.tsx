"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { Splash } from "@/app/components/Skeleton";
import { getMe } from "@/lib/api";
import { logout } from "@/lib/auth";
import { LayoutDashboard, Package, ShoppingBag, LogOut, Clock, Menu, X, Settings, ChevronDown } from "lucide-react";
import { useCurrency } from "@/lib/currency";

type ThemeId = "vibrant" | "night" | "walnut" | "cream" | "slate" | "solarized-dark" | "solarized-light";

const THEMES: { id: ThemeId; label: string; color: string }[] = [
  { id: "vibrant",         label: "Vibrant",         color: "#e11d48" },
  { id: "night",           label: "Night",           color: "#fb7185" },
  { id: "walnut",          label: "Walnut",          color: "#e08c42" },
  { id: "cream",           label: "Cream",           color: "#c45e18" },
  { id: "slate",           label: "Slate",           color: "#6b9ef0" },
  { id: "solarized-dark",  label: "Solarized Dark",  color: "#2aa198" },
  { id: "solarized-light", label: "Solarized Light", color: "#268bd2" },
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
  const [ready, setReady]           = useState(false);
  const [theme, setTheme]           = useState<ThemeId>("vibrant");
  const [navigating, setNavigating] = useState(false);
  const [sellerStatus, setSellerStatus] = useState<string | null>(null);
  const [authError, setAuthError]   = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [userEmail, setUserEmail]   = useState<string | null>(null);
  const prevPath = useRef(pathname);
  const clock    = useClock();
  const { currency, setCurrency, currencies, ratesLoading, ratesDate, stale } = useCurrency();

  useEffect(() => {
    const saved = (localStorage.getItem("zap-theme") as ThemeId) ?? "vibrant";
    setTheme(saved);
    document.documentElement.setAttribute("data-theme", saved);

    getMe()
      .then((me) => {
        setSellerStatus(me.user.seller_status ?? "APPROVED");
        setUserEmail(me.user.email ?? null);
        setReady(true);
      })
      .catch((err: Error) => {
        if (err.message.startsWith("HTTP 401") || err.message === "Unauthorized") {
          router.replace("/login");
        } else {
          setAuthError(true);
          setReady(true);
        }
      });
  }, [router]);

  function applyTheme(id: ThemeId) {
    setTheme(id);
    localStorage.setItem("zap-theme", id);
    document.documentElement.setAttribute("data-theme", id);
  }

  useEffect(() => {
    if (prevPath.current !== pathname) {
      prevPath.current = pathname;
      setDrawerOpen(false);
      setSettingsOpen(false);
      setNavigating(true);
      const t = setTimeout(() => setNavigating(false), 500);
      return () => clearTimeout(t);
    }
  }, [pathname]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") { setDrawerOpen(false); setSettingsOpen(false); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    document.body.style.overflow = drawerOpen ? "hidden" : "";
    return () => { document.body.style.overflow = ""; };
  }, [drawerOpen]);

  if (!ready) return <Splash />;

  if (authError) {
    return (
      <div style={{ minHeight: "100dvh", display: "flex", alignItems: "center", justifyContent: "center", background: "var(--bg)", flexDirection: "column", gap: "1.5rem", padding: "2rem", textAlign: "center" }}>
        <p style={{ color: "var(--danger)", fontSize: "0.9375rem" }}>Failed to load session. Please check your connection.</p>
        <button className="btn btn-primary" onClick={() => window.location.reload()}>Retry</button>
        <button onClick={async () => { await logout(); router.replace("/login"); }} className="btn btn-ghost">Sign out</button>
      </div>
    );
  }

  if (sellerStatus !== "APPROVED") {
    return (
      <div style={{ minHeight: "100dvh", display: "flex", alignItems: "center", justifyContent: "center", background: "var(--bg)", flexDirection: "column", gap: "1.5rem", padding: "2rem", textAlign: "center" }}>
        <div style={{ width: 64, height: 64, borderRadius: "50%", background: "var(--surface)", border: "2px solid var(--border)", display: "flex", alignItems: "center", justifyContent: "center" }}>
          <Clock size={28} style={{ color: "var(--accent)" }} />
        </div>
        <div>
          <h2 style={{ margin: 0, fontSize: "1.25rem", fontWeight: 700, color: "var(--text)", fontFamily: '"Rubik", "Outfit", system-ui, sans-serif' }}>
            {sellerStatus === "SUSPENDED" ? "Account Suspended" : "Approval Pending"}
          </h2>
          <p style={{ margin: "0.5rem 0 0", color: "var(--muted)", fontSize: "0.9rem", maxWidth: 360 }}>
            {sellerStatus === "SUSPENDED"
              ? "Your seller account has been suspended. Please contact support."
              : "Your seller account is under review. An admin will approve your application shortly."}
          </p>
        </div>
        <button onClick={async () => { await logout(); router.replace("/login"); }} className="btn btn-ghost">
          Sign out
        </button>
      </div>
    );
  }

  /* User avatar initials */
  const initials = userEmail ? userEmail.slice(0, 2).toUpperCase() : "ZM";

  const sidebarContent = (
    <>
      {/* Brand */}
      <div style={{ padding: "1.25rem", borderBottom: "1px solid var(--border)" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.625rem" }}>
            <div style={{
              width: 34, height: 34, borderRadius: 10,
              background: "var(--accent)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 13, fontWeight: 800, color: "var(--accent-text)",
              letterSpacing: "-0.04em", flexShrink: 0,
              boxShadow: "0 2px 8px color-mix(in srgb, var(--accent) 35%, transparent)",
            }}>ZM</div>
            <div>
              <div style={{ fontWeight: 700, fontSize: "0.9375rem", color: "var(--text)", letterSpacing: "-0.02em", lineHeight: 1.2, fontFamily: '"Rubik", "Outfit", system-ui, sans-serif' }}>
                ZapMarket
              </div>
              <div style={{ fontSize: "0.625rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.1em", marginTop: 1 }}>
                Seller Portal
              </div>
            </div>
          </div>
          <button
            className="sidebar-close-btn"
            onClick={() => setDrawerOpen(false)}
            aria-label="Close menu"
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--muted)", padding: 4, display: "none" }}
          >
            <X size={20} />
          </button>
        </div>
      </div>

      {/* Nav */}
      <nav style={{ flex: 1, padding: "0.875rem 0.625rem" }}>
        <div style={{ fontSize: "0.625rem", fontWeight: 600, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.1em", padding: "0 0.875rem", marginBottom: "0.375rem" }}>
          Navigation
        </div>
        {NAV.map(({ href, label, Icon }) => {
          const active = pathname === href || (href !== "/dashboard" && pathname.startsWith(href));
          return (
            <Link
              key={href}
              href={href}
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.75rem",
                padding: "0.625rem 0.875rem",
                borderRadius: 10,
                marginBottom: 2,
                fontSize: "0.875rem",
                fontWeight: active ? 700 : 500,
                color: active ? "var(--accent)" : "var(--text-2)",
                background: active ? "var(--accent-bg)" : "transparent",
                textDecoration: "none",
                transition: "background 0.15s, color 0.15s",
              }}
              onMouseEnter={(e) => { if (!active) { e.currentTarget.style.background = "var(--surface2)"; e.currentTarget.style.color = "var(--text)"; } }}
              onMouseLeave={(e) => { if (!active) { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--text-2)"; } }}
            >
              <div style={{
                width: 28, height: 28,
                borderRadius: 7,
                background: active ? "color-mix(in srgb, var(--accent) 18%, transparent)" : "var(--surface2)",
                display: "flex", alignItems: "center", justifyContent: "center",
                flexShrink: 0, transition: "background 0.15s",
              }}>
                <Icon size={14} strokeWidth={active ? 2.5 : 1.8} />
              </div>
              {label}
              {active && (
                <div style={{ marginLeft: "auto", width: 6, height: 6, borderRadius: "50%", background: "var(--accent)", flexShrink: 0 }} />
              )}
            </Link>
          );
        })}
      </nav>

      {/* Footer: user info + settings */}
      <div style={{ borderTop: "1px solid var(--border)", padding: "0.875rem 1rem" }}>
        {/* Settings toggle */}
        <button
          onClick={() => setSettingsOpen((p) => !p)}
          style={{
            width: "100%", display: "flex", alignItems: "center", gap: "0.625rem",
            background: "var(--surface2)", border: "1px solid var(--border)",
            borderRadius: 10, padding: "0.5rem 0.75rem", cursor: "pointer",
            marginBottom: "0.625rem", transition: "background 0.15s",
          }}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--surface3)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "var(--surface2)"; }}
          aria-expanded={settingsOpen}
        >
          <Settings size={14} style={{ color: "var(--muted)", flexShrink: 0 }} />
          <span style={{ flex: 1, textAlign: "left", fontSize: "0.75rem", color: "var(--text-2)", fontWeight: 500 }}>Settings</span>
          <ChevronDown
            size={13}
            style={{ color: "var(--muted)", transition: "transform 0.2s", transform: settingsOpen ? "rotate(180deg)" : "rotate(0deg)" }}
          />
        </button>

        {settingsOpen && (
          <div style={{ marginBottom: "0.625rem", display: "flex", flexDirection: "column", gap: "0.5rem", animation: "slide-down 0.15s ease" }}>
            {/* Clock */}
            <div style={{
              fontFamily: '"DM Mono", monospace', fontSize: "0.6875rem",
              color: "var(--muted)", letterSpacing: "0.04em",
              padding: "0.25rem 0.25rem 0",
            }}>
              {clock}
            </div>

            {/* Theme */}
            <div>
              <label htmlFor="zap-theme-select" style={{ display: "block", fontSize: "0.625rem", fontWeight: 600, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: "0.25rem" }}>
                Theme
              </label>
              <div style={{ position: "relative" }}>
                <div style={{ position: "absolute", left: "0.5rem", top: "50%", transform: "translateY(-50%)", width: 8, height: 8, borderRadius: "50%", background: THEMES.find((t) => t.id === theme)?.color ?? "var(--accent)", pointerEvents: "none" }} />
                <select
                  id="zap-theme-select"
                  className="input"
                  value={theme}
                  onChange={(e) => applyTheme(e.target.value as ThemeId)}
                  style={{ paddingLeft: "1.5rem", fontSize: "0.75rem", height: 30, paddingTop: 0, paddingBottom: 0, cursor: "pointer" }}
                  aria-label="Select theme"
                >
                  {THEMES.map((t) => <option key={t.id} value={t.id}>{t.label}</option>)}
                </select>
              </div>
            </div>

            {/* Currency */}
            <div>
              <label style={{ display: "block", fontSize: "0.625rem", fontWeight: 600, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: "0.25rem" }}>
                Currency
              </label>
              <div style={{ position: "relative" }}>
                <div style={{ position: "absolute", left: "0.5rem", top: "50%", transform: "translateY(-50%)", fontSize: "0.75rem", pointerEvents: "none", lineHeight: 1 }}>
                  {currencies.find((c) => c.code === currency)?.flag ?? "🌐"}
                </div>
                <select
                  className="input"
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value)}
                  style={{ paddingLeft: "1.75rem", fontSize: "0.75rem", height: 30, paddingTop: 0, paddingBottom: 0, cursor: "pointer" }}
                  aria-label="Select display currency"
                >
                  {currencies.map((c) => <option key={c.code} value={c.code}>{c.code} — {c.name}</option>)}
                </select>
              </div>
              <div style={{ fontSize: "0.6rem", color: stale ? "var(--warning, #d97706)" : "var(--muted)", marginTop: "0.2rem", fontFamily: '"DM Mono", monospace' }}>
                {ratesLoading ? "Fetching rates…" : stale ? `Stale · ${ratesDate}` : `Live · ${ratesDate}`}
              </div>
            </div>
          </div>
        )}

        {/* User row */}
        <div style={{ display: "flex", alignItems: "center", gap: "0.625rem" }}>
          <div style={{
            width: 32, height: 32, borderRadius: 8,
            background: "var(--accent-bg)",
            border: "1.5px solid color-mix(in srgb, var(--accent) 30%, transparent)",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: "0.6875rem", fontWeight: 700, color: "var(--accent)",
            letterSpacing: "-0.02em", flexShrink: 0,
          }}>
            {initials}
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: "0.75rem", fontWeight: 600, color: "var(--text)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {userEmail ?? "Seller"}
            </div>
            <div style={{ fontSize: "0.625rem", color: "var(--success)", fontWeight: 500, letterSpacing: "0.03em", textTransform: "uppercase" }}>
              Active
            </div>
          </div>
          <button
            onClick={async () => { await logout(); router.push("/login"); }}
            title="Sign out"
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--muted)", padding: 4, display: "flex", borderRadius: 6, transition: "color 0.15s" }}
            onMouseEnter={(e) => { e.currentTarget.style.color = "var(--danger)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.color = "var(--muted)"; }}
            aria-label="Sign out"
          >
            <LogOut size={14} strokeWidth={1.8} />
          </button>
        </div>
      </div>
    </>
  );

  return (
    <div style={{ display: "flex", minHeight: "100dvh" }}>
      {navigating && <div className="nav-progress" key={pathname} />}

      {/* Desktop sidebar */}
      <aside
        className="dash-sidebar"
        style={{
          width: "15.5rem", flexShrink: 0,
          display: "flex", flexDirection: "column",
          background: "var(--surface)",
          borderRight: "1px solid var(--border)",
          boxShadow: "2px 0 16px rgba(0,0,0,0.06)",
          position: "sticky", top: 0,
          height: "100dvh", overflowY: "auto", zIndex: 10,
        }}
      >
        {sidebarContent}
      </aside>

      {/* Mobile drawer overlay */}
      {drawerOpen && (
        <div
          className="dash-overlay"
          onClick={() => setDrawerOpen(false)}
          style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.45)", backdropFilter: "blur(3px)", zIndex: 40 }}
          aria-hidden="true"
        />
      )}

      {/* Mobile drawer */}
      <aside
        className="dash-drawer"
        style={{
          position: "fixed", top: 0, left: 0,
          height: "100dvh", width: "16rem",
          display: "flex", flexDirection: "column",
          background: "var(--surface)",
          borderRight: "1px solid var(--border)",
          boxShadow: "4px 0 32px rgba(0,0,0,0.18)",
          zIndex: 50, overflowY: "auto",
          transform: drawerOpen ? "translateX(0)" : "translateX(-100%)",
          transition: "transform 0.25s cubic-bezier(0.4,0,0.2,1)",
        }}
      >
        {sidebarContent}
      </aside>

      {/* Main content */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
        {/* Mobile topbar */}
        <header
          className="dash-topbar"
          style={{
            display: "none", alignItems: "center", justifyContent: "space-between",
            padding: "0.75rem 1rem",
            background: "var(--surface)",
            borderBottom: "1px solid var(--border)",
            position: "sticky", top: 0, zIndex: 20,
          }}
        >
          <button
            onClick={() => setDrawerOpen(true)}
            aria-label="Open menu"
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--text)", padding: 4, display: "flex", alignItems: "center" }}
          >
            <Menu size={22} />
          </button>
          <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <div style={{ width: 28, height: 28, borderRadius: 7, background: "var(--accent)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 11, fontWeight: 800, color: "var(--accent-text)", letterSpacing: "-0.04em" }}>ZM</div>
            <span style={{ fontWeight: 700, fontSize: "0.9375rem", color: "var(--text)", letterSpacing: "-0.02em", fontFamily: '"Rubik", "Outfit", system-ui, sans-serif' }}>ZapMarket</span>
          </div>
          <div style={{ width: 30 }} />
        </header>

        <main style={{ flex: 1, padding: "clamp(1rem, 4vw, 2.5rem)", overflowY: "auto", background: "var(--bg)" }}>
          {children}
        </main>

        {/* Mobile bottom nav */}
        <nav
          className="dash-bottom-nav"
          style={{
            display: "none", position: "fixed", bottom: 0, left: 0, right: 0,
            height: "4rem",
            background: "var(--surface)",
            borderTop: "1px solid var(--border)",
            boxShadow: "0 -4px 16px rgba(0,0,0,0.08)",
            zIndex: 30, paddingBottom: "env(safe-area-inset-bottom)",
          }}
        >
          <div style={{ display: "flex", height: "100%", alignItems: "center" }}>
            {NAV.map(({ href, label, Icon }) => {
              const active = pathname === href || (href !== "/dashboard" && pathname.startsWith(href));
              return (
                <Link
                  key={href}
                  href={href}
                  style={{
                    flex: 1, display: "flex", flexDirection: "column", alignItems: "center",
                    justifyContent: "center", gap: "0.2rem", textDecoration: "none",
                    color: active ? "var(--accent)" : "var(--muted)",
                    fontSize: "0.625rem", fontWeight: active ? 700 : 500,
                    letterSpacing: "0.02em", paddingTop: "0.25rem", transition: "color 0.15s",
                  }}
                >
                  <Icon size={20} strokeWidth={active ? 2.5 : 1.8} />
                  {label}
                </Link>
              );
            })}
          </div>
        </nav>
      </div>

      <style>{`
        @media (max-width: 767px) {
          .dash-sidebar    { display: none !important; }
          .dash-topbar     { display: flex !important; }
          .dash-bottom-nav { display: block !important; }
          .sidebar-close-btn { display: flex !important; }
          main { padding-bottom: 5rem !important; }
        }
        @media (min-width: 768px) and (max-width: 1023px) {
          .dash-sidebar { width: 13rem !important; }
          .dash-drawer  { display: none !important; }
          .dash-overlay { display: none !important; }
        }
        @media (min-width: 1024px) {
          .dash-drawer  { display: none !important; }
          .dash-overlay { display: none !important; }
        }
      `}</style>
    </div>
  );
}
