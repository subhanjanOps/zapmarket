"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { Zap, ArrowLeft, ArrowRight, Loader2, Store, Building2, FileText } from "lucide-react";

const CATEGORIES = [
  "Electronics", "Fashion", "Home & Living", "Beauty & Personal Care",
  "Sports & Fitness", "Books & Stationery", "Toys & Games",
  "Automotive", "Health & Wellness", "Other",
];

type Step = 1 | 2 | 3;

const STEPS = [
  { label: "Account",  icon: Store },
  { label: "Store",    icon: Building2 },
  { label: "Business", icon: FileText },
];

interface FormState {
  full_name: string; email: string; password: string;
  store_name: string; tagline: string; category: string;
  gstin: string; pan: string;
  business_phone: string; city: string; pincode: string;
}

const INITIAL: FormState = {
  full_name: "", email: "", password: "",
  store_name: "", tagline: "", category: "",
  gstin: "", pan: "",
  business_phone: "", city: "", pincode: "",
};

export default function SellerRegisterPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>(1);
  const [form, setForm] = useState<FormState>(INITIAL);
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function set(field: keyof FormState, value: string) {
    setForm(f => ({ ...f, [field]: value }));
  }

  function validateStep(): string | null {
    if (step === 1) {
      if (!form.full_name.trim()) return "Full name is required";
      if (!form.email.trim()) return "Email is required";
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) return "Enter a valid email address";
      if (form.password.length < 8) return "Password must be at least 8 characters";
    }
    if (step === 2) {
      if (!form.store_name.trim()) return "Store name is required";
      if (!form.category) return "Please select a category";
    }
    if (step === 3) {
      if (!form.business_phone.trim()) return "Business phone is required";
      if (!form.city.trim()) return "City is required";
      if (!form.pincode.trim()) return "Pincode is required";
      if (!form.gstin.trim() && !form.pan.trim()) return "Provide at least one of GSTIN or PAN";
    }
    return null;
  }

  function handleNext() {
    const err = validateStep();
    if (err) { setError(err); return; }
    setError(null);
    setStep(s => (s + 1) as Step);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const err = validateStep();
    if (err) { setError(err); return; }
    setError(null);
    setLoading(true);

    const payload = {
      full_name:      form.full_name,
      email:          form.email,
      password:       form.password,
      store_name:     form.store_name,
      tagline:        form.tagline,
      category:       form.category,
      gstin:          form.gstin || undefined,
      pan:            form.pan   || undefined,
      business_phone: form.business_phone,
      city:           form.city,
      pincode:        form.pincode,
    };

    try {
      const res = await fetch("/api/auth/register/seller", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        setError((data as Record<string, string>).error ?? "Registration failed. Please try again.");
        return;
      }
      router.push("/register/seller/pending");
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  const slideVariants = {
    enter: { opacity: 0, x: 24 },
    center: { opacity: 1, x: 0 },
    exit: { opacity: 0, x: -24 },
  };

  return (
    <div className="min-h-[calc(100vh-128px)] flex items-center justify-center p-6 bg-[#F9F8F5]">
      <div className="w-full max-w-md">
        <div className="flex items-center gap-3 mb-8">
          <Link
            href={step === 1 ? "/register" : "#"}
            onClick={step > 1 ? (e) => { e.preventDefault(); setError(null); setStep(s => (s - 1) as Step); } : undefined}
            className="p-1.5 rounded-lg hover:bg-black/5 transition-colors text-[#7A6856]"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex items-center gap-2">
            <div className="h-8 w-8 bg-[#4F46E5] rounded-xl flex items-center justify-center">
              <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-lg text-[#0F0A04]">Seller Registration</span>
          </div>
        </div>

        <div className="flex items-center gap-2 mb-8">
          {STEPS.map((s, i) => {
            const n = (i + 1) as Step;
            const active = step === n;
            const done = step > n;
            return (
              <div key={s.label} className="flex items-center gap-2 flex-1 last:flex-none">
                <div className={`flex items-center gap-1.5 ${active ? "text-[#4F46E5]" : done ? "text-green-600" : "text-[#C4B9AA]"}`}>
                  <div className={`h-7 w-7 rounded-full flex items-center justify-center text-xs font-bold transition-colors ${active ? "bg-[#4F46E5] text-white" : done ? "bg-green-600 text-white" : "bg-[#EDE9E3] text-[#B8A898]"}`}>
                    {done ? "✓" : n}
                  </div>
                  <span className="text-xs font-medium hidden sm:block">{s.label}</span>
                </div>
                {i < STEPS.length - 1 && (
                  <div className={`h-px flex-1 mx-1 transition-colors ${done ? "bg-green-400" : "bg-[#EDE9E3]"}`} />
                )}
              </div>
            );
          })}
        </div>

        <form onSubmit={step === 3 ? handleSubmit : (e) => { e.preventDefault(); handleNext(); }}>
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div key="step1" variants={slideVariants} initial="enter" animate="center" exit="exit" transition={{ duration: 0.2 }} className="space-y-4">
                <div className="mb-6">
                  <h2 className="font-display font-bold text-xl text-[#0F0A04]">Create your account</h2>
                  <p className="text-[#7A6856] text-sm mt-1">Your personal login credentials</p>
                </div>
                <Field label="Full name" id="full_name" type="text" autoComplete="name" placeholder="Jane Doe" value={form.full_name} onChange={v => set("full_name", v)} />
                <Field label="Email" id="email" type="email" autoComplete="email" placeholder="you@example.com" value={form.email} onChange={v => set("email", v)} />
                <div className="space-y-1.5">
                  <label htmlFor="password" className="text-sm font-semibold text-[#3D2E1A]">Password</label>
                  <div className="relative">
                    <input
                      id="password"
                      type={showPw ? "text" : "password"}
                      autoComplete="new-password"
                      placeholder="Minimum 8 characters"
                      value={form.password}
                      onChange={e => set("password", e.target.value)}
                      className="input-zap pr-11"
                    />
                    <button type="button" onClick={() => setShowPw(v => !v)} aria-label={showPw ? "Hide" : "Show"} className="absolute right-3.5 top-1/2 -translate-y-1/2 text-[#B8A898] hover:text-[#7A6856] transition-colors">
                      <span className="text-xs">{showPw ? "Hide" : "Show"}</span>
                    </button>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 2 && (
              <motion.div key="step2" variants={slideVariants} initial="enter" animate="center" exit="exit" transition={{ duration: 0.2 }} className="space-y-4">
                <div className="mb-6">
                  <h2 className="font-display font-bold text-xl text-[#0F0A04]">Your store</h2>
                  <p className="text-[#7A6856] text-sm mt-1">Tell buyers about your shop</p>
                </div>
                <Field label="Store name" id="store_name" type="text" placeholder="e.g. Priya's Electronics" value={form.store_name} onChange={v => set("store_name", v)} />
                <Field label="Tagline (optional)" id="tagline" type="text" placeholder="e.g. Best prices on genuine parts" value={form.tagline} onChange={v => set("tagline", v)} />
                <div className="space-y-1.5">
                  <label htmlFor="category" className="text-sm font-semibold text-[#3D2E1A]">Primary category</label>
                  <select
                    id="category"
                    value={form.category}
                    onChange={e => set("category", e.target.value)}
                    className="input-zap"
                  >
                    <option value="" disabled>Select a category</option>
                    {CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div key="step3" variants={slideVariants} initial="enter" animate="center" exit="exit" transition={{ duration: 0.2 }} className="space-y-4">
                <div className="mb-6">
                  <h2 className="font-display font-bold text-xl text-[#0F0A04]">Business details</h2>
                  <p className="text-[#7A6856] text-sm mt-1">Required for KYC — reviewed by our team</p>
                </div>
                <Field label="Business phone" id="business_phone" type="tel" placeholder="9876543210" value={form.business_phone} onChange={v => set("business_phone", v)} />
                <div className="grid grid-cols-2 gap-3">
                  <Field label="City" id="city" type="text" placeholder="Bangalore" value={form.city} onChange={v => set("city", v)} />
                  <Field label="Pincode" id="pincode" type="text" placeholder="560001" value={form.pincode} onChange={v => set("pincode", v)} />
                </div>
                <div className="pt-1">
                  <p className="text-xs font-semibold text-[#3D2E1A] mb-2">KYC — provide at least one</p>
                  <div className="space-y-3">
                    <Field label="GSTIN (optional)" id="gstin" type="text" placeholder="29ABCDE1234F1Z5" value={form.gstin} onChange={v => set("gstin", v)} />
                    <Field label="PAN (optional)" id="pan" type="text" placeholder="ABCDE1234F" value={form.pan} onChange={v => set("pan", v)} />
                  </div>
                </div>
                <p className="text-xs text-[#B8A898] leading-relaxed pt-1">
                  Your KYC details are encrypted and reviewed only by our compliance team.
                  By submitting you agree to our{" "}
                  <Link href="/seller-terms" className="underline hover:text-[#7A6856]">Seller Terms</Link>.
                </p>
              </motion.div>
            )}
          </AnimatePresence>

          {error && (
            <p className="mt-4 text-sm text-[#DC2626] bg-red-50 border border-red-100 rounded-xl px-3 py-2" role="alert">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full h-11 mt-6 bg-[#4F46E5] hover:bg-[#3730A3] disabled:opacity-60 text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(79,70,229,0.25)] flex items-center justify-center gap-2"
          >
            {loading ? (
              <><Loader2 className="h-4 w-4 animate-spin" />Submitting…</>
            ) : step < 3 ? (
              <>Continue<ArrowRight className="h-4 w-4" /></>
            ) : (
              <>Submit application<ArrowRight className="h-4 w-4" /></>
            )}
          </button>
        </form>

        <p className="mt-5 text-sm text-center text-[#7A6856]">
          Already have an account?{" "}
          <Link href="/login" className="font-semibold text-[#4F46E5] hover:text-[#3730A3] transition-colors">Sign in</Link>
        </p>
      </div>
    </div>
  );
}

function Field({ label, id, type, placeholder, value, onChange, autoComplete }: {
  label: string; id: string; type: string; placeholder: string;
  value: string; onChange: (v: string) => void; autoComplete?: string;
}) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="text-sm font-semibold text-[#3D2E1A]">{label}</label>
      <input
        id={id} type={type} autoComplete={autoComplete}
        placeholder={placeholder} value={value}
        onChange={e => onChange(e.target.value)}
        className="input-zap"
      />
    </div>
  );
}
