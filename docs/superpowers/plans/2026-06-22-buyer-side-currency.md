# Buyer-Side Currency Display Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make monetary amounts in buyer-facing notification messages human-readable and correctly formatted, and show order totals in the seller's preferred display currency on the seller-ui order detail page.

**Architecture:** Two independent improvements:
1. **Notification formatting** — the `notification-service` consumer already receives `amount` (in cents) and `currency` (ISO code) in payment events. The templates currently print raw integers like `"123456 USD"`. This task formats them as `"$1,234.56"` using Go's standard number formatting without calling any external service.
2. **Seller-ui order detail** — the order detail page fetches an order and shows its total. The existing `useCurrency()` context exposes `formatFrom(cents, fromCurrency)` which already converts to the seller's display currency. This task wires it into the order detail page.

**Tech Stack:** Go 1.25 (notification-service), Next.js + TypeScript (seller-ui), existing `useCurrency()` hook.

## Global Constraints

- notification-service has NO HTTP server — it is a Kafka consumer only; no new endpoints
- Amount formatting in Go: `fmt.Sprintf("%.2f", float64(cents)/100)` for 2-decimal currencies; for zero-decimal currencies (JPY, KRW, IDR) omit the decimal: `fmt.Sprintf("%d", cents/100)` — do NOT call currency-service from notification-service
- Zero-decimal currencies for notification-service: JPY, KRW, IDR (hardcoded set — not an API call)
- seller-ui order detail page must use `formatFrom` from `useCurrency()` — do NOT re-implement currency conversion
- This is NOT standard Next.js — read `node_modules/next/dist/docs/` if unsure about any API

---

### Task 1: Notification message amount formatting

**Files:**
- Modify: `services/notification-service/internal/consumer/handler.go`
- Test: `services/notification-service/internal/consumer/handler_test.go`

**Interfaces:**
- Consumes: existing `payload["amount"]` (string, amount in cents) and `payload["currency"]` (string, ISO code)
- Produces: `formatAmount(amountCents string, currency string) string` private helper; updated notification bodies

- [ ] **Step 1: Write the failing test**

```go
// services/notification-service/internal/consumer/handler_test.go
package consumer_test

import (
    "testing"

    "github.com/zapmarket/zapmarket/services/notification-service/internal/consumer"
)

func TestFormatAmount_USD(t *testing.T) {
    got := consumer.FormatAmountForTest("123456", "USD")
    want := "$1,234.56"
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}

func TestFormatAmount_JPY_ZeroDecimal(t *testing.T) {
    got := consumer.FormatAmountForTest("150000", "JPY")
    want := "¥1,500"
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}

func TestFormatAmount_EUR(t *testing.T) {
    got := consumer.FormatAmountForTest("9999", "EUR")
    want := "€99.99"
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}

func TestFormatAmount_InvalidAmount(t *testing.T) {
    got := consumer.FormatAmountForTest("bad", "USD")
    // Should return the raw string rather than crash
    if got == "" {
        t.Error("expected non-empty fallback for bad input")
    }
}
```

**Note:** `consumer.FormatAmountForTest` is a thin exported wrapper around the unexported `formatAmount` — add it only for testing, gated with a build tag or in a `_test.go`-adjacent exported function.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/notification-service
go test ./internal/consumer/... -run TestFormatAmount -v
```

Expected: FAIL with `undefined: consumer.FormatAmountForTest`

- [ ] **Step 3: Implement formatAmount in handler.go**

Add to `services/notification-service/internal/consumer/handler.go`:

```go
import (
    "strconv"
    "strings"
)

var zeroDecimalCurrencies = map[string]bool{
    "JPY": true, "KRW": true, "IDR": true,
}

var currencySymbols = map[string]string{
    "USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥", "KRW": "₩",
    "INR": "₹", "CNY": "¥", "AUD": "A$", "CAD": "C$", "CHF": "Fr",
}

