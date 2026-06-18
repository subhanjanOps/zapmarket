"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import { register } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName]   = useState("");
  const [email, setEmail]         = useState("");
  const [password, setPassword]   = useState("");
  const [confirm, setConfirm]     = useState("");
  const [error, setError]         = useState("");
  const [loading, setLoading]     = useState(false);

  useEffect(() => {
    if (getToken()) { router.replace("/dashboard"); return; }
    const saved = localStorage.getItem("zap-theme") ?? "walnut";
    document.documentElement.setAttribute("data-theme", saved);
  }, [router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (password !== confirm) { setError("Passwords do not match"); return; }
    if (password.length < 8)  { setError("Password must be at least 8 characters"); return; }
    setLoading(true);
    try {
      await register(firstName, lastName, email, password);
      router.push("/login?registered=1");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
      setLoading(false);
    }
  }

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", background: "var(--bg)", padding: "1.5rem" }}>
      <div style={{ marginBottom: "1.75rem", textAlign: "center" }}>
        <div style={{ fontWeight: 700, fontSize: "1.75rem", letterSpacing: "-0.02em", marginBottom: "0.375rem" }}>
          Zap<span style={{ color: "var(--accent)" }}>Market</span>
        </div>
        <p style={{ margin: 0, fontSize: "0.875rem", color: "var(--muted)" }}>Create your seller account</p>
      </div>

      <form onSubmit={handleSubmit} style={{ width: "min(440px, 100%)", background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 10, padding: "1.75rem", boxShadow: "var(--shadow-md)", display: "flex", flexDirection: "column", gap: "1rem" }}>
        {error && (
          <div style={{ background: "color-mix(in srgb, var(--danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)", borderRadius: 7, padding: "0.625rem 0.875rem", fontSize: "0.8125rem", color: "var(--danger)" }}>
            {error}
          </div>
        )}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "0.75rem" }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">First name</label>
            <input className="input" value={firstName} onChange={(e) => setFirstName(e.target.value)} required placeholder="Jane" />
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">Last name</label>
            <input className="input" value={lastName} onChange={(e) => setLastName(e.target.value)} required placeholder="Smith" />
          </div>
        </div>
        <div className="form-group" style={{ marginBottom: 0 }}>
          <label className="form-label">Email address</label>
          <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required placeholder="seller@example.com" autoComplete="email" />
        </div>
        <div className="form-group" style={{ marginBottom: 0 }}>
          <label className="form-label">Password</label>
          <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="new-password" />
        </div>
        <div className="form-group" style={{ marginBottom: 0 }}>
          <label className="form-label">Confirm password</label>
          <input className="input" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required autoComplete="new-password" />
        </div>
        <button className="btn btn-primary" type="submit" disabled={loading} style={{ width: "100%", justifyContent: "center", marginTop: "0.25rem" }}>
          {loading ? "Creating account…" : "Create seller account"}
        </button>
        <p style={{ margin: 0, textAlign: "center", fontSize: "0.8125rem", color: "var(--muted)" }}>
          Already have an account?{" "}
          <Link href="/login" style={{ color: "var(--accent)", textDecoration: "none" }}>Sign in</Link>
        </p>
      </form>

      <p style={{ marginTop: "2rem", fontSize: "0.75rem", color: "var(--muted)" }}>ZapMarket Seller Portal · v1.0</p>
    </div>
  );
}
