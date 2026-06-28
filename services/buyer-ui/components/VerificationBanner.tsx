"use client";

import { useState } from "react";
import { Mail, X, Loader2 } from "lucide-react";

interface Props { email: string; }

export default function VerificationBanner({ email }: Props) {
  const [dismissed, setDismissed] = useState(false);
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  if (dismissed) return null;

  async function resend() {
    setLoading(true);
    try {
      await fetch("/api/auth/otp/send", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });
      setSent(true);
    } catch {
      // Silently ignore — user can try again
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="bg-amber-50 border-b border-amber-200 px-4 py-2.5 flex items-center gap-3">
      <Mail className="h-4 w-4 text-amber-600 shrink-0" />
      <p className="flex-1 text-sm text-amber-800 min-w-0">
        Verify your email <span className="font-semibold truncate">{email}</span> to unlock all features.{" "}
        {sent ? (
          <span className="text-green-700 font-medium">Email sent!</span>
        ) : (
          <button
            onClick={resend}
            disabled={loading}
            className="underline font-medium hover:text-amber-900 disabled:opacity-60 inline-flex items-center gap-1"
          >
            {loading && <Loader2 className="h-3 w-3 animate-spin" />}
            {loading ? "Sending…" : "Resend verification"}
          </button>
        )}
      </p>
      <button
        onClick={() => setDismissed(true)}
        aria-label="Dismiss"
        className="p-0.5 text-amber-400 hover:text-amber-700 transition-colors shrink-0"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
