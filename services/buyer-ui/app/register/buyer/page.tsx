"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { Zap, Eye, EyeOff, ArrowRight, Loader2, CheckCircle2, ArrowLeft } from "lucide-react";

const PERKS = [
  "Free delivery on your first 3 orders",
  "Exclusive member-only deals",
  "Early access to flash sales",
  "Hassle-free returns & refunds",
];

export default function BuyerRegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({ full_name: "", email: "", password: "" });
  const [showPw, setShowPw] = useState(false);
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
      if (!res.ok) {
        setError((data as Record<string, string>).error ?? "Registration failed");
        return;
      }
      router.push("/");
      router.refresh();
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[calc(100vh-128px)] grid lg:grid-cols-2">
      {/* Brand panel */}
      <div className="hidden lg:flex flex-col justify-between bg-gradient-to-br from-[#0F0A04] via-[#1C0F08] to-[#0F1A2A] p-12 relative overflow-hidden">
        <div className="absolute top-0 right-0 h-64 w-64 bg-[#4F46E5]/10 rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 h-48 w-48 bg-[#E91E8C]/8 rounded-full blur-3xl" />

        <div className="flex items-center gap-2 relative z-10">
          <Link href="/register" className="mr-2 p-1.5 rounded-lg hover:bg-white/10 transition-colors text-white/50 hover:text-white">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <Link href="/" className="flex items-center gap-2.5">
            <div className="h-9 w-9 bg-[#E91E8C] rounded-xl flex items-center justify-center shadow-[0_2px_12px_rgba(233,30,140,0.4)]">
              <Zap className="h-5 w-5 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-xl text-white">ZapMarket</span>
          </Link>
        </div>

        <div className="relative z-10">
          <h2 className="font-display font-bold text-white text-2xl mb-2 leading-tight">
            Join 2 million+ smart shoppers
          </h2>
          <p className="text-white/50 text-sm mb-7 leading-relaxed">
            Get access to the best deals, verified sellers, and a shopping experience unlike any other.
          </p>
          <ul className="space-y-3">
            {PERKS.map(perk => (
              <li key={perk} className="flex items-start gap-3">
                <CheckCircle2 className="h-4 w-4 text-[#E91E8C] shrink-0 mt-0.5" />
                <span className="text-white/70 text-sm">{perk}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Form panel */}
      <div className="flex items-center justify-center p-6 sm:p-12 bg-[#F9F8F5]">
        <motion.div
          className="w-full max-w-sm"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        >
          <div className="flex items-center gap-2 mb-8 lg:hidden">
            <Link href="/register" className="p-1.5 rounded-lg hover:bg-black/5 transition-colors text-[#7A6856]">
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center">
              <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-lg text-[#0F0A04]">ZapMarket</span>
          </div>

          <div className="mb-7">
            <h1 className="font-display font-bold text-2xl text-[#0F0A04] mb-1.5">
              Create your account
            </h1>
            <p className="text-[#7A6856] text-sm">Free forever · No credit card required</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label htmlFor="full_name" className="text-sm font-semibold text-[#3D2E1A]">Full name</label>
              <input
                id="full_name"
                type="text"
                autoComplete="name"
                required
                placeholder="Jane Doe"
                value={form.full_name}
                onChange={e => setForm(f => ({ ...f, full_name: e.target.value }))}
                className="input-zap"
              />
            </div>

            <div className="space-y-1.5">
              <label htmlFor="email" className="text-sm font-semibold text-[#3D2E1A]">Email</label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                required
                placeholder="you@example.com"
                value={form.email}
                onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                className="input-zap"
              />
            </div>

            <div className="space-y-1.5">
              <label htmlFor="password" className="text-sm font-semibold text-[#3D2E1A]">Password</label>
              <div className="relative">
                <input
                  id="password"
                  type={showPw ? "text" : "password"}
                  autoComplete="new-password"
                  required
                  minLength={8}
                  placeholder="Minimum 8 characters"
                  value={form.password}
                  onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                  className="input-zap pr-11"
                />
                <button
                  type="button"
                  onClick={() => setShowPw(v => !v)}
                  aria-label={showPw ? "Hide password" : "Show password"}
                  className="absolute right-3.5 top-1/2 -translate-y-1/2 p-0.5 text-[#B8A898] hover:text-[#7A6856] transition-colors"
                >
                  {showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {error && (
              <p className="text-sm text-[#DC2626] bg-red-50 border border-red-100 rounded-xl px-3 py-2" role="alert">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full h-11 bg-[#E91E8C] hover:bg-[#B5166E] disabled:opacity-60 text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(233,30,140,0.25)] flex items-center justify-center gap-2 mt-2"
            >
              {loading ? (
                <><Loader2 className="h-4 w-4 animate-spin" />Creating account…</>
              ) : (
                <>Create account<ArrowRight className="h-4 w-4" /></>
              )}
            </button>

            <p className="text-xs text-[#B8A898] text-center leading-relaxed">
              By creating an account you agree to our{" "}
              <Link href="/terms" className="underline hover:text-[#7A6856]">Terms</Link>
              {" "}and{" "}
              <Link href="/privacy" className="underline hover:text-[#7A6856]">Privacy Policy</Link>.
            </p>
          </form>

          <p className="mt-5 text-sm text-center text-[#7A6856]">
            Already have an account?{" "}
            <Link href="/login" className="font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors">Sign in</Link>
          </p>
        </motion.div>
      </div>
    </div>
  );
}
