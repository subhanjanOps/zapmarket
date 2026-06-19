"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { login } from "@/lib/api";
import { saveToken, getToken } from "@/lib/auth";

const BOOT_LINES = [
  "$ zapmarket gateway boot --env production",
  "> loading auth subsystem……………… [OK]",
  "> binding grpc endpoints………………… [OK]",
  "> starting route registry…………… [OK]",
  "> initialising rate limiter………… [OK]",
  "> connecting to auth-service:50051… [OK]",
  "> health checks passing ✓",
  "",
  "gateway ready · http://0.0.0.0:8000",
];

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [time, setTime] = useState("");
  const [lines, setLines] = useState<string[]>([]);

  useEffect(() => {
    if (getToken()) { router.replace("/dashboard"); return; }
    const saved = localStorage.getItem("zap-theme") ?? "terminal";
    document.documentElement.setAttribute("data-theme", saved);

    function tick() { setTime(new Date().toLocaleTimeString("en-GB", { hour12: false })); }
    tick();
    const clockId = setInterval(tick, 1000);

    let i = 0;
    const termId = setInterval(() => {
      if (i < BOOT_LINES.length) {
        const nextLine = BOOT_LINES[i];
        i++;
        setLines((prev) => [...prev, nextLine]);
      } else {
        clearInterval(termId);
      }
    }, 280);

    return () => { clearInterval(clockId); clearInterval(termId); };
  }, [router]);

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

      {/* ── Left panel: terminal ───────────────────────────────────────────── */}
      <div style={{
        flex: "0 0 45%",
        display: "none", /* hidden on mobile, shown via CSS */
        flexDirection: "column",
        justifyContent: "space-between",
        padding: "2.5rem",
        background: "var(--surface)",
        borderRight: "1px solid var(--border)",
        fontFamily: '"JetBrains Mono", monospace',
        position: "relative",
        overflow: "hidden",
      }} className="login-left-panel">

        {/* Scanline overlay */}
        <div style={{
          position: "absolute",
          inset: 0,
          backgroundImage: "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.08) 2px, rgba(0,0,0,0.08) 4px)",
          pointerEvents: "none",
        }} />

        {/* Top brand */}
        <div>
          <div style={{
            fontSize: "0.8125rem",
            fontWeight: 600,
            color: "var(--text)",
            letterSpacing: "-0.01em",
            display: "flex",
            alignItems: "center",
            gap: "0.375rem",
            marginBottom: "0.25rem",
          }}>
            <span style={{ color: "var(--accent)", fontWeight: 700 }}>{">"}_</span>
            Zap<span style={{ color: "var(--accent)" }}>Market</span>
          </div>
          <div style={{
            fontSize: "0.5625rem",
            color: "var(--muted)",
            textTransform: "uppercase",
            letterSpacing: "0.1em",
          }}>
            API Gateway · v1.0
          </div>
        </div>

        {/* Terminal output */}
        <div style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          padding: "2rem 0",
        }}>
          <div style={{
            fontSize: "0.625rem",
            color: "var(--muted)",
            textTransform: "uppercase",
            letterSpacing: "0.1em",
            marginBottom: "0.875rem",
          }}>
            boot log
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: "0.3125rem" }}>
            {lines.map((line, i) => (
              <div key={i} style={{
                fontSize: "0.75rem",
                color: line.startsWith("$")
                  ? "var(--accent)"
                  : line.startsWith(">")
                    ? "var(--text-2)"
                    : line.includes("[OK]")
                      ? "var(--success)"
                      : line.includes("ready")
                        ? "var(--text)"
                        : "var(--muted)",
                lineHeight: 1.55,
                letterSpacing: "0.01em",
                animation: "fade-in-line 0.2s ease",
              }}>
                {line || " "}
              </div>
            ))}
            {lines.length > 0 && lines.length < BOOT_LINES.length && (
              <span style={{
                display: "inline-block",
                width: "0.5rem",
                height: "0.875rem",
                background: "var(--accent)",
                animation: "blink 1s step-end infinite",
                verticalAlign: "text-bottom",
              }} />
            )}
          </div>
        </div>

        {/* Bottom clock */}
        <div style={{
          fontSize: "0.6875rem",
          color: "var(--muted)",
          letterSpacing: "0.05em",
          display: "flex",
          alignItems: "center",
          gap: "0.5rem",
        }}>
          <span className="status-dot status-dot-green status-dot-pulse" />
          <span>{time}</span>
          <span style={{ color: "var(--border)" }}>·</span>
          <span>gateway online</span>
        </div>
      </div>

      {/* ── Right panel: login form ────────────────────────────────────────── */}
      <div style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: "clamp(1.5rem, 5vw, 3rem)",
        overflowY: "auto",
      }}>

        {/* Mobile brand (only on mobile) */}
        <div className="login-mobile-brand" style={{
          display: "none",
          marginBottom: "2rem",
          textAlign: "center",
          fontFamily: '"JetBrains Mono", monospace',
        }}>
          <div style={{
            fontSize: "1rem",
            fontWeight: 600,
            color: "var(--text)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            gap: "0.375rem",
            marginBottom: "0.25rem",
          }}>
            <span style={{ color: "var(--accent)" }}>{">"}_</span>
            Zap<span style={{ color: "var(--accent)" }}>Market</span>
          </div>
          <div style={{ fontSize: "0.5625rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.1em" }}>
            API Gateway
          </div>
        </div>

        <div style={{ width: "100%", maxWidth: "22rem" }}>
          <div style={{ marginBottom: "2rem" }}>
            <h1 style={{
              fontFamily: '"Space Grotesk", system-ui, sans-serif',
              fontWeight: 700,
              fontSize: "1.375rem",
              color: "var(--text)",
              margin: "0 0 0.375rem",
              letterSpacing: "-0.025em",
            }}>
              Admin Console
            </h1>
            <p style={{
              fontFamily: '"JetBrains Mono", monospace',
              fontSize: "0.6875rem",
              color: "var(--muted)",
              margin: 0,
              letterSpacing: "0.02em",
            }}>
              authenticate to access gateway
            </p>
          </div>

          <div className="card" style={{ padding: "1.5rem", borderColor: "var(--border-strong)" }}>
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
                  autoFocus
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
                {loading ? (
                  <><span className="spinner" style={{ width: 14, height: 14, borderWidth: "2px" }} /> Authenticating…</>
                ) : "Authenticate →"}
              </button>
            </form>
          </div>

          <div style={{
            marginTop: "1.25rem",
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: "0.5625rem",
            color: "var(--muted)",
            textAlign: "center",
            letterSpacing: "0.04em",
            textTransform: "uppercase",
          }}>
            restricted access · admin only
          </div>
        </div>
      </div>

      <style>{`
        @keyframes blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }
        @keyframes fade-in-line {
          from { opacity: 0; transform: translateY(2px); }
          to   { opacity: 1; transform: translateY(0); }
        }
        @media (min-width: 768px) {
          .login-left-panel { display: flex !important; }
        }
        @media (max-width: 767px) {
          .login-mobile-brand { display: block !important; }
        }
      `}</style>
    </div>
  );
}
