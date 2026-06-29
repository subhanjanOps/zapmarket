"use client";

import { useReducer, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import {
  Zap, ArrowLeft, ArrowRight, Loader2, CheckCircle2,
  Eye, EyeOff, Upload, User, Phone, Mail,
} from "lucide-react";

// ── Types ────────────────────────────────────────────────────────────────────

type Role = "buyer" | "seller";

type State = {
  step: 1 | 2 | 3 | 4;
  userId: string;
  // Step 1
  fullName: string;
  email: string;
  phone: string;
  password: string;
  confirmPassword: string;
  // Step 2
  emailCode: string;
  phoneCode: string;
  emailVerified: boolean;
  phoneVerified: boolean;
  // Step 3
  dob: string;
  gender: string;
  pfpFile: File | null;
  pfpPreview: string;
  pfpUrl: string;
  line1: string;
  city: string;
  state: string;
  pincode: string;
  // Seller-only Step 3
  storeName: string;
  tagline: string;
  category: string;
  // Step 4
  termsAccepted: boolean;
};

type Action =
  | { type: "SET"; payload: Partial<State> }
  | { type: "NEXT" }
  | { type: "PREV" };

function reducer(s: State, a: Action): State {
  switch (a.type) {
    case "SET": return { ...s, ...a.payload };
    case "NEXT": return { ...s, step: Math.min(s.step + 1, 4) as State["step"] };
    case "PREV": return { ...s, step: Math.max(s.step - 1, 1) as State["step"] };
    default: return s;
  }
}

const initial: State = {
  step: 1, userId: "",
  fullName: "", email: "", phone: "", password: "", confirmPassword: "",
  emailCode: "", phoneCode: "", emailVerified: false, phoneVerified: false,
  dob: "", gender: "", pfpFile: null, pfpPreview: "", pfpUrl: "",
  line1: "", city: "", state: "", pincode: "",
  storeName: "", tagline: "", category: "",
  termsAccepted: false,
};

// ── Sub-components ───────────────────────────────────────────────────────────

function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <div className="flex items-center gap-2 mb-8">
      {Array.from({ length: total }).map((_, i) => (
        <div key={i} className={`h-1.5 flex-1 rounded-full transition-all duration-300 ${i < current ? "bg-[#E91E8C]" : i === current - 1 ? "bg-[#E91E8C]" : "bg-[#E8E3DC]"}`} />
      ))}
    </div>
  );
}

function OTPInput({ value, onChange, label }: { value: string; onChange: (v: string) => void; label: string }) {
  return (
    <div className="space-y-1.5">
      <label className="text-sm font-semibold text-[#3D2E1A]">{label}</label>
      <input
        type="text"
        inputMode="numeric"
        maxLength={6}
        value={value}
        onChange={e => onChange(e.target.value.replace(/\D/g, "").slice(0, 6))}
        placeholder="000000"
        className="input-zap tracking-[0.3em] text-center font-mono"
      />
    </div>
  );
}

// ── Wizard ───────────────────────────────────────────────────────────────────

