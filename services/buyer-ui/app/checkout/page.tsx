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
        <div className="flex-1 bg-white border border-gray-200 rounded-lg p-6">
          <h2 className="font-semibold mb-4">Shipping Address</h2>
          <div className="space-y-3">
            {(["name", "phone", "line1", "city", "pincode"] as const).map((field) => (
              <input key={field} type="text"
                placeholder={field === "line1" ? "Address Line 1" : field.charAt(0).toUpperCase() + field.slice(1)}
                value={address[field]}
                onChange={(e) => setAddress((a) => ({ ...a, [field]: e.target.value }))}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm outline-none focus:border-[#FF9900]" />
            ))}
          </div>
        </div>

        <div className="lg:w-72">
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h2 className="font-semibold mb-4">Order Review</h2>
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
            <button onClick={handlePlaceOrder} disabled={loading}
              className="mt-4 w-full bg-[#FF9900] text-black font-semibold py-3 rounded hover:bg-[#e68900] disabled:opacity-50">
              {loading ? "Placing Order…" : "Place Order"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
