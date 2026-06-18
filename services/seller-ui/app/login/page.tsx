"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { getToken, saveToken } from "@/lib/auth";
import { login } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [error, setError]       = useState("");
  const [loading, setLoading]   = useState(false);

  useEffect(() => {
    if (getToken()) { router.replace("/dashboard"); return; }
    const saved = localStorage.getItem("zap-theme") ?? "walnut";
    document.documentElement.setAttribute("data-theme", saved);
  }, [router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const { token } = await login(email, password);
      saveToken(token);
      router.replace("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
      setLoading(false);
    }
  }

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", background: "var(--bg)", padding: "1.5rem" }}>
      <div style={{ marginBottom: "1.75rem", textAlign: "center" }}>
        <div style={{ fontWeight: 700, fontSize: "1.75rem", letterSpacing: "-0.02em", marginBottom: "0.375rem" }}>
          Zap<span style={{ color: "var(--accent)" }}>Market</span>
        </div>
        <p style={{ margin: 0, fontSize: "0.875rem", color: "var(--muted)" }}>Sign in to your seller account</p>
      </div>

      <form onSubmit={handleSubmit} style={{ width: "min(400px, 100%)", background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 10, padding: "1.75rem", boxShadow: "var(--shadow-md)", display: "flex", flexDirection: "column", gap: "1rem" }}>
        {error && (
          <div style={{ background: "color-mix(in srgb, var(--danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)", borderRadius: 7, padding: "0.625rem 0.875rem", fontSize: "0.8125rem", color: "var(--danger)" }}>
            {error}
          </div>
        )}
        <div className="form-group" style={{ marginBottom: 0 }}>
          <label className="form-label">Email address</label>
          <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="seller@example.com" required autoComplete="email" />
        </div>
        <div className="form-group" style={{ marginBottom: 0 }}>
          <label className="form-label">Password</label>
          <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" />
        </div>
        <button className="btn btn-primary" type="submit" disabled={loading} style={{ width: "100%", justifyContent: "center", marginTop: "0.25rem" }}>
          {loading ? "Signing in…" : "Sign in →"}
        </button>
        <p style={{ margin: 0, textAlign: "center", fontSize: "0.8125rem", color: "var(--muted)" }}>
          Don&apos;t have an account?{" "}
          <Link href="/register" style={{ color: "var(--accent)", textDecoration: "none" }}>Register</Link>
        </p>
      </form>

      <p style={{ marginTop: "2rem", fontSize: "0.75rem", color: "var(--muted)" }}>ZapMarket Seller Portal · v1.0</p>
    </div>
  );
}
