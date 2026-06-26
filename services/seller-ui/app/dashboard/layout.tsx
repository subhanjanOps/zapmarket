"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { Clock } from "lucide-react";
import { Splash } from "@/app/components/Skeleton";
import { getMe } from "@/lib/api";
import { logout } from "@/lib/auth";
import { useCurrency } from "@/lib/currency";
import { SidebarContent } from "./components/Sidebar";
import { MobileTopBar } from "./components/MobileTopBar";
import { MobileBottomNav } from "./components/MobileBottomNav";

type ThemeId = "vibrant" | "night" | "walnut" | "cream" | "slate" | "solarized-dark" | "solarized-light";

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
  const [ready, setReady]                   = useState(false);
  const [theme, setTheme]                   = useState<ThemeId>("vibrant");
  const [navigating, setNavigating]         = useState(false);
  const [sellerStatus, setSellerStatus]     = useState<string | null>(null);
  const [authError, setAuthError]           = useState(false);
  const [drawerOpen, setDrawerOpen]         = useState(false);
  const [settingsOpen, setSettingsOpen]     = useState(false);
  const [userEmail, setUserEmail]           = useState<string | null>(null);
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
        <button onClick={async () => { await logout(); router.replace("/login"); }} className="btn btn-ghost">Sign out</button>
      </div>
    );
  }

  const sidebarProps = {
    pathname, theme, applyTheme, settingsOpen, setSettingsOpen,
    clock, currency, setCurrency, currencies, ratesLoading, ratesDate, stale,
    userEmail, onClose: () => setDrawerOpen(false),
    onLogout: async () => { await logout(); router.push("/login"); },
  };

  return (
    <div style={{ display: "flex", minHeight: "100dvh" }}>
      {navigating && <div className="nav-progress" key={pathname} />}

      {/* Desktop sidebar */}
      <aside className="dash-sidebar" style={{ width: "15.5rem", flexShrink: 0, display: "flex", flexDirection: "column", background: "var(--surface)", borderRight: "1px solid var(--border)", boxShadow: "2px 0 16px rgba(0,0,0,0.06)", position: "sticky", top: 0, height: "100dvh", overflowY: "auto", zIndex: 10 }}>
        <SidebarContent {...sidebarProps} />
      </aside>

      {/* Mobile drawer overlay */}
      {drawerOpen && (
        <div className="dash-overlay" onClick={() => setDrawerOpen(false)} style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.45)", backdropFilter: "blur(3px)", zIndex: 40 }} aria-hidden="true" />
      )}

      {/* Mobile drawer */}
      <aside className="dash-drawer" style={{ position: "fixed", top: 0, left: 0, height: "100dvh", width: "16rem", display: "flex", flexDirection: "column", background: "var(--surface)", borderRight: "1px solid var(--border)", boxShadow: "4px 0 32px rgba(0,0,0,0.18)", zIndex: 50, overflowY: "auto", transform: drawerOpen ? "translateX(0)" : "translateX(-100%)", transition: "transform 0.25s cubic-bezier(0.4,0,0.2,1)" }}>
        <SidebarContent {...sidebarProps} />
      </aside>

      {/* Main content */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
        <MobileTopBar onMenuOpen={() => setDrawerOpen(true)} />
        <main style={{ flex: 1, padding: "clamp(1rem, 4vw, 2.5rem)", overflowY: "auto", background: "var(--bg)" }}>
          {children}
        </main>
        <MobileBottomNav pathname={pathname} />
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
