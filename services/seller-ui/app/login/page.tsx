"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { login, getMe } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [error, setError]       = useState("");
  const [loading, setLoading]   = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem("zap-theme") ?? "vibrant";
    document.documentElement.setAttribute("data-theme", saved);
    // Redirect if already authenticated
    getMe().then(() => router.replace("/dashboard")).catch(() => {/* not logged in */});
  }, [router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(email, password);
      router.replace("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
      setLoading(false);
    }
  }

  return (
    <div style={{ minHeight: "100dvh", display: "flex" }}>

      {/* Left panel — rose brand panel */}
      <div style={{
        display: "none",
        flex: "0 0 44%",
        background: "var(--accent)",
        padding: "3rem",
        flexDirection: "column",
        justifyContent: "space-between",
        position: "relative",
        overflow: "hidden",
      }} className="login-panel">
        <div style={{ position: "absolute", top: "-6rem", right: "-6rem", width: "22rem", height: "22rem", borderRadius: "50%", background: "rgba(255,255,255,0.08)", pointerEvents: "none" }} />
        <div style={{ position: "absolute", bottom: "-4rem", left: "-4rem", width: "16rem", height: "16rem", borderRadius: "50%", background: "rgba(255,255,255,0.06)", pointerEvents: "none" }} />

        <div style={{ display: "flex", alignItems: "center", gap: "0.625rem" }}>
          <span style={{ width: 36, height: 36, borderRadius: 9, background: "rgba(255,255,255,0.2)", backdropFilter: "blur(8px)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 14, fontWeight: 800, color: "#fff", letterSpacing: "-0.04em", border: "1px solid rgba(255,255,255,0.3)" }}>ZM</span>
          <span style={{ fontSize: "1.125rem", fontWeight: 700, color: "#fff", letterSpacing: "-0.02em" }}>ZapMarket</span>
        </div>

        <div>
          <div style={{ fontSize: "2.25rem", fontWeight: 800, color: "#fff", lineHeight: 1.1, letterSpacing: "-0.03em", marginBottom: "1rem", textWrap: "balance" }}>
            Your seller dashboard, built for growth.
          </div>
          <p style={{ fontSize: "1rem", color: "rgba(255,255,255,0.75)", lineHeight: 1.6, margin: 0 }}>
            Manage products, track orders, and grow your business from one place.
          </p>
        </div>

        <div style={{ display: "flex", gap: "2rem" }}>
          {[["10k+", "Active sellers"], ["99.4%", "Uptime"], ["24h", "Support"]].map(([num, label]) => (
            <div key={label}>
              <div style={{ fontSize: "1.5rem", fontWeight: 800, color: "#fff", fontVariantNumeric: "tabular-nums" }}>{num}</div>
              <div style={{ fontSize: "0.75rem", color: "rgba(255,255,255,0.6)", marginTop: 2 }}>{label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Right panel — form */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", background: "var(--bg)", padding: "2rem 1.5rem" }}>
        <div className="login-mobile-logo" style={{ marginBottom: "2.5rem", textAlign: "center" }}>
          <div style={{ display: "inline-flex", alignItems: "center", gap: "0.625rem", marginBottom: "0.5rem" }}>
            <span style={{ width: 40, height: 40, borderRadius: 11, background: "var(--accent)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 16, fontWeight: 800, color: "var(--accent-text)", letterSpacing: "-0.04em", boxShadow: "0 4px 16px color-mix(in srgb, var(--accent) 35%, transparent)" }}>ZM</span>
            <span style={{ fontSize: "1.5rem", fontWeight: 700, color: "var(--text)", letterSpacing: "-0.025em" }}>ZapMarket</span>
          </div>
        </div>

        <div style={{ width: "min(400px, 100%)" }}>
          <div style={{ marginBottom: "2rem" }}>
            <h1 style={{ margin: 0, fontSize: "1.75rem", fontWeight: 800, color: "var(--text)", letterSpacing: "-0.03em", lineHeight: 1.1 }}>Welcome back</h1>
            <p style={{ margin: "0.375rem 0 0", fontSize: "0.9375rem", color: "var(--muted)" }}>Sign in to your seller account</p>
          </div>

          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
            {error && (
              <p role="alert" style={{ margin: 0, background: "color-mix(in srgb, var(--danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)", borderRadius: 9, padding: "0.625rem 0.875rem", fontSize: "0.8125rem", color: "var(--danger)" }}>
                {error}
              </p>
            )}
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label" htmlFor="login-email">Email address</label>
              <input id="login-email" className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="seller@example.com" required autoComplete="email" autoFocus style={{ height: 44 }} />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label" htmlFor="login-password">Password</label>
              <input id="login-password" className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" style={{ height: 44 }} />
            </div>
            <button className="btn btn-primary" type="submit" disabled={loading} style={{ width: "100%", justifyContent: "center", height: 46, fontSize: "1rem", fontWeight: 700, marginTop: "0.375rem", borderRadius: 11 }}>
              {loading ? <><span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }} />Signing in…</> : "Sign in →"}
            </button>
            <p style={{ margin: 0, textAlign: "center", fontSize: "0.875rem", color: "var(--muted)" }}>
              Don&apos;t have an account?{" "}
              <Link href="/register" style={{ color: "var(--accent)", textDecoration: "none", fontWeight: 700 }}>Create one</Link>
            </p>
          </form>
        </div>
      </div>

      <style>{`
        @media (min-width: 768px) {
          .login-panel { display: flex !important; }
          .login-mobile-logo { display: none !important; }
        }
      `}</style>
    </div>
  );
}
