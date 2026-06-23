"use client";
import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

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
    <div className="min-h-[60vh] flex items-center justify-center px-4">
      <div className="w-full max-w-sm bg-white border border-gray-200 rounded-lg p-8">
        <h1 className="text-2xl font-bold mb-6 text-center">Sign In</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input type="email" placeholder="Email" required value={form.email}
            onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
            className="w-full border border-gray-300 rounded px-3 py-2 text-sm outline-none focus:border-[#FF9900]" />
          <input type="password" placeholder="Password" required value={form.password}
            onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
            className="w-full border border-gray-300 rounded px-3 py-2 text-sm outline-none focus:border-[#FF9900]" />
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-[#FF9900] text-black font-semibold py-2 rounded hover:bg-[#e68900] disabled:opacity-50">
            {loading ? "Signing in…" : "Sign In"}
          </button>
        </form>
        <p className="mt-4 text-sm text-center text-gray-600">
          New to ZapMarket?{" "}
          <Link href="/register" className="text-[#FF9900] hover:underline">Create account</Link>
        </p>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return <Suspense><LoginForm /></Suspense>;
}
