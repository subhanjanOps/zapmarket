"use client";
import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Zap } from "lucide-react";

function LoginForm() {
  const router = useRouter();
  const sp = useSearchParams();
  const next = sp.get("next") ?? "/";
  const [form, setForm] = useState({ email: "", password: "" });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      const data = await res.json();
      if (!res.ok) { setError((data as Record<string, string>).error ?? "Login failed"); return; }
      router.push(next);
      router.refresh();
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4 py-12" style={{ background: "#FFFCF5" }}>
      <div className="w-full max-w-sm rounded-2xl overflow-hidden" style={{ border: "1px solid #F0EDE8", background: "#fff" }}>
        <div className="h-1.5 w-full" style={{ background: "#FF2D78" }} />
        <div className="p-8">
          <div className="flex items-center gap-2 mb-6">
            <Zap size={22} fill="#FF2D78" stroke="#FF2D78" />
            <h1 className="text-2xl font-extrabold" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>Sign In</h1>
          </div>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="email" className="block text-xs font-semibold mb-1 uppercase tracking-wide" style={{ color: "#6B6052" }}>Email</label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                required
                value={form.email}
                onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                className="w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-colors"
                style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                onFocus={(e) => { e.target.style.borderColor = "#FF2D78"; }}
                onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; }}
              />
            </div>
            <div>
              <label htmlFor="password" className="block text-xs font-semibold mb-1 uppercase tracking-wide" style={{ color: "#6B6052" }}>Password</label>
              <input
                id="password"
                type="password"
                autoComplete="current-password"
                required
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                className="w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-colors"
                style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                onFocus={(e) => { e.target.style.borderColor = "#FF2D78"; }}
                onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; }}
              />
            </div>
            {error && (
              <p className="text-sm rounded-lg px-3 py-2" style={{ background: "#FFF0F3", color: "#e0245f" }} role="alert">
                {error}
              </p>
            )}
            <button
              type="submit"
              disabled={loading}
              className="w-full font-bold py-3 rounded-xl text-white transition-opacity hover:opacity-90 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed"
              style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
            >
              {loading ? "Signing in…" : "Sign In"}
            </button>
          </form>
          <p className="mt-5 text-sm text-center" style={{ color: "#6B6052" }}>
            New to ZapMarket?{" "}
            <Link href="/register" className="font-semibold hover:underline" style={{ color: "#FF2D78" }}>Create account</Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return <Suspense><LoginForm /></Suspense>;
}
