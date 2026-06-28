import type { Metadata } from "next";
import Link from "next/link";
import { Zap, ShoppingBag, Store, ArrowRight } from "lucide-react";

export const metadata: Metadata = { title: "Create Account" };

export default function RegisterPage() {
  return (
    <div className="min-h-[calc(100vh-128px)] flex items-center justify-center p-6 bg-[#F9F8F5]">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <div className="inline-flex h-11 w-11 bg-[#E91E8C] rounded-xl items-center justify-center mb-5 shadow-[0_2px_12px_rgba(233,30,140,0.35)]">
            <Zap className="h-6 w-6 text-white" strokeWidth={2.5} />
          </div>
          <h1 className="font-display font-bold text-2xl text-[#0F0A04]">Join ZapMarket</h1>
          <p className="text-[#7A6856] text-sm mt-1.5">How would you like to get started?</p>
        </div>

        <div className="grid gap-4">
          <Link
            href="/register/buyer"
            className="group flex items-center gap-5 p-5 rounded-2xl border-2 border-[#EDE9E3] bg-white hover:border-[#E91E8C] hover:shadow-[0_0_0_4px_rgba(233,30,140,0.06)] transition-all"
          >
            <div className="h-12 w-12 bg-[#FFF0F8] rounded-xl flex items-center justify-center shrink-0 group-hover:bg-[#FCE4F3] transition-colors">
              <ShoppingBag className="h-6 w-6 text-[#E91E8C]" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-semibold text-[#0F0A04]">Shop as a Buyer</p>
              <p className="text-sm text-[#7A6856] mt-0.5">Discover deals, track orders, earn rewards.</p>
            </div>
            <ArrowRight className="h-4 w-4 text-[#C4B9AA] group-hover:text-[#E91E8C] transition-colors shrink-0" />
          </Link>

          <Link
            href="/register/seller"
            className="group flex items-center gap-5 p-5 rounded-2xl border-2 border-[#EDE9E3] bg-white hover:border-[#4F46E5] hover:shadow-[0_0_0_4px_rgba(79,70,229,0.06)] transition-all"
          >
            <div className="h-12 w-12 bg-[#F0F0FF] rounded-xl flex items-center justify-center shrink-0 group-hover:bg-[#E4E3FC] transition-colors">
              <Store className="h-6 w-6 text-[#4F46E5]" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-semibold text-[#0F0A04]">Sell on ZapMarket</p>
              <p className="text-sm text-[#7A6856] mt-0.5">List products, manage orders, grow your business.</p>
            </div>
            <ArrowRight className="h-4 w-4 text-[#C4B9AA] group-hover:text-[#4F46E5] transition-colors shrink-0" />
          </Link>
        </div>

        <p className="text-sm text-center text-[#7A6856]">
          Already have an account?{" "}
          <Link href="/login" className="font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
