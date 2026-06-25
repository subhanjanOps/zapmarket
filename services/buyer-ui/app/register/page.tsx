"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Zap, Eye, EyeOff, ArrowRight } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

export default function RegisterPage() {
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
      if (!res.ok) { setError((data as Record<string, string>).error ?? "Registration failed"); return; }
      router.push("/login");
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 py-16" style={{ background: "#FFFCF5" }}>
      <div className="w-full max-w-sm animate-scale-in">
        <Card
          className="rounded-3xl overflow-hidden border-0 p-0"
          style={{ border: "1px solid #F0EDE8", background: "#fff", boxShadow: "0 8px 40px rgba(26,18,8,0.08)" }}
        >
          {/* Teal/green accent strip */}
          <div className="h-1.5 bg-secondary" />

          <CardHeader className="px-8 pt-8 pb-0">
            <div className="flex items-center gap-2 mb-2">
              <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: "#00736A12" }}>
                <Zap size={20} fill="#00736A" stroke="#00736A" />
              </div>
              <div>
                <CardTitle
                  className="text-xl font-extrabold leading-none"
                  style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
                >
                  Create Account
                </CardTitle>
                <p className="text-xs mt-0.5" style={{ color: "#9CA3AF" }}>Join ZapMarket today</p>
              </div>
            </div>
          </CardHeader>

          <CardContent className="px-8 pt-6 pb-8">
            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Full Name */}
              <div className="space-y-1.5">
                <Label
                  htmlFor="full_name"
                  className="text-xs font-bold uppercase tracking-wider"
                  style={{ color: "#6B6052" }}
                >
                  Full Name
                </Label>
                <Input
                  id="full_name"
                  type="text"
                  autoComplete="name"
                  required
                  value={form.full_name}
                  onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))}
                  placeholder="Jane Doe"
                  className="rounded-xl text-sm outline-none transition-all duration-150 focus-visible:ring-0 focus-visible:ring-offset-0"
                  style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                  onFocus={(e) => { e.target.style.borderColor = "#00736A"; e.target.style.boxShadow = "0 0 0 3px rgba(0,115,106,0.08)"; }}
                  onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; e.target.style.boxShadow = "none"; }}
                />
              </div>

              {/* Email */}
              <div className="space-y-1.5">
                <Label
                  htmlFor="email"
                  className="text-xs font-bold uppercase tracking-wider"
                  style={{ color: "#6B6052" }}
                >
                  Email
                </Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={form.email}
                  onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                  placeholder="you@example.com"
                  className="rounded-xl text-sm outline-none transition-all duration-150 focus-visible:ring-0 focus-visible:ring-offset-0"
                  style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                  onFocus={(e) => { e.target.style.borderColor = "#00736A"; e.target.style.boxShadow = "0 0 0 3px rgba(0,115,106,0.08)"; }}
                  onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; e.target.style.boxShadow = "none"; }}
                />
              </div>

              {/* Password */}
              <div className="space-y-1.5">
                <Label
                  htmlFor="password"
                  className="text-xs font-bold uppercase tracking-wider"
                  style={{ color: "#6B6052" }}
                >
                  Password
                </Label>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPw ? "text" : "password"}
                    autoComplete="new-password"
                    required
                    minLength={8}
                    value={form.password}
                    onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                    placeholder="••••••••"
                    className="rounded-xl pr-11 text-sm outline-none transition-all duration-150 focus-visible:ring-0 focus-visible:ring-offset-0"
                    style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                    onFocus={(e) => { e.target.style.borderColor = "#00736A"; e.target.style.boxShadow = "0 0 0 3px rgba(0,115,106,0.08)"; }}
                    onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; e.target.style.boxShadow = "none"; }}
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
                <p className="text-xs" style={{ color: "#9CA3AF" }}>Minimum 8 characters</p>
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

              <Button
                type="submit"
                disabled={loading}
                className="w-full flex items-center justify-center gap-2 font-bold py-3.5 rounded-xl text-white
                           transition-all duration-200 hover:scale-[1.02] hover:shadow-lg active:scale-[0.98]
                           disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 cursor-pointer mt-2
                           bg-secondary hover:bg-secondary/90"
                style={{ fontFamily: "var(--font-syne)" }}
              >
                {loading ? "Creating account…" : (<>Create Account <ArrowRight size={16} /></>)}
              </Button>
            </form>

            <p className="mt-6 text-sm text-center" style={{ color: "#6B6052" }}>
              Already have an account?{" "}
              <Link href="/login" className="font-bold hover:underline" style={{ color: "#FF2D78" }}>
                Sign in
              </Link>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
