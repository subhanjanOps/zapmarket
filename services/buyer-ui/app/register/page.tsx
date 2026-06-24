"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Zap } from "lucide-react";

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({ full_name: "", email: "", password: "" });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      const data = await res.json();
      if (!res.ok) { setError((data as Record<string, string>).error ?? "Registration failed"); return; }
      router.push("/login");
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  const inputStyle = {
    border: "2px solid #F0EDE8",
    background: "#FFFCF5",
    color: "#1A1208",
  };

  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4 py-12" style={{ background: "#FFFCF5" }}>
      <div className="w-full max-w-sm rounded-2xl overflow-hidden" style={{ border: "1px solid #F0EDE8", background: "#fff" }}>
        <div className="h-1.5 w-full" style={{ background: "#00736A" }} />
        <div className="p-8">
          <div className="flex items-center gap-2 mb-6">
            <Zap size={22} fill="#FF2D78" stroke="#FF2D78" />
            <h1 className="text-2xl font-extrabold" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>Create Account</h1>
          </div>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="full_name" className="block text-xs font-semibold mb-1 uppercase tracking-wide" style={{ color: "#6B6052" }}>Full Name</label>
              <input
                id="full_name"
                type="text"
                autoComplete="name"
                required
                value={form.full_name}
                onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))}
                className="w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-colors"
                style={inputStyle}
                onFocus={(e) => { e.target.style.borderColor = "#00736A"; }}
                onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; }}
              />
            </div>
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
                style={inputStyle}
                onFocus={(e) => { e.target.style.borderColor = "#00736A"; }}
                onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; }}
              />
            </div>
            <div>
              <label htmlFor="password" className="block text-xs font-semibold mb-1 uppercase tracking-wide" style={{ color: "#6B6052" }}>Password</label>
              <input
                id="password"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                className="w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-colors"
                style={inputStyle}
                onFocus={(e) => { e.target.style.borderColor = "#00736A"; }}
                onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; }}
              />
              <p className="mt-1 text-xs" style={{ color: "#9CA3AF" }}>Minimum 8 characters</p>
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
              style={{ background: "#00736A", fontFamily: "var(--font-syne)" }}
            >
              {loading ? "Creating account…" : "Create Account"}
            </button>
          </form>
          <p className="mt-5 text-sm text-center" style={{ color: "#6B6052" }}>
            Already have an account?{" "}
            <Link href="/login" className="font-semibold hover:underline" style={{ color: "#FF2D78" }}>Sign in</Link>
          </p>
        </div>
      </div>
    </div>
  );
}
