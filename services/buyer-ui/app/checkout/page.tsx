"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useCartStore } from "@/lib/cart";

export default function CheckoutPage() {
  const { items, total, clearCart } = useCartStore();
  const router = useRouter();
  const idempotencyKey = useRef(crypto.randomUUID());
  const [address, setAddress] = useState({ name: "", phone: "", line1: "", city: "", pincode: "" });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (items.length === 0) router.replace("/cart");
  }, [items.length, router]);

  async function handlePlaceOrder() {
    const missing = (["name", "phone", "line1", "city", "pincode"] as const).filter((f) => !address[f].trim());
    if (missing.length > 0) {
      setError(`Please fill in: ${missing.join(", ")}`);
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
        headers: { "Content-Type": "application/json", "Idempotency-Key": idempotencyKey.current },
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
      setError((err as Record<string, string>).error ?? (err as Record<string, string>).message ?? "Failed to place order. Please try again.");
    } catch {
      setError("Network error. Please check your connection.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-8">Checkout</h1>
      <div className="flex flex-col lg:flex-row gap-8">
        <div className="flex-1 rounded-2xl p-6" style={{ background: "#fff", border: "1px solid #F0EDE8" }}>
          <h2 className="font-bold mb-5" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>Shipping Address</h2>
          <div className="space-y-3">
            {(["name", "phone", "line1", "city", "pincode"] as const).map((field) => (
              <div key={field}>
                <label htmlFor={`field-${field}`} className="block text-xs font-semibold mb-1 uppercase tracking-wide" style={{ color: "#6B6052" }}>
                  {field === "line1" ? "Address Line 1" : field.charAt(0).toUpperCase() + field.slice(1)}
                </label>
                <input
                  id={`field-${field}`}
                  type={field === "phone" ? "tel" : "text"}
                  autoComplete={field === "name" ? "name" : field === "phone" ? "tel" : field === "line1" ? "address-line1" : field === "city" ? "address-level2" : "postal-code"}
                  value={address[field]}
                  onChange={(e) => setAddress((a) => ({ ...a, [field]: e.target.value }))}
                  className="w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-colors"
                  style={{ border: "2px solid #F0EDE8", background: "#FFFCF5", color: "#1A1208" }}
                  onFocus={(e) => { e.target.style.borderColor = "#FF2D78"; }}
                  onBlur={(e) => { e.target.style.borderColor = "#F0EDE8"; }}
                />
              </div>
            ))}
          </div>
        </div>

        <div className="lg:w-72">
          <div className="rounded-2xl p-6" style={{ background: "#fff", border: "1px solid #F0EDE8" }}>
            <h2 className="font-bold mb-4" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>Order Review</h2>
            {items.map((i) => (
              <div key={i.skuId} className="flex justify-between text-sm mb-2">
                <span className="text-gray-700 truncate max-w-[160px]">{i.name} ×{i.qty}</span>
                <span>INR {((i.price * i.qty) / 100).toFixed(2)}</span>
              </div>
            ))}
            <div className="border-t mt-3 pt-3 flex justify-between font-bold">
              <span>Total</span>
              <span>INR {(total() / 100).toFixed(2)}</span>
            </div>
            {error && <p className="mt-3 text-red-500 text-sm">{error}</p>}
            <button
              onClick={handlePlaceOrder}
              disabled={loading}
              className="mt-4 w-full font-bold py-3 rounded-xl text-white transition-opacity hover:opacity-90 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed"
              style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
            >
              {loading ? "Placing Order…" : "Place Order"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
