"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useCartStore } from "@/lib/cart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

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
    <div className="max-w-5xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-8">Checkout</h1>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left: form sections */}
        <div className="lg:col-span-2 flex flex-col gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Shipping Address</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {(["name", "phone", "line1", "city", "pincode"] as const).map((field) => (
                <div key={field} className="space-y-1.5">
                  <Label htmlFor={`field-${field}`}>{FIELD_LABELS[field]}</Label>
                  <Input
                    id={`field-${field}`}
                    type={field === "phone" ? "tel" : "text"}
                    autoComplete={FIELD_AUTOCOMPLETE[field]}
                    value={address[field]}
                    onChange={(e) => setAddress((a) => ({ ...a, [field]: e.target.value }))}
                    placeholder={FIELD_LABELS[field]}
                  />
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Payment</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-3 rounded-md border px-4 py-3 bg-muted/40">
                <input
                  type="radio"
                  id="payment-cod"
                  name="payment"
                  defaultChecked
                  className="accent-primary h-4 w-4"
                />
                <Label htmlFor="payment-cod" className="cursor-pointer font-medium">
                  Cash on Delivery
                </Label>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right: order summary */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader>
              <CardTitle>Order Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {items.map((i) => (
                <div key={i.skuId} className="flex justify-between text-sm">
                  <span className="text-muted-foreground truncate max-w-[160px]">
                    {i.name} &times;{i.qty}
                  </span>
                  <span className="font-medium">INR {((i.price * i.qty) / 100).toFixed(2)}</span>
                </div>
              ))}

              <div className="flex justify-between text-sm text-muted-foreground">
                <span>Subtotal</span>
                <span>INR {(total() / 100).toFixed(2)}</span>
              </div>

              <Separator />

              <div className="flex justify-between font-bold text-base">
                <span>Total</span>
                <span>INR {(total() / 100).toFixed(2)}</span>
              </div>

              {error && <p className="text-destructive text-sm">{error}</p>}

              <Button
                size="lg"
                className="w-full bg-primary text-primary-foreground"
                onClick={handlePlaceOrder}
                disabled={loading}
              >
                {loading ? "Placing Order…" : "Place Order"}
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
