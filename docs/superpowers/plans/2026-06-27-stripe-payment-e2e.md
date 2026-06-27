# Stripe Payment E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire Stripe card payments end-to-end: Stripe Elements card input in buyer-ui checkout → `pm_xxx` token → order request → order-management-service → payment-service gRPC → StripeGateway charges card → order confirmed.

**Architecture:** The buyer-ui checkout page gains a Stripe Elements `CardElement` on Step 2 (Payment). On submit, `stripe.createPaymentMethod()` is called client-side to tokenize the card into a `pm_xxx` string, which is sent as `payment_method_id` in the order POST body. The order-management-service passes it through its `paymentGateway` interface and gRPC client to payment-service, which already has `StripeGateway` implemented. COD skips the card UI and sends no `payment_method_id`.

**Tech Stack:** Next.js 14 (App Router), `@stripe/stripe-js`, `@stripe/react-stripe-js`, Go 1.22, `stripe-go/v82`, existing gRPC proto with `payment_method_id` field already added.

## Global Constraints

- No ORM — repositories use `database/sql` directly
- `payment_method_id` field already exists in proto as field 6 on `ChargeCardRequest` — do not regenerate proto
- buyer-ui uses `fetch("/api/proxy/v1/orders", ...)` — no direct backend calls
- COD orders must work without any card input — `payment_method_id` stays empty string for COD
- Stripe publishable key is read from env var `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` in buyer-ui
- `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` already wired in payment-service `main.go`; only docker-compose env block needs the vars added
- Do not change the proto or regenerate `*.pb.go`

---

## File Map

| File | Action | What changes |
|------|--------|--------------|
| `services/buyer-ui/app/checkout/page.tsx` | Modify | Add Stripe Elements card input on Payment step; collect `pm_xxx` before submitting |
| `services/buyer-ui/.env.local` (or `.env.example`) | Modify | Add `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` |
| `services/order-management-service/internal/service/order_service.go` | Modify | `paymentGateway` interface + `Checkout` + `finalisePayment` accept `paymentMethodID string` |
| `services/order-management-service/internal/service/order_service_test.go` | Modify | Update mock `ChargeCard` signature and test calls |
| `services/order-management-service/internal/clients/payment_client.go` | Modify | `ChargeCard` passes `PaymentMethodId` in gRPC request |
| `services/order-management-service/internal/handler/http/order_handler.go` | Modify | `checkoutRequest` gains `PaymentMethodID` field; pass to `svc.Checkout` |
| `docker-compose.yml` | Modify | Add `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` to payment-service environment |

---

### Task 1: Pass `payment_method_id` through order-management-service

**Files:**
- Modify: `services/order-management-service/internal/service/order_service.go`
- Modify: `services/order-management-service/internal/clients/payment_client.go`
- Modify: `services/order-management-service/internal/handler/http/order_handler.go`
- Modify: `services/order-management-service/internal/service/order_service_test.go`

**Interfaces:**
- Produces: `OrderService.Checkout(ctx, userID, idempotencyKey, items []CheckoutItem, currency, paymentMethodID string) (*domain.Order, error)`
- Produces: `PaymentClient.ChargeCard(ctx, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, string, error)`

- [ ] **Step 1: Update the `paymentGateway` interface in order_service.go**

In `services/order-management-service/internal/service/order_service.go`, change the `paymentGateway` interface (currently line ~27):

```go
// paymentGateway is the subset of clients.PaymentClient the saga needs.
type paymentGateway interface {
	ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, string, error)
}
```

- [ ] **Step 2: Update `OrderService.Checkout` signature and `CheckoutItem`**

In `order_service.go`, update the interface and struct:

```go
// OrderService defines the public interface for order operations.
type OrderService interface {
	Checkout(ctx context.Context, userID, idempotencyKey uuid.UUID, items []CheckoutItem, currency, paymentMethodID string) (*domain.Order, error)
	// ... rest unchanged
}

// CheckoutItem is the per-SKU input to Checkout.
type CheckoutItem struct {
	SKUID     uuid.UUID
	SellerID  *uuid.UUID
	Quantity  int
	UnitPrice int64
}
```

- [ ] **Step 3: Update `orderService.Checkout` implementation**

Find the `func (s *orderService) Checkout(` implementation. Change its signature and pass `paymentMethodID` down to `finalisePayment`:

```go
func (s *orderService) Checkout(ctx context.Context, userID, idempotencyKey uuid.UUID, items []CheckoutItem, currency, paymentMethodID string) (*domain.Order, error) {
```

Find the call to `finalisePayment` inside `Checkout` and update it to pass `paymentMethodID`:

```go
paymentID, err := s.finalisePayment(ctx, order, orderItems, userID, currency, idempotencyKey, paymentMethodID)
```

