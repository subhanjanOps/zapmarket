"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { login } from "@/lib/api";
import { saveToken } from "@/lib/auth";

const HIGHLIGHTS = [
  { label: "Products", value: "12,481" },
  { label: "Active Orders", value: "3,204" },
  { label: "Sellers", value: "847" },
  { label: "Users", value: "98,612" },
];

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [error, setError]       = useState("");
  const [loading, setLoading]   = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem("bo_theme");
    document.documentElement.setAttribute("data-theme", saved ?? "enterprise-dark");
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const token = await login(email, password);
      saveToken(token);
      router.push("/dashboard");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ minHeight: "100dvh", display: "flex", background: "var(--bg)" }}>

      {/* ── Left accent panel ─────────────────────────────────────────────── */}
      <div className="login-accent-panel" style={{
        display: "none",
        flex: "0 0 42%",
        flexDirection: "column",
        justifyContent: "space-between",
        padding: "3rem",
        background: "linear-gradient(145deg, var(--accent) 0%, color-mix(in srgb, var(--accent) 60%, #000) 100%)",
        position: "relative",
        overflow: "hidden",
      }}>
        {/* Background grid texture */}
        <div style={{
          position: "absolute",
          inset: 0,
          backgroundImage: `
            linear-gradient(rgba(255,255,255,0.04) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px)
          `,
          backgroundSize: "40px 40px",
          pointerEvents: "none",
        }} />

        {/* Glow */}
        <div style={{
          position: "absolute",
          top: "-20%",
          right: "-20%",
          width: "60%",
          height: "60%",
          borderRadius: "50%",
          background: "radial-gradient(circle, rgba(255,255,255,0.15) 0%, transparent 70%)",
          pointerEvents: "none",
        }} />

        {/* Brand */}
        <div style={{ position: "relative", zIndex: 1 }}>
          <div style={{
            display: "flex",
            alignItems: "center",
            gap: "0.75rem",
            marginBottom: "0.5rem",
          }}>
            <div style={{
              width: 32, height: 32, borderRadius: 8,
              background: "rgba(255,255,255,0.2)",
              backdropFilter: "blur(4px)",
              border: "1px solid rgba(255,255,255,0.25)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 13, fontWeight: 800, color: "#fff",
              letterSpacing: "-0.04em",
            }}>
              ZM
            </div>
            <div>
              <div style={{ fontSize: "1rem", fontWeight: 700, color: "#fff", letterSpacing: "-0.02em", lineHeight: 1.2 }}>
                ZapMarket
              </div>
              <div style={{ fontSize: "0.5625rem", color: "rgba(255,255,255,0.6)", textTransform: "uppercase", letterSpacing: "0.1em" }}>
                Admin Console
              </div>
            </div>
          </div>
        </div>

        {/* Headline */}
        <div style={{ position: "relative", zIndex: 1 }}>
          <h2 style={{
            fontSize: "clamp(1.375rem, 3vw, 2rem)",
            fontWeight: 700,
            color: "#fff",
            margin: "0 0 1rem",
            letterSpacing: "-0.03em",
            lineHeight: 1.2,
          }}>
            Manage your marketplace with confidence.
          </h2>
          <p style={{
            fontSize: "0.875rem",
            color: "rgba(255,255,255,0.7)",
            margin: "0 0 2.5rem",
            lineHeight: 1.6,
          }}>
            Full visibility into products, orders, users, and compliance from a single console.
          </p>

          {/* Stat grid */}
          <div style={{
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: "1rem",
          }}>
            {HIGHLIGHTS.map(({ label, value }) => (
              <div key={label} style={{
                background: "rgba(255,255,255,0.1)",
                backdropFilter: "blur(8px)",
                border: "1px solid rgba(255,255,255,0.15)",
                borderRadius: "10px",
                padding: "0.875rem 1rem",
              }}>
                <div style={{
                  fontSize: "1.375rem",
                  fontWeight: 700,
                  color: "#fff",
                  letterSpacing: "-0.03em",
                  lineHeight: 1,
                  marginBottom: "0.25rem",
                  fontFeatureSettings: '"tnum"',
                }}>
                  {value}
                </div>
                <div style={{
                  fontSize: "0.625rem",
                  color: "rgba(255,255,255,0.6)",
                  textTransform: "uppercase",
                  letterSpacing: "0.08em",
                  fontWeight: 500,
                }}>
                  {label}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div style={{ position: "relative", zIndex: 1, fontSize: "0.6875rem", color: "rgba(255,255,255,0.45)" }}>
          ZapMarket Admin Console · Restricted Access
        </div>
      </div>

      {/* ── Right: form ───────────────────────────────────────────────────── */}
      <div style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: "clamp(1.5rem, 5vw, 3rem)",
        overflowY: "auto",
      }}>

        {/* Mobile wordmark */}
        <div className="login-mobile-brand" style={{
          display: "none",
          textAlign: "center",
          marginBottom: "2rem",
        }}>
          <div style={{
            display: "inline-flex", alignItems: "center", gap: "0.625rem", marginBottom: "0.5rem",
          }}>
            <span style={{
              width: 32, height: 32, borderRadius: 8,
              background: "var(--accent)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: 12, fontWeight: 800, color: "var(--accent-text)", letterSpacing: "-0.04em",
            }}>ZM</span>
            <span style={{ fontSize: "1.125rem", fontWeight: 700, color: "var(--text)", letterSpacing: "-0.025em" }}>
              ZapMarket
            </span>
          </div>
          <p style={{ color: "var(--muted)", fontSize: "0.75rem", margin: 0 }}>
            Admin Console
          </p>
        </div>

        <div style={{ width: "100%", maxWidth: 400 }}>
          <div style={{ marginBottom: "2rem" }}>
            <h1 style={{
              fontSize: "1.375rem",
              fontWeight: 700,
              color: "var(--text)",
              margin: "0 0 0.375rem",
              letterSpacing: "-0.025em",
            }}>
              Sign in
            </h1>
            <p style={{ color: "var(--muted)", fontSize: "0.8125rem", margin: 0, letterSpacing: "0.01em" }}>
              Access restricted to admin accounts.
            </p>
          </div>

          <div className="card" style={{ padding: "2rem", border: "1px solid var(--border-strong)" }}>
            <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>

              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label" htmlFor="login-email">Email address</label>
                <input
                  id="login-email"
                  className="input"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@zapmarket.io"
                  autoComplete="email"
                  required
                  autoFocus
                  style={{ height: 40 }}
                />
              </div>

              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label" htmlFor="login-password">Password</label>
                <input
                  id="login-password"
                  className="input"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  autoComplete="current-password"
                  required
                  style={{ height: 40 }}
                />
              </div>

              {error && (
                <p role="alert" style={{
                  margin: 0, fontSize: "0.8125rem",
                  color: "var(--danger)",
                  background: "color-mix(in srgb, var(--danger) 10%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)",
                  borderRadius: 6, padding: "0.5625rem 0.875rem",
                }}>
                  {error}
                </p>
              )}

              <button
                className="btn btn-primary"
                type="submit"
                disabled={loading}
                style={{ width: "100%", justifyContent: "center", height: 40, fontSize: "0.9375rem", fontWeight: 600, marginTop: "0.25rem" }}
              >
                {loading ? (
                  <><span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }} />Signing in…</>
                ) : "Sign in →"}
              </button>
            </form>
          </div>
        </div>
      </div>

      <style>{`
        @media (min-width: 768px) {
          .login-accent-panel { display: flex !important; }
        }
        @media (max-width: 767px) {
          .login-mobile-brand { display: block !important; }
        }
      `}</style>
    </div>
  );
}
