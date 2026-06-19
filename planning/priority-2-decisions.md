# Priority 2 — Decision Report

**Branch:** `features/cluster-setup`  
**Date:** 2026-06-19  
**Author:** Code Review (Principal Engineer)

---

## Overview

Priority 2 contains four items that must be resolved before load testing or any staging deployment. None require architectural change — all are targeted fixes to existing code. This document records the decision rationale, alternatives considered, and the chosen approach for each item so future engineers understand *why* the change was made, not just *what* changed.

---

## Item 1 — HIGH-2: Replace Fixed-Window Rate Limiter with Sliding Window

### Problem

`services/api-gateway/internal/middleware/ratelimit.go` implements a fixed-window counter using `INCR` + `EXPIRE`:

```go
count, _ := rl.rdb.Incr(ctx, key).Result()
if count == 1 {
    _ = rl.rdb.Expire(ctx, key, rateWindowSecs*time.Second).Err()
}
```

A fixed window resets atomically at a clock boundary. This creates a well-known double-spend window: a client can send `N` requests at second 59, the window resets at second 60, and the client immediately sends another `N` requests — delivering `2N` requests in 2 seconds against an `N req/min` policy.

At the configured limit of 200 req/min per IP, a sustained attacker can reliably push 400 requests in a 2-second burst every 60 seconds.

### Alternatives Considered

| Approach | Correctness | Redis ops/req | Notes |
|---|---|---|---|
| Fixed window (current) | ❌ double-spend | 1–2 | Too leaky |
| Sliding window — sorted set | ✅ exact | 3 (`ZADD` + `ZREMRANGE` + `ZCARD`) | Standard; supports burst inspection |
| Token bucket — Lua script | ✅ approximate | 1 (single Lua) | Better for burst allowance, harder to reason about |
| `redis-cell` / `CL.THROTTLE` | ✅ exact | 1 (module command) | Requires Redis module; not universally available |
| Sliding window — Lua script | ✅ exact | 1 (atomic Lua) | Same as sorted set but atomic; slightly more complex |

### Decision

**Sorted-set sliding window.** Three Redis operations per request (vs. 1–2 for fixed-window) is an acceptable cost at the gateway tier. The sorted-set approach is the industry standard, well-understood, and inspectable via `ZRANGE` for debugging. A Lua script would reduce round trips to one but adds opaqueness we don't need right now — that's a future optimisation.

The window is `[now - 60s, now]`. On each request:
1. `ZADD key now now` — record this request (score = timestamp in ms, member = timestamp)
2. `ZREMRANGEBYSCORE key 0 (now - window_ms)` — evict old entries
3. `ZCARD key` — count requests in current window
4. `EXPIRE key window_s` — keep the key alive

Steps 1–3 must be atomic to avoid a race between ZADD and ZREMRANGE. We wrap them in a Lua script (single round trip).

**Key expiry:** Set TTL to `rateWindowSecs + 1` after each write. This ensures keys auto-clean without a sweep job.

---

## Item 2 — HIGH-1: Gate Localhost CORS Wildcard to Development Only

### Problem

`services/api-gateway/main.go:323-331`:

```go
func originAllowed(origin, allowed string) bool {
    if origin == allowed {
        return true
    }
    return strings.HasPrefix(origin, "http://localhost:") ||
        strings.HasPrefix(origin, "http://127.0.0.1:")
}
```

The localhost wildcard is a development convenience — it lets the seller-ui on port 3002 and backoffice-ui on port 3001 both hit the gateway without reconfiguring `ADMIN_UI_ORIGIN`. However, the same binary and the same `originAllowed` function runs in staging and production. Any process on a victim's machine running a local HTTP server (intentionally or as malware) can make credentialed cross-origin requests to the API gateway through the victim's browser.

### Alternatives Considered

| Approach | Security | DX |
|---|---|---|
| Remove localhost wildcard entirely | ✅ | ❌ breaks local multi-port dev |
| Gate wildcard to `APP_ENV=development` | ✅ | ✅ no change to dev workflow |
| Allow a comma-separated list of origins via env var | ✅ | ✅ flexible for staging with multiple FE origins |
| Move CORS to each upstream service | ✅ | ❌ duplicated config, out of scope |

