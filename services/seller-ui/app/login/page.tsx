"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { login, getMe } from "@/lib/api";
import { Eye, EyeOff } from "lucide-react";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [showPass, setShowPass] = useState(false);
  const [error, setError]       = useState("");
  const [loading, setLoading]   = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem("zap-theme") ?? "vibrant";
    document.documentElement.setAttribute("data-theme", saved);
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
    <div style={{ minHeight: "100dvh", display: "flex", background: "var(--bg)" }}>

      {/* Left brand panel */}
      <div
        className="login-panel"
        style={{
          display: "none",
          flex: "0 0 46%",
          background: "var(--accent)",
          padding: "3rem",
          flexDirection: "column",
          justifyContent: "space-between",
          position: "relative",
          overflow: "hidden",
        }}
      >
        {/* Decorative circles */}
        <div style={{ position: "absolute", top: "-8rem", right: "-8rem", width: "26rem", height: "26rem", borderRadius: "50%", background: "rgba(255,255,255,0.07)", pointerEvents: "none" }} />
        <div style={{ position: "absolute", bottom: "-5rem", left: "-5rem", width: "20rem", height: "20rem", borderRadius: "50%", background: "rgba(255,255,255,0.05)", pointerEvents: "none" }} />
        <div style={{ position: "absolute", top: "40%", left: "55%", width: "10rem", height: "10rem", borderRadius: "50%", background: "rgba(255,255,255,0.04)", pointerEvents: "none" }} />

        {/* Logo */}
        <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", position: "relative" }}>
          <span style={{
            width: 40, height: 40, borderRadius: 11,
            background: "rgba(255,255,255,0.2)",
            backdropFilter: "blur(8px)",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 15, fontWeight: 800, color: "#fff",
            letterSpacing: "-0.04em",
            border: "1.5px solid rgba(255,255,255,0.3)",
          }}>ZM</span>
          <span style={{ fontSize: "1.25rem", fontWeight: 700, color: "#fff", letterSpacing: "-0.02em", fontFamily: '"Rubik", "Outfit", system-ui, sans-serif' }}>
            ZapMarket
          </span>
        </div>

        {/* Hero copy */}
        <div style={{ position: "relative" }}>
          <div style={{
            fontSize: "2.625rem",
            fontWeight: 900,
            color: "#fff",
            lineHeight: 1.1,
            letterSpacing: "-0.04em",
            marginBottom: "1.25rem",
            fontFamily: '"Rubik", "Outfit", system-ui, sans-serif',
            textWrap: "balance",
          }}>
            Your store.<br />Your rules.<br />Your growth.
          </div>
          <p style={{ fontSize: "1rem", color: "rgba(255,255,255,0.72)", lineHeight: 1.65, margin: 0, maxWidth: "30ch" }}>
            Manage products, track orders, and scale your business — all from one powerful seller dashboard.
          </p>
        </div>

        {/* Stats */}
        <div style={{ display: "flex", gap: "2.5rem", position: "relative" }}>
          {[["10k+", "Active sellers"], ["99.4%", "Uptime SLA"], ["24h", "Support"]].map(([num, label]) => (
            <div key={label}>
              <div style={{
                fontSize: "1.625rem", fontWeight: 800, color: "#fff",
                fontVariantNumeric: "tabular-nums",
                fontFamily: '"Rubik", "Outfit", system-ui, sans-serif',
                letterSpacing: "-0.02em",
              }}>{num}</div>
              <div style={{ fontSize: "0.75rem", color: "rgba(255,255,255,0.58)", marginTop: 3 }}>{label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Right: form */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: "2rem 1.5rem" }}>
        {/* Mobile-only logo */}
        <div className="login-mobile-logo" style={{ marginBottom: "2.5rem", textAlign: "center" }}>
          <div style={{ display: "inline-flex", alignItems: "center", gap: "0.625rem", marginBottom: "0.375rem" }}>
            <span style={{
              width: 40, height: 40, borderRadius: 11,
              background: "var(--accent)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 16, fontWeight: 800, color: "var(--accent-text)",
              letterSpacing: "-0.04em",
              boxShadow: "0 4px 20px color-mix(in srgb, var(--accent) 40%, transparent)",
            }}>ZM</span>
            <span style={{ fontSize: "1.5rem", fontWeight: 700, color: "var(--text)", letterSpacing: "-0.025em", fontFamily: '"Rubik", "Outfit", system-ui, sans-serif' }}>
              ZapMarket
            </span>
          </div>
        </div>

        <div style={{ width: "min(400px, 100%)" }}>
          <div style={{ marginBottom: "2rem" }}>
            <h1 style={{
              margin: 0,
              fontSize: "2rem",
              fontWeight: 800,
              color: "var(--text)",
              letterSpacing: "-0.04em",
              lineHeight: 1.1,
              fontFamily: '"Rubik", "Outfit", system-ui, sans-serif',
            }}>
              Welcome back
            </h1>
            <p style={{ margin: "0.5rem 0 0", fontSize: "0.9375rem", color: "var(--muted)", lineHeight: 1.5 }}>
              Sign in to your seller account to continue
            </p>
          </div>

          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
            {error && (
              <div
                role="alert"
                style={{
                  background: "color-mix(in srgb, var(--danger) 10%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)",
                  borderRadius: 10,
                  padding: "0.75rem 1rem",
                  fontSize: "0.8125rem",
                  color: "var(--danger)",
                  lineHeight: 1.5,
                }}
              >
                {error}
              </div>
            )}

            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label" htmlFor="login-email">Email address</label>
              <input
                id="login-email"
                className="input"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="seller@example.com"
                required
                autoComplete="email"
                autoFocus
                style={{ height: 46 }}
              />
            </div>

            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label" htmlFor="login-password">Password</label>
              <div style={{ position: "relative" }}>
                <input
                  id="login-password"
                  className="input"
                  type={showPass ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                  style={{ height: 46, paddingRight: "2.5rem" }}
                />
                <button
                  type="button"
                  onClick={() => setShowPass((p) => !p)}
                  aria-label={showPass ? "Hide password" : "Show password"}
                  style={{
                    position: "absolute", right: "0.75rem", top: "50%", transform: "translateY(-50%)",
                    background: "none", border: "none", cursor: "pointer",
                    color: "var(--muted)", padding: 2,
                    display: "flex", alignItems: "center",
                  }}
                >
                  {showPass ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </div>
            </div>

            <button
              className="btn btn-primary"
              type="submit"
              disabled={loading}
              style={{
                width: "100%", justifyContent: "center",
                height: 48,
                fontSize: "1rem",
                fontWeight: 700,
                marginTop: "0.25rem",
                borderRadius: 12,
                letterSpacing: "-0.01em",
              }}
            >
              {loading ? (
                <><span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }} /> Signing in…</>
              ) : (
                "Sign in →"
              )}
            </button>

            <p style={{ margin: 0, textAlign: "center", fontSize: "0.875rem", color: "var(--muted)" }}>
              Don&apos;t have an account?{" "}
              <Link href="/register" style={{ color: "var(--accent)", textDecoration: "none", fontWeight: 700 }}>
                Create one
              </Link>
            </p>
          </form>
        </div>
      </div>

      <style>{`
        @media (min-width: 768px) {
          .login-panel       { display: flex !important; }
          .login-mobile-logo { display: none !important; }
        }
      `}</style>
    </div>
  );
}