func formatAmount(amountCents string, currency string) string {
    cents, err := strconv.ParseInt(amountCents, 10, 64)
    if err != nil {
        return amountCents + " " + currency
    }
    sym := currencySymbols[currency]
    if sym == "" {
        sym = currency + " "
    }
    if zeroDecimalCurrencies[currency] {
        return sym + formatWithCommas(cents/100)
    }
    whole := cents / 100
    frac  := cents % 100
    if frac < 0 {
        frac = -frac
    }
    return fmt.Sprintf("%s%s.%02d", sym, formatWithCommas(whole), frac)
}

func formatWithCommas(n int64) string {
    s := strconv.FormatInt(n, 10)
    if len(s) <= 3 {
        return s
    }
    var b strings.Builder
    rem := len(s) % 3
    if rem > 0 {
        b.WriteString(s[:rem])
    }
    for i := rem; i < len(s); i += 3 {
        if i > 0 || rem > 0 {
            b.WriteByte(',')
        }
        b.WriteString(s[i : i+3])
    }
    return b.String()
}

// FormatAmountForTest exposes formatAmount for unit tests.
// Only used in tests — not called from production code.
func FormatAmountForTest(amountCents, currency string) string {
    return formatAmount(amountCents, currency)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/notification-service
go test ./internal/consumer/... -run TestFormatAmount -v
```

Expected: all 4 tests PASS

- [ ] **Step 5: Update notification templates to use formatAmount**

In `handler.go`, update the `buildNotification` cases that use `payload["amount"]`:

```go
// payment.captured — was:
Body: fmt.Sprintf("Your payment of %s %s for order %s was successful.",
    payload["amount"], payload["currency"], payload["order_id"]),

// Change to:
Body: fmt.Sprintf("Your payment of %s for order %s was successful.",
    formatAmount(payload["amount"], payload["currency"]), payload["order_id"]),

// payment.refunded — was:
Body: fmt.Sprintf("A refund of %s %s for order %s has been processed...",
    payload["amount"], payload["currency"], payload["order_id"]),

// Change to:
Body: fmt.Sprintf("A refund of %s for order %s has been processed and will appear within 3-5 business days.",
    formatAmount(payload["amount"], payload["currency"]), payload["order_id"]),
```

- [ ] **Step 6: Run all notification-service tests**

```bash
cd services/notification-service
go test ./... -v
```

Expected: all pass

- [ ] **Step 7: Build to verify**

```bash
cd services/notification-service
go build ./...
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add services/notification-service/internal/consumer/handler.go \
        services/notification-service/internal/consumer/handler_test.go
git commit -m "feat(notification): format monetary amounts in notification messages"
```

---

### Task 2: seller-ui order detail page — converted total

**Files:**
- Modify: `services/seller-ui/app/dashboard/orders/[id]/page.tsx`

**Interfaces:**
- Consumes: `useCurrency()` hook from `@/lib/currency` (already imported in orders list page)
- Produces: order total displayed in seller's preferred currency using `formatFrom(order.total_cents, order.currency)`

- [ ] **Step 1: Read the current order detail page**

Read `services/seller-ui/app/dashboard/orders/[id]/page.tsx` in full before making changes.

- [ ] **Step 2: Add currency formatting to order total display**

Find where the order's total/amount is rendered (look for the field that shows the monetary value — it will be something like `order.total` or `order.amount`).

Add `useCurrency` import if not already present:

```typescript
import { useCurrency } from "@/lib/currency";
```

Inside the component, destructure `formatFrom` and `currency` (the display currency label):

```typescript
const { formatFrom, currency } = useCurrency();
```

Where the total is rendered, replace the raw amount display with:

```tsx
// If the order has a total_cents field and a currency field:
<span>{formatFrom(order.total_cents, order.currency)}</span>
<span style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
  ({currency})
</span>
```

The exact field names (`total_cents`, `currency`) depend on the `Order` type in `lib/api.ts`. Read that type first and use the correct field names.

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd services/seller-ui
npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add services/seller-ui/app/dashboard/orders/
git commit -m "feat(seller-ui): show order total in seller's preferred display currency"
```