export default function RegistrationWizard({ role }: { role: Role }) {
  const router = useRouter();
  const [s, dispatch] = useReducer(reducer, initial);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [showPw, setShowPw] = useState(false);
  const [emailOtpSent, setEmailOtpSent] = useState(false);
  const [phoneOtpSent, setPhoneOtpSent] = useState(false);
  const pfpRef = useRef<HTMLInputElement>(null);

  const set = (p: Partial<State>) => dispatch({ type: "SET", payload: p });

  // ── Step 1: Register ───────────────────────────────────────────────────────
  async function submitStep1() {
    if (s.password !== s.confirmPassword) { setError("Passwords do not match"); return; }
    if (s.password.length < 8) { setError("Password must be at least 8 characters"); return; }
    setError(null); setLoading(true);
    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ full_name: s.fullName, email: s.email, phone: s.phone, password: s.password, role }),
      });
      const data = await res.json() as { ok?: boolean; error?: string; user_id?: string };
      if (!res.ok) { setError(data.error ?? "Registration failed"); return; }
      set({ userId: data.user_id ?? "" });
      dispatch({ type: "NEXT" });
      // Auto-send email OTP
      sendEmailOTP(data.user_id ?? "");
    } finally { setLoading(false); }
  }

  async function sendEmailOTP(uid?: string) {
    const userId = uid ?? s.userId;
    await fetch("/api/auth/otp/send", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ user_id: userId }),
    });
    setEmailOtpSent(true);
  }

  async function sendPhoneOTP() {
    if (!s.phone) { setError("Phone number required"); return; }
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/auth/phone-otp/send", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: s.userId, phone: s.phone }),
      });
      const data = await res.json() as { error?: string };
      if (!res.ok) { setError(data.error ?? "Failed to send phone OTP"); return; }
      setPhoneOtpSent(true);
    } finally { setLoading(false); }
  }

  async function verifyEmailOTP() {
    if (s.emailCode.length !== 6) { setError("Enter the 6-digit code"); return; }
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/auth/otp/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: s.userId, code: s.emailCode }),
      });
      const data = await res.json() as { error?: string };
      if (!res.ok) { setError(data.error ?? "Invalid OTP"); return; }
      set({ emailVerified: true });
    } finally { setLoading(false); }
  }

  async function verifyPhoneOTP() {
    if (s.phoneCode.length !== 6) { setError("Enter the 6-digit code"); return; }
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/auth/phone-otp/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: s.userId, code: s.phoneCode }),
      });
      const data = await res.json() as { error?: string };
      if (!res.ok) { setError(data.error ?? "Invalid OTP"); return; }
      set({ phoneVerified: true });
    } finally { setLoading(false); }
  }

  // ── Step 3: Profile ────────────────────────────────────────────────────────
  async function uploadPFP(file: File): Promise<string> {
    const fd = new FormData();
    fd.append("picture", file);
    fd.append("user_id", s.userId);
    const res = await fetch("/api/auth/profile/picture", { method: "POST", body: fd });
    const data = await res.json() as { pfp_url?: string; error?: string };
    if (!res.ok) throw new Error(data.error ?? "Upload failed");
    return data.pfp_url ?? "";
  }

  async function submitStep3() {
    setLoading(true); setError(null);
    try {
      let pfpUrl = s.pfpUrl;
      if (s.pfpFile && !s.pfpUrl) {
        pfpUrl = await uploadPFP(s.pfpFile);
        set({ pfpUrl });
      }
      const body: Record<string, unknown> = {
        user_id: s.userId,
        dob: s.dob || undefined,
        gender: s.gender || undefined,
        pfp_url: pfpUrl || undefined,
        address_line1: s.line1 || undefined,
        city: s.city || undefined,
        state: s.state || undefined,
        pincode: s.pincode || undefined,
      };
      if (role === "seller") {
        body.store_name = s.storeName || undefined;
        body.tagline = s.tagline || undefined;
        body.category = s.category || undefined;
      }
      const res = await fetch("/api/auth/registration/profile", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json() as { error?: string };
      if (!res.ok) { setError(data.error ?? "Failed to save profile"); return; }
      dispatch({ type: "NEXT" });
    } finally { setLoading(false); }
  }

  // ── Step 4: Complete ───────────────────────────────────────────────────────
  async function submitStep4() {
    if (!s.termsAccepted) { setError("Please accept the terms to continue"); return; }
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/auth/registration/complete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: s.userId, terms_accepted: true }),
      });
      const data = await res.json() as { error?: string };
      if (!res.ok) { setError(data.error ?? "Failed to complete registration"); return; }
      if (role === "seller") {
        router.push("/register/seller/pending");
      } else {
        router.push("/");
        router.refresh();
      }
    } finally { setLoading(false); }
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  const stepTitles = [
    "Create your account",
    "Verify your identity",
    "Complete your profile",
    "Review & submit",
  ];

  return (
    <div className="min-h-[calc(100vh-128px)] grid lg:grid-cols-[1fr_1.2fr]">
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
          <p className="text-white/40 text-xs uppercase tracking-widest mb-2">Step {s.step} of 4</p>
          <h2 className="font-display font-bold text-white text-2xl mb-2 leading-tight">
            {stepTitles[s.step - 1]}
          </h2>
          <p className="text-white/50 text-sm leading-relaxed">
            {role === "seller"
              ? "Set up your store and reach millions of buyers."
              : "Join 2 million+ smart shoppers on ZapMarket."}
          </p>
        </div>
      </div>

      {/* Form panel */}
      <div className="flex items-center justify-center p-6 sm:p-12 bg-[#F9F8F5]">
        <motion.div
          className="w-full max-w-md"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
        >
          <div className="flex items-center gap-2 mb-6 lg:hidden">
            <Link href="/register" className="p-1.5 rounded-lg hover:bg-black/5 transition-colors text-[#7A6856]">
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center">
              <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
            </div>
            <span className="font-display font-bold text-lg text-[#0F0A04]">ZapMarket</span>
          </div>

          <StepIndicator current={s.step} total={4} />

          <AnimatePresence mode="wait">
            {s.step === 1 && (
              <motion.div key="step1" initial={{ opacity: 0, x: 30 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -30 }} transition={{ duration: 0.3 }}>
                <div className="mb-6">
                  <h1 className="font-display font-bold text-2xl text-[#0F0A04] mb-1">Create your account</h1>
                  <p className="text-[#7A6856] text-sm">
                    {role === "seller" ? "Start selling on ZapMarket" : "Free forever · No credit card required"}
                  </p>
                </div>
                <div className="space-y-4">
                  <Field label="Full name">
                    <input id="full_name" type="text" autoComplete="name" required placeholder="Jane Doe"
                      value={s.fullName} onChange={e => set({ fullName: e.target.value })} className="input-zap" />
                  </Field>
                  <Field label="Email">
                    <input id="email" type="email" autoComplete="email" required placeholder="you@example.com"
                      value={s.email} onChange={e => set({ email: e.target.value })} className="input-zap" />
                  </Field>
                  <Field label="Phone number">
                    <input id="phone" type="tel" autoComplete="tel" required placeholder="+91 98765 43210"
                      value={s.phone} onChange={e => set({ phone: e.target.value })} className="input-zap" />
                  </Field>
                  <Field label="Password">
                    <div className="relative">
                      <input id="password" type={showPw ? "text" : "password"} autoComplete="new-password" required
                        minLength={8} placeholder="Minimum 8 characters"
                        value={s.password} onChange={e => set({ password: e.target.value })} className="input-zap pr-11" />
                      <button type="button" onClick={() => setShowPw(v => !v)}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 text-[#B8A898] hover:text-[#7A6856]">
                        {showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </Field>
                  <Field label="Confirm password">
                    <input type="password" autoComplete="new-password" required placeholder="Re-enter password"
                      value={s.confirmPassword} onChange={e => set({ confirmPassword: e.target.value })} className="input-zap" />
                  </Field>
                  <ErrorBox error={error} />
                  <Btn onClick={submitStep1} loading={loading} label="Continue" />
                  <p className="text-xs text-[#B8A898] text-center">
                    Already have an account?{" "}
                    <Link href="/login" className="text-[#E91E8C] font-semibold hover:text-[#B5166E]">Sign in</Link>
                  </p>
                </div>
              </motion.div>
            )}

            {s.step === 2 && (
              <motion.div key="step2" initial={{ opacity: 0, x: 30 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -30 }} transition={{ duration: 0.3 }}>
                <div className="mb-6">
                  <h1 className="font-display font-bold text-2xl text-[#0F0A04] mb-1">Verify your identity</h1>
                  <p className="text-[#7A6856] text-sm">Confirm your email and phone to continue.</p>
                </div>
                <div className="space-y-6">
                  {/* Email OTP */}
                  <div className="p-4 rounded-2xl border border-[#E8E3DC] bg-white space-y-3">
                    <div className="flex items-center gap-2">
                      <Mail className="h-4 w-4 text-[#E91E8C]" />
                      <span className="text-sm font-semibold text-[#3D2E1A]">Email verification</span>
                      {s.emailVerified && <CheckCircle2 className="h-4 w-4 text-green-500 ml-auto" />}
                    </div>
                    <p className="text-xs text-[#7A6856]">Code sent to {s.email}</p>
                    {!s.emailVerified && (
                      <>
                        <OTPInput value={s.emailCode} onChange={v => set({ emailCode: v })} label="6-digit code" />
                        <div className="flex gap-2">
                          <button onClick={verifyEmailOTP} disabled={loading || s.emailCode.length !== 6}
                            className="flex-1 h-9 bg-[#E91E8C] disabled:opacity-50 text-white text-sm font-semibold rounded-xl flex items-center justify-center gap-1">
                            {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Verify"}
                          </button>
                          <button onClick={() => sendEmailOTP()} disabled={loading}
                            className="h-9 px-3 border border-[#E8E3DC] text-sm text-[#7A6856] rounded-xl hover:bg-[#F0EDE8]">
                            Resend
                          </button>
                        </div>
                      </>
                    )}
                  </div>

                  {/* Phone OTP */}
                  <div className="p-4 rounded-2xl border border-[#E8E3DC] bg-white space-y-3">
                    <div className="flex items-center gap-2">
                      <Phone className="h-4 w-4 text-[#E91E8C]" />
                      <span className="text-sm font-semibold text-[#3D2E1A]">Phone verification</span>
                      {s.phoneVerified && <CheckCircle2 className="h-4 w-4 text-green-500 ml-auto" />}
                    </div>
                    <p className="text-xs text-[#7A6856]">{s.phone}</p>
                    {!s.phoneVerified && (
                      <>
                        {!phoneOtpSent ? (
                          <button onClick={sendPhoneOTP} disabled={loading}
                            className="w-full h-9 border border-[#E91E8C] text-[#E91E8C] text-sm font-semibold rounded-xl hover:bg-pink-50 flex items-center justify-center gap-1">
                            {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Send SMS code"}
                          </button>
                        ) : (
                          <>
                            <OTPInput value={s.phoneCode} onChange={v => set({ phoneCode: v })} label="6-digit code" />
                            <div className="flex gap-2">
                              <button onClick={verifyPhoneOTP} disabled={loading || s.phoneCode.length !== 6}
                                className="flex-1 h-9 bg-[#E91E8C] disabled:opacity-50 text-white text-sm font-semibold rounded-xl flex items-center justify-center gap-1">
                                {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Verify"}
                              </button>
                              <button onClick={sendPhoneOTP} disabled={loading}
                                className="h-9 px-3 border border-[#E8E3DC] text-sm text-[#7A6856] rounded-xl hover:bg-[#F0EDE8]">
                                Resend
                              </button>
                            </div>
                          </>
                        )}
                      </>
                    )}
                  </div>

                  <ErrorBox error={error} />
                  <Btn
                    onClick={() => { setError(null); dispatch({ type: "NEXT" }); }}
                    loading={false}
                    label={s.emailVerified && s.phoneVerified ? "Continue" : "Skip for now"}
                    disabled={false}
                  />
                </div>
              </motion.div>
            )}

            {s.step === 3 && (
              <motion.div key="step3" initial={{ opacity: 0, x: 30 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -30 }} transition={{ duration: 0.3 }}>
                <div className="mb-6">
                  <h1 className="font-display font-bold text-2xl text-[#0F0A04] mb-1">Complete your profile</h1>
                  <p className="text-[#7A6856] text-sm">Help us personalise your experience.</p>
                </div>
                <div className="space-y-4">
                  {/* PFP */}
                  <div className="flex items-center gap-4">
                    <button type="button" onClick={() => pfpRef.current?.click()}
                      className="h-16 w-16 rounded-full border-2 border-dashed border-[#E8E3DC] bg-white flex items-center justify-center hover:border-[#E91E8C] transition-colors overflow-hidden shrink-0">
                      {s.pfpPreview ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={s.pfpPreview} alt="Preview" className="h-full w-full object-cover" />
                      ) : (
                        <User className="h-6 w-6 text-[#B8A898]" />
                      )}
                    </button>
                    <div>
                      <button type="button" onClick={() => pfpRef.current?.click()}
                        className="text-sm font-semibold text-[#E91E8C] hover:text-[#B5166E] flex items-center gap-1">
                        <Upload className="h-3.5 w-3.5" /> Upload photo
                      </button>
                      <p className="text-xs text-[#B8A898] mt-0.5">JPEG or PNG, max 5MB</p>
                    </div>
                    <input ref={pfpRef} type="file" accept="image/jpeg,image/png" className="hidden"
                      onChange={e => {
                        const file = e.target.files?.[0];
                        if (!file) return;
                        set({ pfpFile: file, pfpUrl: "", pfpPreview: URL.createObjectURL(file) });
                      }} />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <Field label="Date of birth">
                      <input type="date" value={s.dob} onChange={e => set({ dob: e.target.value })} className="input-zap" />
                    </Field>
                    <Field label="Gender">
                      <select value={s.gender} onChange={e => set({ gender: e.target.value })} className="input-zap">
                        <option value="">Prefer not to say</option>
                        <option value="male">Male</option>
                        <option value="female">Female</option>
                        <option value="non_binary">Non-binary</option>
                        <option value="other">Other</option>
                      </select>
                    </Field>
                  </div>

                  <Field label="Address line 1">
                    <input type="text" placeholder="Flat / house / building" value={s.line1} onChange={e => set({ line1: e.target.value })} className="input-zap" />
                  </Field>
                  <div className="grid grid-cols-3 gap-3">
                    <Field label="City">
                      <input type="text" placeholder="Mumbai" value={s.city} onChange={e => set({ city: e.target.value })} className="input-zap" />
                    </Field>
                    <Field label="State">
                      <input type="text" placeholder="MH" value={s.state} onChange={e => set({ state: e.target.value })} className="input-zap" />
                    </Field>
                    <Field label="Pincode">
                      <input type="text" placeholder="400001" value={s.pincode} onChange={e => set({ pincode: e.target.value })} className="input-zap" />
                    </Field>
                  </div>

                  {role === "seller" && (
                    <>
                      <hr className="border-[#E8E3DC]" />
                      <p className="text-xs font-semibold text-[#7A6856] uppercase tracking-wide">Store details</p>
                      <Field label="Store name">
                        <input type="text" required placeholder="My Awesome Store" value={s.storeName} onChange={e => set({ storeName: e.target.value })} className="input-zap" />
                      </Field>
                      <Field label="Tagline">
                        <input type="text" placeholder="Quality products, fast delivery" value={s.tagline} onChange={e => set({ tagline: e.target.value })} className="input-zap" />
                      </Field>
                      <Field label="Category">
                        <input type="text" placeholder="Electronics, Fashion, etc." value={s.category} onChange={e => set({ category: e.target.value })} className="input-zap" />
                      </Field>
                    </>
                  )}

                  <ErrorBox error={error} />
                  <Btn onClick={submitStep3} loading={loading} label="Continue" />
                </div>
              </motion.div>
            )}

            {s.step === 4 && (
              <motion.div key="step4" initial={{ opacity: 0, x: 30 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -30 }} transition={{ duration: 0.3 }}>
                <div className="mb-6">
                  <h1 className="font-display font-bold text-2xl text-[#0F0A04] mb-1">Almost there!</h1>
                  <p className="text-[#7A6856] text-sm">Review your details and agree to our terms.</p>
                </div>
                <div className="space-y-4">
                  <div className="rounded-2xl border border-[#E8E3DC] bg-white divide-y divide-[#F0EDE8]">
                    <ReviewRow label="Name" value={s.fullName} />
                    <ReviewRow label="Email" value={s.email} badge={s.emailVerified ? "Verified" : "Unverified"} />
                    <ReviewRow label="Phone" value={s.phone} badge={s.phoneVerified ? "Verified" : "Unverified"} />
                    {s.dob && <ReviewRow label="Date of birth" value={s.dob} />}
                    {s.gender && <ReviewRow label="Gender" value={s.gender} />}
                    {s.city && <ReviewRow label="City" value={`${s.city}${s.state ? ", " + s.state : ""}`} />}
                    {role === "seller" && s.storeName && <ReviewRow label="Store name" value={s.storeName} />}
                  </div>

                  <label className="flex items-start gap-3 cursor-pointer select-none">
                    <input type="checkbox" checked={s.termsAccepted} onChange={e => set({ termsAccepted: e.target.checked })}
                      className="mt-0.5 h-4 w-4 rounded border-[#E8E3DC] text-[#E91E8C] accent-[#E91E8C]" />
                    <span className="text-sm text-[#3D2E1A]">
                      I agree to the{" "}
                      <Link href="/terms" className="text-[#E91E8C] hover:text-[#B5166E] font-medium">Terms of Service</Link>
                      {" "}and{" "}
                      <Link href="/privacy" className="text-[#E91E8C] hover:text-[#B5166E] font-medium">Privacy Policy</Link>.
                      I confirm I am at least 18 years old.
                    </span>
                  </label>

                  <ErrorBox error={error} />
                  <Btn onClick={submitStep4} loading={loading} label={role === "seller" ? "Submit for review" : "Complete registration"} disabled={!s.termsAccepted} />
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {s.step > 1 && (
            <button onClick={() => { setError(null); dispatch({ type: "PREV" }); }}
              className="mt-4 flex items-center gap-1.5 text-sm text-[#7A6856] hover:text-[#3D2E1A] transition-colors">
              <ArrowLeft className="h-3.5 w-3.5" /> Back
            </button>
          )}
        </motion.div>
      </div>
    </div>
  );
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label className="text-sm font-semibold text-[#3D2E1A]">{label}</label>
      {children}
    </div>
  );
}

function ErrorBox({ error }: { error: string | null }) {
  if (!error) return null;
  return (
    <p className="text-sm text-[#DC2626] bg-red-50 border border-red-100 rounded-xl px-3 py-2" role="alert">
      {error}
    </p>
  );
}

function Btn({ onClick, loading, label, disabled }: { onClick: () => void; loading: boolean; label: string; disabled?: boolean }) {
  return (
    <button type="button" onClick={onClick} disabled={loading || disabled}
      className="w-full h-11 bg-[#E91E8C] hover:bg-[#B5166E] disabled:opacity-60 text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(233,30,140,0.25)] flex items-center justify-center gap-2">
      {loading ? <><Loader2 className="h-4 w-4 animate-spin" />Please wait…</> : <>{label}<ArrowRight className="h-4 w-4" /></>}
    </button>
  );
}

function ReviewRow({ label, value, badge }: { label: string; value: string; badge?: string }) {
  return (
    <div className="flex items-center justify-between px-4 py-3">
      <span className="text-xs text-[#7A6856] w-28 shrink-0">{label}</span>
      <span className="text-sm text-[#0F0A04] font-medium text-right flex-1 truncate">{value}</span>
      {badge && (
        <span className={`ml-2 text-xs px-2 py-0.5 rounded-full font-medium shrink-0 ${badge === "Verified" ? "bg-green-50 text-green-700" : "bg-amber-50 text-amber-700"}`}>
          {badge}
        </span>
      )}
    </div>
  );
}
