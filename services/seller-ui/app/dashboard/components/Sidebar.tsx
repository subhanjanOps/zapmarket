"use client";
import Link from "next/link";
import { LayoutDashboard, Package, ShoppingBag, LogOut, X, Settings, ChevronDown } from "lucide-react";
import type { CurrencyMeta } from "@/lib/currency";

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

interface Props {
  pathname: string;
  theme: ThemeId;
  applyTheme: (id: ThemeId) => void;
  settingsOpen: boolean;
  setSettingsOpen: React.Dispatch<React.SetStateAction<boolean>>;
  clock: string;
  currency: string;
  setCurrency: (c: string) => void;
  currencies: CurrencyMeta[];
  ratesLoading: boolean;
  ratesDate: string;
  stale: boolean;
  userEmail: string | null;
  onClose: () => void;
  onLogout: () => void;
}

export function SidebarContent({
  pathname, theme, applyTheme, settingsOpen, setSettingsOpen,
  clock, currency, setCurrency, currencies, ratesLoading, ratesDate, stale,
  userEmail, onClose, onLogout,
}: Props) {
  const initials = userEmail ? userEmail.slice(0, 2).toUpperCase() : "ZM";

  return (
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
            onClick={onClose}
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
                display: "flex", alignItems: "center", gap: "0.75rem",
                padding: "0.625rem 0.875rem", borderRadius: 10, marginBottom: 2,
                fontSize: "0.875rem", fontWeight: active ? 700 : 500,
                color: active ? "var(--accent)" : "var(--text-2)",
                background: active ? "var(--accent-bg)" : "transparent",
                textDecoration: "none", transition: "background 0.15s, color 0.15s",
              }}
              onMouseEnter={(e) => { if (!active) { e.currentTarget.style.background = "var(--surface2)"; e.currentTarget.style.color = "var(--text)"; } }}
              onMouseLeave={(e) => { if (!active) { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--text-2)"; } }}
            >
              <div style={{
                width: 28, height: 28, borderRadius: 7,
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

      {/* Footer */}
      <div style={{ borderTop: "1px solid var(--border)", padding: "0.875rem 1rem" }}>
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
          <ChevronDown size={13} style={{ color: "var(--muted)", transition: "transform 0.2s", transform: settingsOpen ? "rotate(180deg)" : "rotate(0deg)" }} />
        </button>

        {settingsOpen && (
          <div style={{ marginBottom: "0.625rem", display: "flex", flexDirection: "column", gap: "0.5rem", animation: "slide-down 0.15s ease" }}>
            <div style={{ fontFamily: '"DM Mono", monospace', fontSize: "0.6875rem", color: "var(--muted)", letterSpacing: "0.04em", padding: "0.25rem 0.25rem 0" }}>
              {clock}
            </div>
            <div>
              <label htmlFor="zap-theme-select" style={{ display: "block", fontSize: "0.625rem", fontWeight: 600, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: "0.25rem" }}>
                Theme
              </label>
              <div style={{ position: "relative" }}>
                <div style={{ position: "absolute", left: "0.5rem", top: "50%", transform: "translateY(-50%)", width: 8, height: 8, borderRadius: "50%", background: THEMES.find((t) => t.id === theme)?.color ?? "var(--accent)", pointerEvents: "none" }} />
                <select id="zap-theme-select" className="input" value={theme} onChange={(e) => applyTheme(e.target.value as ThemeId)} style={{ paddingLeft: "1.5rem", fontSize: "0.75rem", height: 30, paddingTop: 0, paddingBottom: 0, cursor: "pointer" }} aria-label="Select theme">
                  {THEMES.map((t) => <option key={t.id} value={t.id}>{t.label}</option>)}
                </select>
              </div>
            </div>
            <div>
              <label style={{ display: "block", fontSize: "0.625rem", fontWeight: 600, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: "0.25rem" }}>
                Currency
              </label>
              <div style={{ position: "relative" }}>
                <div style={{ position: "absolute", left: "0.5rem", top: "50%", transform: "translateY(-50%)", fontSize: "0.75rem", pointerEvents: "none", lineHeight: 1 }}>
                  {currencies.find((c) => c.code === currency)?.flag ?? "🌐"}
                </div>
                <select className="input" value={currency} onChange={(e) => setCurrency(e.target.value)} style={{ paddingLeft: "1.75rem", fontSize: "0.75rem", height: 30, paddingTop: 0, paddingBottom: 0, cursor: "pointer" }} aria-label="Select display currency">
                  {currencies.map((c) => <option key={c.code} value={c.code}>{c.code} — {c.name}</option>)}
                </select>
              </div>
              <div style={{ fontSize: "0.6rem", color: stale ? "var(--warning, #d97706)" : "var(--muted)", marginTop: "0.2rem", fontFamily: '"DM Mono", monospace' }}>
                {ratesLoading ? "Fetching rates…" : stale ? `Stale · ${ratesDate}` : `Live · ${ratesDate}`}
              </div>
            </div>
          </div>
        )}

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
            onClick={onLogout}
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
}
