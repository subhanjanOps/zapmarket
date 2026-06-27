"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import Image from "next/image";
import Link from "next/link";
import {
  Wallet,
  CreditCard,
  Lock,
  AlertCircle,
  Loader2,
  ShoppingBag,
} from "lucide-react";
import {
  Elements,
  CardElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";
import { useCartStore, CartItem } from "@/lib/cart";
import { getStripe } from "@/lib/stripe";
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
  { id: 1, label: "Address" },
  { id: 2, label: "Payment" },
  { id: 3, label: "Done" },
];

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StepIndicator({ current }: { current: number }) {
  return (
    <div className="flex items-start gap-0 mb-8">
      {STEPS.map((step, idx) => {
        const isActive = step.id === current;
        const isDone = step.id < current;
        return (
          <div key={step.id} className="flex items-center">
            <div className="flex flex-col items-center">
              <div
                className={cn(
                  "w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-all",
                  isDone
                    ? "bg-[#111111] text-white"
                    : isActive
                    ? "bg-[#E91E8C] text-white"
                    : "bg-[#F0F0F0] text-[#999999]"
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
                  "text-xs mt-1 whitespace-nowrap",
                  isActive || isDone ? "text-[#555555]" : "text-[#999999]"
                )}
              >
                {step.label}
              </span>
            </div>
            {idx < STEPS.length - 1 && (
              <div
                className={cn(
                  "h-px w-12 sm:w-20 mx-1 mb-4 transition-colors",
                  step.id < current ? "bg-[#111111]" : "bg-[#E8E8E8]"
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
  required = false,
}: {
  id: string;
  type?: string;
  autoComplete?: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  required?: boolean;
}) {
  return (
    <input
      id={id}
      type={type}
      autoComplete={autoComplete}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      required={required}
      aria-required={required ? "true" : undefined}
      className="w-full h-10 border border-[#E8E8E8] rounded-md px-3 text-sm text-[#111111] bg-white
        focus:outline-none focus:border-[#111111] transition-colors placeholder:text-[#999999]"
    />
  );
}

// ---------------------------------------------------------------------------
// Inner form (needs Stripe context via Elements wrapper)
// ---------------------------------------------------------------------------

type PaymentMethodType = "cod" | "card" | "upi";

interface CheckoutFormProps {
  address: { name: string; phone: string; line1: string; city: string; pincode: string };
  setAddress: React.Dispatch<React.SetStateAction<{ name: string; phone: string; line1: string; city: string; pincode: string }>>;
  paymentMethod: PaymentMethodType;
  setPaymentMethod: (m: PaymentMethodType) => void;
  promoCode: string;
  setPromoCode: (v: string) => void;
  promoMsg: string | null;
  setPromoMsg: (v: string | null) => void;
  error: string | null;
  setError: (v: string | null) => void;
  loading: boolean;
  setLoading: (v: boolean) => void;
  items: CartItem[];
  subtotal: number;
  clearCart: () => void;
  getIdempotencyKey: () => string;
  resetIdempotencyKey: () => void;
}

function CheckoutForm({
  address, setAddress, paymentMethod, setPaymentMethod,
  promoCode, setPromoCode, promoMsg, setPromoMsg,
  error, setError, loading, setLoading,
  items, subtotal, clearCart, getIdempotencyKey, resetIdempotencyKey,
}: CheckoutFormProps) {
  const router = useRouter();
  const stripe = useStripe();
  const elements = useElements();

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
      let paymentMethodId = "";

      if (paymentMethod === "card") {
        if (!stripe || !elements) {
          setError("Stripe is not loaded yet. Please wait a moment and try again.");
          setLoading(false);
          return;
        }
        const cardElement = elements.getElement(CardElement);
        if (!cardElement) {
          setError("Card input not found. Please refresh the page.");
          setLoading(false);
          return;
        }
        const { paymentMethod: pm, error: stripeError } = await stripe.createPaymentMethod({
          type: "card",
          card: cardElement,
          billing_details: { name: address.name, phone: address.phone },
        });
        if (stripeError) {
          setError(stripeError.message ?? "Card error. Please check your card details.");
          setLoading(false);
          return;
        }
        paymentMethodId = pm!.id;
      }

      const idemKey = getIdempotencyKey();
      const body = {
        idempotency_key: idemKey,
        items: items.map((i) => ({ sku_id: i.skuId, quantity: i.qty, unit_price: i.price })),
        payment_method: paymentMethod,
        payment_method_id: paymentMethodId,
        shipping_address: {
          full_name: address.name,
          phone: address.phone,
          address_line1: address.line1,
          city: address.city,
          pincode: address.pincode,
          country: "IN",
        },
      };

      const res = await fetch("/api/proxy/v1/orders", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idemKey,
        },
        body: JSON.stringify(body),
      });

      if (res.status === 201 || res.status === 409) {
        const data = await res.json();
        const orderId = data.data?.id ?? data.id;
        sessionStorage.removeItem("checkout_idempotency_key");
        clearCart();
        router.push(`/account/orders/${orderId}?new=1`);
        return;
      }

      const err = await res.json().catch(() => ({}));
      resetIdempotencyKey();
      setError(
        (err as Record<string, string>).error ??
          (err as Record<string, string>).message ??
          "Failed to place order. Please try again."
      );
    } catch {
      resetIdempotencyKey();
      setError("Network error. Please check your connection.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-8">
      {/* ---- LEFT COLUMN ---- */}
      <div className="flex flex-col gap-8">
        {/* Delivery address section */}
        <div>
          <p className="text-sm font-semibold text-[#111111] uppercase tracking-wider mb-4 pb-3 border-b border-[#E8E8E8]">
            Delivery Address
          </p>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="field-name" className="text-xs font-medium text-[#555555]">Full Name</label>
              <FormInput id="field-name" autoComplete={FIELD_AUTOCOMPLETE.name} value={address.name} onChange={(v) => setAddress((a) => ({ ...a, name: v }))} placeholder="Jane Doe" required />
            </div>
            <div className="flex flex-col gap-1.5">
              <label htmlFor="field-phone" className="text-xs font-medium text-[#555555]">Phone</label>
              <FormInput id="field-phone" type="tel" autoComplete={FIELD_AUTOCOMPLETE.phone} value={address.phone} onChange={(v) => setAddress((a) => ({ ...a, phone: v }))} placeholder="+91 98765 43210" required />
            </div>
            <div className="flex flex-col gap-1.5">
              <label htmlFor="field-line1" className="text-xs font-medium text-[#555555]">Address Line 1</label>
              <FormInput id="field-line1" autoComplete={FIELD_AUTOCOMPLETE.line1} value={address.line1} onChange={(v) => setAddress((a) => ({ ...a, line1: v }))} placeholder="House no., Street, Area" required />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="flex flex-col gap-1.5">
                <label htmlFor="field-city" className="text-xs font-medium text-[#555555]">City</label>
                <FormInput id="field-city" autoComplete={FIELD_AUTOCOMPLETE.city} value={address.city} onChange={(v) => setAddress((a) => ({ ...a, city: v }))} placeholder="Mumbai" required />
              </div>
              <div className="flex flex-col gap-1.5">
                <label htmlFor="field-pincode" className="text-xs font-medium text-[#555555]">Pincode</label>
                <FormInput id="field-pincode" autoComplete={FIELD_AUTOCOMPLETE.pincode} value={address.pincode} onChange={(v) => setAddress((a) => ({ ...a, pincode: v }))} placeholder="400001" required />
              </div>
            </div>
          </div>
        </div>

        {/* Payment method section */}
        <div>
          <p className="text-sm font-semibold text-[#111111] uppercase tracking-wider mb-4 pb-3 border-b border-[#E8E8E8]">
            Payment Method
          </p>
          <div className="flex flex-col gap-3">
            {/* Card option */}
            <button
              type="button"
              onClick={() => setPaymentMethod("card")}
              className={cn(
                "border rounded-md px-4 py-3 flex items-center gap-3 w-full text-left transition-all",
                paymentMethod === "card"
                  ? "border-[#111111] bg-[#F6F6F6]"
                  : "border-[#E8E8E8] bg-white hover:border-[#D0D0D0]"
              )}
            >
              <CreditCard size={17} className={paymentMethod === "card" ? "text-[#111111]" : "text-[#999999]"} />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-[#111111]">Credit / Debit Card</p>
                <p className="text-xs text-[#999999] mt-0.5">Visa, Mastercard, RuPay</p>
              </div>
              <div className={cn("w-4 h-4 rounded-full border-2 shrink-0 flex items-center justify-center", paymentMethod === "card" ? "border-[#111111]" : "border-[#D0D0D0]")}>
                {paymentMethod === "card" && <div className="w-2 h-2 rounded-full bg-[#111111]" />}
              </div>
            </button>

            {/* Stripe CardElement — shown only when card is selected */}
            {paymentMethod === "card" && (
              <div className="border border-[#E8E8E8] rounded-md px-4 py-3 bg-white">
                <CardElement
                  options={{
                    style: {
                      base: {
                        fontSize: "14px",
                        color: "#111111",
                        fontFamily: "inherit",
                        "::placeholder": { color: "#999999" },
                      },
                      invalid: { color: "#E91E8C" },
                    },
                  }}
                />
              </div>
            )}

            {/* COD option */}
            <button
              type="button"
              onClick={() => setPaymentMethod("cod")}
              className={cn(
                "border rounded-md px-4 py-3 flex items-center gap-3 w-full text-left transition-all",
                paymentMethod === "cod"
                  ? "border-[#111111] bg-[#F6F6F6]"
                  : "border-[#E8E8E8] bg-white hover:border-[#D0D0D0]"
              )}
            >
              <Wallet size={17} className={paymentMethod === "cod" ? "text-[#111111]" : "text-[#999999]"} />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-[#111111]">Cash on Delivery</p>
                <p className="text-xs text-[#999999] mt-0.5">Pay when your order arrives</p>
              </div>
              <div className={cn("w-4 h-4 rounded-full border-2 shrink-0 flex items-center justify-center", paymentMethod === "cod" ? "border-[#111111]" : "border-[#D0D0D0]")}>
                {paymentMethod === "cod" && <div className="w-2 h-2 rounded-full bg-[#111111]" />}
              </div>
            </button>

            {/* UPI (coming soon) */}
            <div className="border border-[#E8E8E8] rounded-md px-4 py-3 flex items-center gap-3 w-full opacity-50 cursor-not-allowed">
              <CreditCard size={17} className="text-[#999999]" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-[#111111] flex items-center gap-2">
                  UPI
                  <span className="text-[9px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded-full bg-[#F0F0F0] text-[#999999]">Soon</span>
                </p>
                <p className="text-xs text-[#999999] mt-0.5">GPay, PhonePe, Paytm &amp; more</p>
              </div>
              <div className="w-4 h-4 rounded-full border-2 shrink-0 border-[#D0D0D0]" />
            </div>
          </div>
        </div>
      </div>

      {/* ---- RIGHT COLUMN: Order Summary ---- */}
      <div className="lg:sticky lg:top-24 h-fit">
        <div className="bg-[#F6F6F6] rounded-lg p-5 border border-[#E8E8E8] flex flex-col gap-4">
          <p className="text-sm font-semibold text-[#111111] uppercase tracking-wider">Order Summary</p>

          <div className="flex flex-col gap-3">
            {items.map((item) => (
              <div key={item.skuId} className="flex items-center gap-3">
                <div className="w-11 h-11 rounded-md bg-white border border-[#E8E8E8] shrink-0 overflow-hidden flex items-center justify-center">
                  {item.image ? (
                    <Image src={item.image} alt={item.name} width={44} height={44} className="object-contain w-full h-full" />
                  ) : (
                    <ShoppingBag size={18} className="text-[#D0D0D0]" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-[#111111] font-medium truncate leading-snug">{item.name}</p>
                  <p className="text-xs text-[#999999] mt-0.5">Qty: {item.qty}</p>
                </div>
                <span className="text-sm font-bold text-[#111111] shrink-0 tabular-nums">
                  ₹{((item.price * item.qty) / 100).toFixed(2)}
                </span>
              </div>
            ))}
          </div>

          <div className="h-px bg-[#E8E8E8]" />

          <div>
            <div className="flex gap-2">
              <input type="text" value={promoCode} onChange={(e) => setPromoCode(e.target.value)} placeholder="Promo code"
                className="flex-1 h-10 border border-[#E8E8E8] rounded-md px-3 text-sm text-[#111111] bg-white focus:outline-none focus:border-[#111111] transition-colors placeholder:text-[#999999]" />
              <button type="button" onClick={() => setPromoMsg("Promo codes coming soon")}
                className="h-10 px-4 rounded-md border border-[#E8E8E8] text-xs font-medium text-[#111111] hover:bg-white transition-colors bg-white">
                Apply
              </button>
            </div>
            {promoMsg && <p className="mt-1.5 text-xs text-[#999999]">{promoMsg}</p>}
          </div>

          <div className="h-px bg-[#E8E8E8]" />

          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-[#555555]">Subtotal</span>
              <span className="text-[#111111] font-medium tabular-nums">₹{subtotal.toFixed(2)}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-[#555555]">Delivery</span>
              <span className="text-[#16A34A] font-semibold">Free</span>
            </div>
          </div>

          <div className="h-px bg-[#E8E8E8]" />

          <div className="flex items-center justify-between">
            <span className="text-sm font-semibold text-[#111111]">Total</span>
            <span className="text-lg font-bold text-[#111111] tabular-nums">₹{subtotal.toFixed(2)}</span>
          </div>

          <motion.button
            type="button"
            onClick={handlePlaceOrder}
            disabled={loading}
            whileTap={{ scale: 0.97 }}
            className="w-full h-11 bg-[#E91E8C] hover:bg-[#C2187A] disabled:opacity-60 disabled:cursor-not-allowed
              text-white font-medium rounded-md text-sm transition-colors flex items-center justify-center gap-2"
          >
            {loading ? (
              <><Loader2 size={15} className="animate-spin" />Placing Order...</>
            ) : (
              "Place Order"
            )}
          </motion.button>

          <div className="flex items-center justify-center gap-1.5 mt-1">
            <Lock size={11} className="text-[#999999]" />
            <span className="text-xs text-[#999999]">Secure 256-bit encrypted checkout</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page — wraps form in Stripe Elements provider
// ---------------------------------------------------------------------------

export default function CheckoutPage() {
  const { items, total, clearCart } = useCartStore();
  const router = useRouter();
  const idempotencyKey = useRef<string>("");
  const getIdempotencyKey = useCallback(() => {
    if (idempotencyKey.current) return idempotencyKey.current;
    const stored = sessionStorage.getItem("checkout_idempotency_key");
    if (stored) { idempotencyKey.current = stored; return stored; }
    const fresh = crypto.randomUUID();
    sessionStorage.setItem("checkout_idempotency_key", fresh);
    idempotencyKey.current = fresh;
    return fresh;
  }, []);
  const resetIdempotencyKey = useCallback(() => {
    sessionStorage.removeItem("checkout_idempotency_key");
    idempotencyKey.current = "";
  }, []);

  const [address, setAddress] = useState({ name: "", phone: "", line1: "", city: "", pincode: "" });
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethodType>("card");
  const [promoCode, setPromoCode] = useState("");
  const [promoMsg, setPromoMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (items.length === 0) router.replace("/cart");
  }, [items.length, router]);

  const subtotal = total() / 100;
  const stripePromise = getStripe();

  const formProps = {
    address, setAddress, paymentMethod, setPaymentMethod,
    promoCode, setPromoCode, promoMsg, setPromoMsg,
    error, setError, loading, setLoading,
    items, subtotal, clearCart, getIdempotencyKey, resetIdempotencyKey,
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: [0.25, 0.1, 0.25, 1] }}
      className="min-h-screen bg-white"
    >
      <div className="max-w-4xl mx-auto px-4 py-10">
        <Link href="/cart" className="inline-block text-sm text-[#555555] hover:text-[#111111] transition-colors mb-6">
          &larr; Back to cart
        </Link>
        <h1 className="text-2xl font-bold text-[#111111] tracking-tight mb-6">Checkout</h1>
        <StepIndicator current={1} />

        <AnimatePresence>
          {error && (
            <motion.div
              initial={{ opacity: 0, y: -8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.2 }}
              className="mb-6 flex items-center gap-2.5 bg-red-50 border border-red-200 text-red-700 rounded-md px-4 py-3 text-sm"
            >
              <AlertCircle size={15} className="shrink-0" />
              {error}
            </motion.div>
          )}
        </AnimatePresence>

        <Elements stripe={stripePromise}>
          <CheckoutForm {...formProps} />
        </Elements>
      </div>
    </motion.div>
  );
}
