import type { Metadata } from "next";
import Link from "next/link";
import { Clock, CheckCircle2, Mail, ShoppingBag } from "lucide-react";

export const metadata: Metadata = { title: "Application Submitted" };

const TIMELINE = [
  { label: "Application submitted",    done: true  },
  { label: "Verify your email",        done: false },
  { label: "Admin review (up to 48h)", done: false },
  { label: "Start selling!",           done: false },
];

export default function SellerPendingPage() {
  return (
    <div className="min-h-[calc(100vh-128px)] flex items-center justify-center p-6 bg-[#F9F8F5]">
      <div className="w-full max-w-sm text-center space-y-7">
        <div className="inline-flex h-16 w-16 bg-amber-50 border border-amber-100 rounded-2xl items-center justify-center mx-auto">
          <Clock className="h-8 w-8 text-amber-500" />
        </div>

        <div>
          <h1 className="font-display font-bold text-2xl text-[#0F0A04]">Application submitted!</h1>
          <p className="text-[#7A6856] mt-2 text-sm leading-relaxed">
            Our team will review your application within 48 hours.
            You&apos;ll receive an email at the address you provided once you&apos;re approved.
          </p>
        </div>

        <ul className="text-left space-y-3.5 bg-white border border-[#EDE9E3] rounded-2xl p-5">
          {TIMELINE.map(({ label, done }) => (
            <li key={label} className="flex items-center gap-3">
              <CheckCircle2 className={`h-5 w-5 shrink-0 ${done ? "text-green-500" : "text-[#D9CFC4]"}`} />
              <span className={`text-sm ${done ? "text-[#0F0A04] font-medium" : "text-[#B8A898]"}`}>{label}</span>
            </li>
          ))}
        </ul>

        <div className="bg-blue-50 border border-blue-100 rounded-xl p-4 flex items-start gap-3 text-left">
          <Mail className="h-4 w-4 text-blue-500 shrink-0 mt-0.5" />
          <p className="text-xs text-blue-700 leading-relaxed">
            <span className="font-semibold">Check your inbox.</span> Verifying your email speeds up the approval process — look for a message from ZapMarket.
          </p>
        </div>

        <div className="space-y-3">
          <Link
            href="/products"
            className="flex items-center justify-center gap-2 w-full h-11 bg-[#4F46E5] hover:bg-[#3730A3] text-white font-bold rounded-xl transition-colors shadow-[0_2px_12px_rgba(79,70,229,0.25)]"
          >
            <ShoppingBag className="h-4 w-4" />
            Browse the marketplace
          </Link>
          <Link href="/login" className="block text-sm text-[#7A6856] hover:text-[#0F0A04] transition-colors">
            Sign in to check your application status →
          </Link>
        </div>
      </div>
    </div>
  );
}
