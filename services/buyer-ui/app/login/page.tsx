"use client";
import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Zap, Eye, EyeOff, ArrowRight } from "lucide-react";

function LoginForm() {
  const router = useRouter();
  const sp = useSearchParams();
  const next = sp.get("next") ?? "/";
  const [form, setForm] = useState({ email: "", password: "" });
  const [showPw, setShowPw] = useState(false);
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
    <div
      className="min-h-[80vh] flex items-center justify-center px-4 py-16"
      style={{ background: "#FFFCF5" }}
    >
      <div className="w-full max-w-sm animate-scale-in">
        {/* Card */}
        <div
          className="rounded-3xl overflow-hidden"
          style={{
            border: "1px solid #F0EDE8",
            background: "#fff",
            boxShadow: "0 8px 40px rgba(26,18,8,0.08)",
          }}
        >
          {/* Top band */}
          <div className="h-1.5" style={{ background: "linear-gradient(90deg, #FF2D78, #FF8C00)" }} />

          <div className="p-8">
            {/* Logo */}
            <div className="flex items-center gap-2 mb-8">
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center"
                style={{ background: "#FF2D7812" }}
              >
                <Zap size={20} fill="#FF2D78" stroke="#FF2D78" />
              </div>
              <div>
                <h1
                  className="text-xl font-extrabold leading-none"
                  style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
                >
                  Sign In
                </h1>
                <p className="text-xs mt-0.5" style={{ color: "#9CA3AF" }}>Welcome back!</p>
              </div>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="email" className="block text-xs font-bold mb-1.5 uppercase tracking-wider" style={{ color: "#6B6052" }}>
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={form.email}
                  onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                  className="w-full rounded-xl px-4 py-3 text-sm outline-none transition-all duration-150"
                  style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                  onFocus={(e) => { e.target.style.borderColor = "#FF2D78"; e.target.style.boxShadow = "0 0 0 3px rgba(255,45,120,0.08)"; }}
                  onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; e.target.style.boxShadow = "none"; }}
                  placeholder="you@example.com"
                />
              </div>

              <div>
                <label htmlFor="password" className="block text-xs font-bold mb-1.5 uppercase tracking-wider" style={{ color: "#6B6052" }}>
                  Password
                </label>
                <div className="relative">
                  <input
                    id="password"
                    type={showPw ? "text" : "password"}
                    autoComplete="current-password"
                    required
                    value={form.password}
                    onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                    className="w-full rounded-xl px-4 py-3 pr-11 text-sm outline-none transition-all duration-150"
                    style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                    onFocus={(e) => { e.target.style.borderColor = "#FF2D78"; e.target.style.boxShadow = "0 0 0 3px rgba(255,45,120,0.08)"; }}
                    onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; e.target.style.boxShadow = "none"; }}
                    placeholder="••••••••"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPw((v) => !v)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 p-1 rounded cursor-pointer transition-opacity hover:opacity-60"
                    style={{ color: "#9CA3AF" }}
                    aria-label={showPw ? "Hide password" : "Show password"}
                  >
                    {showPw ? <EyeOff size={15} /> : <Eye size={15} />}
                  </button>
                </div>
              </div>

              {error && (
                <div
                  className="text-sm rounded-xl px-4 py-3 animate-slide-up-sm"
                  style={{ background: "#FFF0F3", color: "#e0245f", border: "1px solid #FFD0DC" }}
                  role="alert"
                >
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="w-full flex items-center justify-center gap-2 font-bold py-3.5 rounded-xl text-white
                           transition-all duration-200 hover:scale-[1.02] hover:shadow-lg active:scale-[0.98]
                           disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 cursor-pointer mt-2"
                style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
              >
                {loading ? "Signing in…" : (
                  <>Sign In <ArrowRight size={16} /></>
                )}
              </button>
            </form>

            <p className="mt-6 text-sm text-center" style={{ color: "#6B6052" }}>
              New to ZapMarket?{" "}
              <Link href="/register" className="font-bold hover:underline" style={{ color: "#FF2D78" }}>
                Create account
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return <Suspense><LoginForm /></Suspense>;
}