- [ ] **Step 4: Update `finalisePayment` to accept and forward `paymentMethodID`**

```go
func (s *orderService) finalisePayment(ctx context.Context, order *domain.Order, items []*domain.OrderItem, userID uuid.UUID, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, error) {
	paymentID, paymentStatus, err := s.payment.ChargeCard(ctx, order.ID, userID, order.TotalAmount, currency, idempotencyKey, paymentMethodID)
	// rest of function unchanged
```

- [ ] **Step 5: Update `PaymentClient.ChargeCard` to send `PaymentMethodId`**

In `services/order-management-service/internal/clients/payment_client.go`:

```go
func (c *PaymentClient) ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, string, error) {
	resp, err := c.client.ChargeCard(ctx, &pb.ChargeCardRequest{
		OrderId:         orderID.String(),
		UserId:          userID.String(),
		Amount:          amount,
		Currency:        currency,
		IdempotencyKey:  idempotencyKey.String(),
		PaymentMethodId: paymentMethodID,
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("payment ChargeCard: %w", err)
	}
	paymentID, err := uuid.Parse(resp.PaymentId)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("payment returned invalid payment_id %q: %w", resp.PaymentId, err)
	}
	return paymentID, resp.Status, nil
}
```

- [ ] **Step 6: Update the HTTP handler to accept and pass `payment_method_id`**

In `services/order-management-service/internal/handler/http/order_handler.go`, update `checkoutRequest`:

```go
type checkoutRequest struct {
	Items           []checkoutItemRequest `json:"items"`
	IdempotencyKey  string                `json:"idempotency_key"`
	Currency        string                `json:"currency,omitempty"`
	PaymentMethodID string                `json:"payment_method_id,omitempty"`
}
```

Find the `svc.Checkout` call (currently line ~125) and add the new arg:

```go
order, err := h.svc.Checkout(r.Context(), userID, idempotencyKey, items, req.Currency, req.PaymentMethodID)
```

- [ ] **Step 7: Fix the test mock to match the new signature**

In `services/order-management-service/internal/service/order_service_test.go`, find the mock struct that implements `paymentGateway` (it has a `ChargeCard` method). Update its signature:

```go
func (m *mockPaymentGateway) ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, string, error) {
```

Also find every call to `svc.Checkout(` in the test file and add `""` as the last argument (paymentMethodID):

```go
order, err := svc.Checkout(ctx, userID, idemKey, items, "INR", "")
```

- [ ] **Step 8: Build and test**

```bash
cd services/order-management-service
go build ./...
go test ./...
```

Expected: all tests pass, no compilation errors.

- [ ] **Step 9: Commit**

```bash
git add services/order-management-service/
git commit -m "feat(order): thread payment_method_id through checkout saga"
```

---

### Task 2: Add Stripe env vars to docker-compose

**Files:**
- Modify: `docker-compose.yml`

**Interfaces:**
- Produces: payment-service container has `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` env vars at runtime

- [ ] **Step 1: Add Stripe env vars to payment-service in docker-compose.yml**

Find the `payment-service` environment block (currently around line 418). Add two lines:

```yaml
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY:-}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET:-}
```

The full environment block becomes:

```yaml
    environment:
      - DB_HOST=zapmarket-postgres
      - DB_PORT=5432
      - DB_USER=zapuser
      - DB_PASSWORD=zappass123
      - DB_NAME=payment
      - HTTP_PORT=8083
      - GRPC_PORT=50054
      - APP_ENV=development
      - PAYMENT_WEBHOOK_SECRET=change-me-webhook-secret
      - REDIS_URL=zapmarket-redis:6379
      - KAFKA_BROKERS=kafka:9092
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY:-}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET:-}
```

- [ ] **Step 2: Create a `.env` at repo root if it doesn't exist and add Stripe vars**

```bash
# at repo root
STRIPE_SECRET_KEY=sk_test_YOUR_KEY_HERE
STRIPE_WEBHOOK_SECRET=whsec_YOUR_SECRET_HERE
```

Docker Compose reads `.env` from the project root automatically.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(infra): expose STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET to payment-service container"
```

---

### Task 3: Add Stripe Elements card UI to buyer-ui checkout

**Files:**
- Modify: `services/buyer-ui/app/checkout/page.tsx`
- Modify: `services/buyer-ui/.env.local` (create if missing) and `services/buyer-ui/.env.example`

**Interfaces:**
- Consumes: `@stripe/stripe-js` and `@stripe/react-stripe-js` npm packages
- Produces: checkout page renders a Stripe `CardElement` when `paymentMethod === "card"` is selected; on "Place Order" click, creates a PaymentMethod via Stripe.js and puts `pm_xxx` into the order POST body

- [ ] **Step 1: Install Stripe.js packages**

```bash
cd services/buyer-ui
npm install @stripe/stripe-js @stripe/react-stripe-js
```

Expected output: packages added to `package.json`, no peer-dep errors.

- [ ] **Step 2: Add publishable key env var**

Create/edit `services/buyer-ui/.env.local`:

```
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_YOUR_PUBLISHABLE_KEY_HERE
```

Also add the line (with empty default) to `services/buyer-ui/.env.example` so other developers know about it:

```
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=
```

- [ ] **Step 3: Create Stripe provider utility**

Create `services/buyer-ui/lib/stripe.ts`:

```typescript
import { loadStripe, Stripe } from "@stripe/stripe-js";