### Decision

**Gate the localhost wildcard to `APP_ENV=development`.** The gateway already reads `cfg.AppEnv` and passes it through. The fix is a one-line condition inside `originAllowed` — the function receives the env value. No new config needed, no dev workflow change.

In non-development environments, only the exact `ADMIN_UI_ORIGIN` value is allowed. Staging/production operators who need multiple FE origins should set `ADMIN_UI_ORIGIN` to the primary and rely on the exact-match path. A future enhancement (comma-separated allow-list) is noted but out of scope for this fix.

---

## Item 3 — MED-1: Fix Item Index in Checkout Error Messages

### Problem

`services/order-management-service/internal/handler/http/order_handler.go:83`:

```go
"items["+string(rune('0'+i))+"]: sku_id must be a valid UUID"
```

`rune('0' + i)` only produces the correct ASCII digit for `i` in `[0, 9]`. For `i = 10`, `rune('0' + 10)` = `rune(58)` = `':'`, producing `"items[:]"`. For `i = 11`, it produces `"items[;]"`, etc.

An order with 10+ line items — realistic for a marketplace checkout — returns a malformed error message that can confuse clients and makes debugging harder.

### Alternatives Considered

| Approach | Correct | Readable |
|---|---|---|
| `string(rune('0'+i))` (current) | ❌ i > 9 | ✅ |
| `fmt.Sprintf("items[%d]", i)` | ✅ | ✅ |
| `strconv.Itoa(i)` + concatenation | ✅ | ✅ |

### Decision

**`fmt.Sprintf("items[%d]: sku_id must be a valid UUID", i)`** — already importing `fmt` in the handler package. Cleaner than string concatenation with `strconv.Itoa`. The same fix applies to the `seller_id` error message on the line below, which has the same bug.

---

## Item 4 — MED-4: Validation Error on Zero/Negative Refund Amount

### Problem

`services/payment-service/internal/service/payment_service.go:152-154`:

```go
if amount <= 0 || amount > payment.Amount {
    amount = payment.Amount
}
```

A caller passing `amount = 0` (intent unclear — possibly a test call, possibly a client bug) receives a silent full refund. A caller passing `amount = -500` (possible client-side sign error) also receives a silent full refund. This violates the principle of least surprise and creates a financial risk: an integration bug on the client side silently triggers a full refund instead of erroring.

The `amount > payment.Amount` case (partial refund exceeds captured amount) is a legitimate guard — cap-to-full is a reasonable behaviour there and matches how some payment gateways handle it. But `<= 0` is categorically wrong input, not an edge case to normalise.

### Alternatives Considered

| Approach | Safety | API contract |
|---|---|---|
| Silent override to full refund (current) | ❌ financial risk | Unexpected |
| Return validation error for `<= 0`, cap `> payment.Amount` | ✅ | Clear |
| Strict: error for both `<= 0` and `> payment.Amount` | ✅ | Strictest; forces caller to know the amount |

### Decision

**Return a validation error for `amount <= 0`; keep the cap for `amount > payment.Amount`.**

Rationale: zero/negative is unambiguous bad input — there is no valid business intent. Over-amount is a client rounding issue that a graceful cap handles well (and matches Stripe/Razorpay behaviour). Splitting the two cases makes the API behaviour predictable and self-documenting.

---

## Implementation Order

All four items are independent. Apply in this order to minimise diff noise in review:

1. **MED-1** — smallest change, zero risk (string formatting in error path)
2. **MED-4** — two-line change in payment service, purely additive
3. **HIGH-1** — one-line condition, pass `appEnv` into `originAllowed`
4. **HIGH-2** — largest change; replaces the rate-limiter Lua + sorted-set logic

---

## Files Changed

| File | Item | Change type |
|---|---|---|
| `services/order-management-service/internal/handler/http/order_handler.go` | MED-1 | Bug fix |
| `services/payment-service/internal/service/payment_service.go` | MED-4 | Validation added |
| `services/api-gateway/main.go` | HIGH-1 | Security gate |
| `services/api-gateway/internal/middleware/ratelimit.go` | HIGH-2 | Algorithm replacement |
