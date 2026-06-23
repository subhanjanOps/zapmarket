"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

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

  return (
    <div className="min-h-[60vh] flex items-center justify-center px-4">
      <div className="w-full max-w-sm bg-white border border-gray-200 rounded-lg p-8">
        <h1 className="text-2xl font-bold mb-6 text-center">Create Account</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input type="text" placeholder="Full Name" required value={form.full_name}
            onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))}
            className="w-full border border-gray-300 rounded px-3 py-2 text-sm outline-none focus:border-[#FF9900]" />
          <input type="email" placeholder="Email" required value={form.email}
            onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
            className="w-full border border-gray-300 rounded px-3 py-2 text-sm outline-none focus:border-[#FF9900]" />
          <input type="password" placeholder="Password" required minLength={8} value={form.password}
            onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
            className="w-full border border-gray-300 rounded px-3 py-2 text-sm outline-none focus:border-[#FF9900]" />
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-[#FF9900] text-black font-semibold py-2 rounded hover:bg-[#e68900] disabled:opacity-50">
            {loading ? "Creating account…" : "Create Account"}
          </button>
        </form>
        <p className="mt-4 text-sm text-center text-gray-600">
          Already have an account?{" "}
          <Link href="/login" className="text-[#FF9900] hover:underline">Sign in</Link>
        </p>
      </div>
    </div>
  );
}