let stripePromise: Promise<Stripe | null>;

export function getStripe(): Promise<Stripe | null> {
  if (!stripePromise) {
    stripePromise = loadStripe(
      process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? ""
    );
  }
  return stripePromise;
}
```

- [ ] **Step 4: Rewrite checkout page with card option**

Replace the entire contents of `services/buyer-ui/app/checkout/page.tsx` with the following. Key changes vs the original:
- Payment options now include **Card** (via Stripe Elements) in addition to COD; UPI stays disabled
- `Elements` provider wraps the form when card is selected
- On submit, if card is selected, calls `stripe.createPaymentMethod()` first, then sends `payment_method_id` in the body

```tsx
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
import { useCartStore } from "@/lib/cart";
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
  items: ReturnType<typeof useCartStore>["items"];
  subtotal: number;
  clearCart: () => void;
  getIdempotencyKey: () => string;
}

function CheckoutForm({
  address, setAddress, paymentMethod, setPaymentMethod,
  promoCode, setPromoCode, promoMsg, setPromoMsg,
  error, setError, loading, setLoading,
  items, subtotal, clearCart, getIdempotencyKey,
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

      // Tokenize card via Stripe.js before hitting the backend
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
    items, subtotal, clearCart, getIdempotencyKey,
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
```

- [ ] **Step 5: Build check**

```bash
cd services/buyer-ui
npx tsc --noEmit
```

Expected: no TypeScript errors.

- [ ] **Step 6: Commit**

```bash
git add services/buyer-ui/
git commit -m "feat(buyer-ui): add Stripe Elements card payment to checkout"
```

---

### Task 4: Manual E2E smoke test

**Goal:** Verify the full flow works end-to-end in Docker before declaring done.

- [ ] **Step 1: Rebuild and restart the stack**

```bash
# from repo root
docker compose up -d --build order-management-service payment-service
```

- [ ] **Step 2: Start buyer-ui locally**

```bash
cd services/buyer-ui
npm run dev
```

Open `http://localhost:3000` in browser.

- [ ] **Step 3: Test card payment**

1. Log in as a buyer
2. Add any product to cart
3. Go to checkout — the **Credit / Debit Card** option should now be selected by default and show a Stripe card input field
4. Enter Stripe test card: `4242 4242 4242 4242`, expiry `12/26`, CVC `123`, any postcode
5. Fill in address fields
6. Click **Place Order**
7. Expected: redirected to `/account/orders/<id>?new=1`; order status is `CONFIRMED`

- [ ] **Step 4: Test COD payment**

1. Add a product to cart, go to checkout
2. Select **Cash on Delivery** — card input should disappear
3. Fill address, click **Place Order**
4. Expected: order placed successfully; no Stripe charge attempted; order status is `CONFIRMED` (FakeGateway handles COD path since `payment_method_id` is empty)

- [ ] **Step 5: Test Stripe declined card**

1. At checkout, select card, enter `4000 0000 0000 0002` (Stripe test decline card)
2. Click **Place Order**
3. Expected: error banner appears with Stripe's decline message; no order created

- [ ] **Step 6: Commit if any fixes were made during testing, then final commit**

```bash
git add -p
git commit -m "fix: post-smoke-test corrections"
```

---

## Self-Review

**Spec coverage:**
- ✅ Stripe card UI on checkout → Task 3
- ✅ `payment_method_id` through order service → Task 1
- ✅ gRPC client passes `PaymentMethodId` → Task 1, Step 5
- ✅ COD skips card charge (empty `paymentMethodID` → FakeGateway) → Task 3 (card only shown when card selected), Task 4 Step 4
- ✅ docker-compose env vars → Task 2
- ✅ Stripe test card smoke test → Task 4

**Placeholder scan:** None found — all steps include exact code.

**Type consistency:**
- `paymentMethodID string` used consistently across `paymentGateway` interface, `Checkout`, `finalisePayment`, `ChargeCard`
- `PaymentMethodId` (proto field name) used in `pb.ChargeCardRequest` — matches proto field 6
- `payment_method_id` (JSON) in checkout page body and `checkoutRequest.PaymentMethodID` (Go) — consistent via `json:"payment_method_id,omitempty"`
