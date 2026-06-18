"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { login } from "@/lib/api";
import { saveToken, getToken } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [time, setTime] = useState("");

  useEffect(() => {
    if (getToken()) { router.replace("/dashboard"); return; }
    const saved = localStorage.getItem("zap-theme") ?? "walnut";
    document.documentElement.setAttribute("data-theme", saved);
    function tick() { setTime(new Date().toLocaleTimeString("en-GB", { hour12: false })); }
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
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
    <div style={{
      minHeight: "100vh",
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      justifyContent: "center",
      background: "var(--bg)",
      padding: "1rem",
    }}>

      {/* Top bar */}
      <div style={{
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        padding: "0.625rem 1.5rem",
        borderBottom: "1px solid var(--border)",
        display: "flex",
        alignItems: "center",
        gap: "0.75rem",
        fontFamily: '"Roboto", system-ui, sans-serif',
        fontSize: "0.6875rem",
        color: "var(--muted)",
        background: "var(--surface)",
      }}>
        <span style={{ fontWeight: 600, color: "var(--text)" }}>
          Zap<span style={{ color: "var(--accent)" }}>Market</span>
        </span>
        <span style={{ color: "var(--border)" }}>·</span>
        <span>Admin Gateway</span>
        <span style={{ marginLeft: "auto", fontFamily: '"Roboto Mono", monospace' }}>{time}</span>
      </div>

      <div style={{ width: "100%", maxWidth: "22rem" }}>

        {/* Wordmark */}
        <div style={{ marginBottom: "2rem" }}>
          <div style={{
            fontFamily: '"Roboto", system-ui, sans-serif',
            fontWeight: 700,
            fontSize: "1.5rem",
            color: "var(--text)",
            letterSpacing: "-0.02em",
            marginBottom: "0.375rem",
          }}>
            Zap<span style={{ color: "var(--accent)" }}>Market</span>
          </div>
          <div style={{
            fontFamily: '"Roboto", system-ui, sans-serif',
            fontSize: "0.8125rem",
            color: "var(--muted)",
          }}>
            Sign in to the admin console
          </div>
        </div>

        {/* Form card */}
        <div className="card" style={{ padding: "1.5rem" }}>
          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>

            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Email address</label>
              <input
                className="input"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@zapmarket.com"
                autoComplete="email"
                required
              />
            </div>

            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Password</label>
              <input
                className="input"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                autoComplete="current-password"
                required
              />
            </div>

            {error && (
              <div style={{
                padding: "0.5rem 0.75rem",
                borderRadius: "7px",
                background: "color-mix(in srgb, var(--danger) 8%, transparent)",
                border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)",
                fontSize: "0.8125rem",
                color: "var(--danger)",
              }}>
                {error}
              </div>
            )}

            <button
              className="btn btn-primary"
              type="submit"
              disabled={loading}
              style={{ width: "100%", marginTop: "0.25rem" }}
            >
              {loading ? "Signing in…" : "Sign in →"}
            </button>
          </form>
        </div>

        <div style={{
          marginTop: "1rem",
          fontSize: "0.6875rem",
          color: "var(--muted)",
          textAlign: "center",
        }}>
          ZapMarket API Gateway · v1.0
        </div>
      </div>
    </div>
  );
}
