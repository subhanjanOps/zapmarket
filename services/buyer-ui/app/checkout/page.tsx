"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import Image from "next/image";
import Link from "next/link";
import {
  MapPin,
  CreditCard,
  Wallet,
  Lock,
  AlertCircle,
  Loader2,
  ShoppingBag,
} from "lucide-react";
import { useCartStore } from "@/lib/cart";
import { cn } from "@/lib/utils";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const FIELD_LABELS: Record<string, string> = {
  name: "Full Name",
  phone: "Phone",
  line1: "Address Line 1",
  city: "City",
  pincode: "Pincode",
};

const FIELD_AUTOCOMPLETE: Record<string, string> = {
  name: "name",
  phone: "tel",
  line1: "address-line1",
  city: "address-level2",
  pincode: "postal-code",
};

const STEPS = [
  { id: 1, label: "Bag" },
  { id: 2, label: "Shipping" },
  { id: 3, label: "Payment" },
  { id: 4, label: "Done" },
];

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StepIndicator({ current }: { current: number }) {
  return (
    <div className="flex items-center gap-0 mb-8">
      {STEPS.map((step, idx) => {
        const isActive = step.id === current;
        const isDone = step.id < current;
        return (
          <div key={step.id} className="flex items-center">
            {/* dot + label */}
            <div className="flex flex-col items-center gap-1">
              <div
                className={cn(
                  "w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-all",
                  isDone
                    ? "bg-[#E91E8C] text-white"
                    : isActive
                    ? "bg-[#E91E8C] text-white ring-4 ring-[#E91E8C]/20"
                    : "border-2 border-[#EDE9E3] text-[#B8A898] bg-white"
                )}
              >
                {isDone ? (
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                    <path
                      d="M2 6l3 3 5-5"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                ) : (
                  step.id
                )}
              </div>
              <span
                className={cn(
                  "text-[10px] font-medium whitespace-nowrap",
                  isActive ? "text-[#E91E8C]" : isDone ? "text-[#3D2E1A]" : "text-[#B8A898]"
                )}
              >
                {step.label}
              </span>
            </div>
            {/* connector line */}
            {idx < STEPS.length - 1 && (
              <div
                className={cn(
                  "h-px w-10 sm:w-16 mx-1 mb-4 transition-colors",
                  step.id < current ? "bg-[#E91E8C]" : "bg-[#EDE9E3]"
                )}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

function FormInput({
  id,
  type = "text",
  autoComplete,
  value,
  onChange,
  placeholder,
}: {
  id: string;
  type?: string;
  autoComplete?: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  return (
    <input
      id={id}
      type={type}
      autoComplete={autoComplete}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full h-11 border border-[#EDE9E3] rounded-xl px-4 text-sm text-[#0F0A04] bg-white
        focus:outline-none focus:border-[#E91E8C] transition-colors placeholder:text-[#B8A898]"
    />
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function CheckoutPage() {
  const { items, total, clearCart } = useCartStore();
  const router = useRouter();
  const idempotencyKey = useRef(crypto.randomUUID());
  const [address, setAddress] = useState({
    name: "",
    phone: "",
    line1: "",
    city: "",
    pincode: "",
  });
  const [paymentMethod, setPaymentMethod] = useState<"cod" | "upi">("cod");
  const [promoCode, setPromoCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (items.length === 0) router.replace("/cart");
  }, [items.length, router]);

  async function handlePlaceOrder() {
    const missing = (["name", "phone", "line1", "city", "pincode"] as const).filter(
      (f) => !address[f].trim()
    );
    if (missing.length > 0) {
      setError(`Please fill in: ${missing.map((f) => FIELD_LABELS[f]).join(", ")}`);
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const body = {
        items: items.map((i) => ({ sku_id: i.skuId, quantity: i.qty, unit_price: i.price })),
        notes: `Shipping: ${address.name}, ${address.line1}, ${address.city} ${address.pincode}, Phone: ${address.phone}`,
      };
      const res = await fetch("/api/proxy/v1/orders", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey.current,
        },
        body: JSON.stringify(body),
      });
      if (res.status === 201 || res.status === 409) {
        const data = await res.json();
        const orderId = data.data?.id ?? data.id;
        clearCart();
        router.push(`/account/orders/${orderId}?new=1`);
        return;
      }
      const err = await res.json().catch(() => ({}));
      setError(
        (err as Record<string, string>).error ??
          (err as Record<string, string>).message ??
          "Failed to place order. Please try again."
      );
    } catch {
      setError("Network error. Please check your connection.");
    } finally {
      setLoading(false);
    }
  }

  const subtotal = total() / 100;

  return (
    <div className="min-h-screen bg-[#F9F8F5]">
      <div className="max-w-5xl mx-auto px-4 py-10">
        {/* Page heading */}
        <div className="mb-8">
          <Link
            href="/cart"
            className="inline-flex items-center gap-1.5 text-xs text-[#7A6856] hover:text-[#E91E8C] transition-colors mb-4"
          >
            <ShoppingBag size={14} />
            Back to cart
          </Link>
          <h1 className="text-3xl font-display font-bold text-[#0F0A04] tracking-tight">
            Checkout
          </h1>
        </div>

        {/* Step indicator */}
        <StepIndicator current={2} />

        {/* Error pill */}
        <AnimatePresence>
          {error && (
            <motion.div
              initial={{ opacity: 0, y: -8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              className="mb-6 flex items-center gap-2.5 bg-red-50 border border-red-200 text-red-700
                rounded-full px-4 py-2.5 text-sm font-medium"
            >
              <AlertCircle size={15} className="shrink-0" />
              {error}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Two-column layout */}
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_380px] gap-6">
          {/* ---- LEFT COLUMN ---- */}
          <div className="flex flex-col gap-5">
            {/* Delivery address card */}
            <div className="bg-white rounded-2xl shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)] p-6">
              <div className="flex items-center gap-2 mb-5">
                <MapPin size={18} className="text-[#E91E8C]" />
                <h2 className="text-base font-semibold text-[#0F0A04]">Delivery address</h2>
              </div>

              <div className="flex flex-col gap-4">
                {/* Full Name */}
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="field-name" className="text-xs font-medium text-[#3D2E1A]">
                    Full Name
                  </label>
                  <FormInput
                    id="field-name"
                    autoComplete={FIELD_AUTOCOMPLETE.name}
                    value={address.name}
                    onChange={(v) => setAddress((a) => ({ ...a, name: v }))}
                    placeholder="Jane Doe"
                  />
                </div>

                {/* Phone */}
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="field-phone" className="text-xs font-medium text-[#3D2E1A]">
                    Phone
                  </label>
                  <FormInput
                    id="field-phone"
                    type="tel"
                    autoComplete={FIELD_AUTOCOMPLETE.phone}
                    value={address.phone}
                    onChange={(v) => setAddress((a) => ({ ...a, phone: v }))}
                    placeholder="+91 98765 43210"
                  />
                </div>

                {/* Address Line 1 */}
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="field-line1" className="text-xs font-medium text-[#3D2E1A]">
                    Address Line 1
                  </label>
                  <FormInput
                    id="field-line1"
                    autoComplete={FIELD_AUTOCOMPLETE.line1}
                    value={address.line1}
                    onChange={(v) => setAddress((a) => ({ ...a, line1: v }))}
                    placeholder="House no., Street, Area"
                  />
                </div>

                {/* City + Pincode — 2-col grid */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1.5">
                    <label htmlFor="field-city" className="text-xs font-medium text-[#3D2E1A]">
                      City
                    </label>
                    <FormInput
                      id="field-city"
                      autoComplete={FIELD_AUTOCOMPLETE.city}
                      value={address.city}
                      onChange={(v) => setAddress((a) => ({ ...a, city: v }))}
                      placeholder="Mumbai"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label htmlFor="field-pincode" className="text-xs font-medium text-[#3D2E1A]">
                      Pincode
                    </label>
                    <FormInput
                      id="field-pincode"
                      autoComplete={FIELD_AUTOCOMPLETE.pincode}
                      value={address.pincode}
                      onChange={(v) => setAddress((a) => ({ ...a, pincode: v }))}
                      placeholder="400001"
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* Payment method card */}
            <div className="bg-white rounded-2xl shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)] p-6">
              <div className="flex items-center gap-2 mb-5">
                <CreditCard size={18} className="text-[#E91E8C]" />
                <h2 className="text-base font-semibold text-[#0F0A04]">Payment method</h2>
              </div>

              <div className="flex flex-col gap-3">
                {/* COD option */}
                <button
                  type="button"
                  onClick={() => setPaymentMethod("cod")}
                  className={cn(
                    "border rounded-2xl px-5 py-4 flex items-center gap-3 w-full text-left transition-all",
                    paymentMethod === "cod"
                      ? "border-[#E91E8C] bg-[#FDE8F4]/30"
                      : "border-[#EDE9E3] bg-white hover:border-[#B8A898]"
                  )}
                >
                  <div
                    className={cn(
                      "w-9 h-9 rounded-xl flex items-center justify-center shrink-0",
                      paymentMethod === "cod" ? "bg-[#E91E8C]/10" : "bg-[#F9F8F5]"
                    )}
                  >
                    <Wallet
                      size={18}
                      className={paymentMethod === "cod" ? "text-[#E91E8C]" : "text-[#7A6856]"}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p
                      className={cn(
                        "text-sm font-semibold",
                        paymentMethod === "cod" ? "text-[#E91E8C]" : "text-[#0F0A04]"
                      )}
                    >
                      Cash on Delivery
                    </p>
                    <p className="text-xs text-[#7A6856] mt-0.5">Pay when your order arrives</p>
                  </div>
                  <div
                    className={cn(
                      "w-4 h-4 rounded-full border-2 shrink-0 flex items-center justify-center",
                      paymentMethod === "cod" ? "border-[#E91E8C]" : "border-[#B8A898]"
                    )}
                  >
                    {paymentMethod === "cod" && (
                      <div className="w-2 h-2 rounded-full bg-[#E91E8C]" />
                    )}
                  </div>
                </button>

                {/* UPI option (visual only) */}
                <button
                  type="button"
                  onClick={() => setPaymentMethod("upi")}
                  className={cn(
                    "border rounded-2xl px-5 py-4 flex items-center gap-3 w-full text-left transition-all",
                    paymentMethod === "upi"
                      ? "border-[#E91E8C] bg-[#FDE8F4]/30"
                      : "border-[#EDE9E3] bg-white hover:border-[#B8A898]"
                  )}
                >
                  <div
                    className={cn(
                      "w-9 h-9 rounded-xl flex items-center justify-center shrink-0",
                      paymentMethod === "upi" ? "bg-[#E91E8C]/10" : "bg-[#F9F8F5]"
                    )}
                  >
                    <CreditCard
                      size={18}
                      className={paymentMethod === "upi" ? "text-[#E91E8C]" : "text-[#7A6856]"}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p
                      className={cn(
                        "text-sm font-semibold",
                        paymentMethod === "upi" ? "text-[#E91E8C]" : "text-[#0F0A04]"
                      )}
                    >
                      UPI
                    </p>
                    <p className="text-xs text-[#7A6856] mt-0.5">GPay, PhonePe, Paytm &amp; more</p>
                  </div>
                  <div
                    className={cn(
                      "w-4 h-4 rounded-full border-2 shrink-0 flex items-center justify-center",
                      paymentMethod === "upi" ? "border-[#E91E8C]" : "border-[#B8A898]"
                    )}
                  >
                    {paymentMethod === "upi" && (
                      <div className="w-2 h-2 rounded-full bg-[#E91E8C]" />
                    )}
                  </div>
                </button>
              </div>
            </div>
          </div>

          {/* ---- RIGHT COLUMN: Order Summary ---- */}
          <div className="lg:sticky lg:top-24 h-fit">
            <div className="bg-white rounded-2xl shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)] p-6 flex flex-col gap-5">
              <h2 className="font-display text-lg font-bold text-[#0F0A04]">Order Summary</h2>

              {/* Items list */}
              <div className="flex flex-col gap-3">
                {items.map((item) => (
                  <div key={item.skuId} className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-lg bg-[#F9F8F5] shrink-0 overflow-hidden flex items-center justify-center">
                      {item.image ? (
                        <Image
                          src={item.image}
                          alt={item.name}
                          width={48}
                          height={48}
                          className="object-contain w-full h-full"
                        />
                      ) : (
                        <ShoppingBag size={20} className="text-[#B8A898]" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-[#0F0A04] font-medium truncate leading-snug">
                        {item.name}
                      </p>
                      <p className="text-xs text-[#7A6856] mt-0.5">Qty: {item.qty}</p>
                    </div>
                    <span className="text-sm font-semibold text-[#0F0A04] shrink-0">
                      ₹{((item.price * item.qty) / 100).toFixed(2)}
                    </span>
                  </div>
                ))}
              </div>

              {/* Divider */}
              <div className="h-px bg-[#EDE9E3]" />

              {/* Promo code */}
              <div className="flex gap-2">
                <input
                  type="text"
                  value={promoCode}
                  onChange={(e) => setPromoCode(e.target.value)}
                  placeholder="Promo code"
                  className="flex-1 h-10 border border-[#EDE9E3] rounded-xl px-3 text-sm text-[#0F0A04] bg-white
                    focus:outline-none focus:border-[#E91E8C] transition-colors placeholder:text-[#B8A898]"
                />
                <button
                  type="button"
                  className="h-10 px-4 rounded-xl border border-[#EDE9E3] text-xs font-semibold text-[#3D2E1A]
                    hover:border-[#E91E8C] hover:text-[#E91E8C] transition-colors"
                >
                  Apply
                </button>
              </div>

              {/* Divider */}
              <div className="h-px bg-[#EDE9E3]" />

              {/* Totals */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-[#7A6856]">Subtotal</span>
                  <span className="text-[#3D2E1A] font-medium">₹{subtotal.toFixed(2)}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-[#7A6856]">Delivery</span>
                  <span className="text-emerald-600 font-semibold">Free</span>
                </div>
              </div>

              {/* Divider */}
              <div className="h-px bg-[#EDE9E3]" />

              {/* Total */}
              <div className="flex items-center justify-between">
                <span className="font-display font-bold text-[#0F0A04] text-base">Total</span>
                <span className="font-display font-bold text-[#0F0A04] text-xl">
                  ₹{subtotal.toFixed(2)}
                </span>
              </div>

              {/* Place Order button */}
              <button
                type="button"
                onClick={handlePlaceOrder}
                disabled={loading}
                className="w-full h-12 bg-[#E91E8C] hover:bg-[#B5166E] disabled:opacity-60 disabled:cursor-not-allowed
                  text-white font-bold rounded-2xl transition-colors
                  shadow-[0_4px_20px_rgba(233,30,140,0.25)] flex items-center justify-center gap-2"
              >
                {loading ? (
                  <>
                    <Loader2 size={16} className="animate-spin" />
                    Placing Order…
                  </>
                ) : (
                  "Place Order"
                )}
              </button>

              {/* SSL badge */}
              <div className="flex items-center justify-center gap-1.5">
                <Lock size={11} className="text-[#B8A898]" />
                <span className="text-xs text-[#B8A898]">Secure 256-bit SSL encrypted checkout</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
