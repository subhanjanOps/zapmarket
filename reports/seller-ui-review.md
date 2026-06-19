# Seller UI — Principal Engineer Code Review

**Date:** 2026-06-19  
**Reviewer:** Claude (Principal Engineer Review)  
**Scope:** `services/seller-ui/` — all source files

---

## Executive Summary

The Seller UI is a well-structured Next.js 16 App Router application with clean component decomposition, good UX patterns (multi-step wizard, skeleton loaders, responsive mobile layout), and consistent visual design. It is largely production-ready for an MVP. However, it has one **critical security flaw** (token in `localStorage`), several architectural divergences from the project's BFF pattern used in admin-ui, and a handful of medium-priority issues around validation and error recovery.

---

## Critical Findings

### C1: JWT stored in `localStorage` — XSS-exploitable

`lib/auth.ts` stores the JWT in `localStorage`. Any XSS vector (injected script, third-party lib, etc.) can exfiltrate the token silently. The admin-ui uses `httpOnly` cookies via a BFF proxy. The seller-ui bypasses this entirely: the token is attached to every API call from the browser.

```ts
// lib/auth.ts — "use client"
export const saveToken = (t: string) => localStorage.setItem(KEY, t);
```

**Impact:** Full account takeover if any XSS is ever introduced. **Fix:** Mirror admin-ui's approach — proxy through a Next.js API route, set `httpOnly; Secure; SameSite=Strict` cookie on login, never expose raw JWT to JavaScript.

---

## High Priority Findings

### H1: Token read on every render, not stored in React state

`getToken()` is called inline during every effect and handler (`getProducts(getToken(), ...)`) instead of once on mount. If the token is cleared mid-session (logout in another tab), in-flight effects proceed with `null`. The auth guard in `layout.tsx` only fires once on mount.

### H2: `getProducts` fetches 100 products on dashboard load

`dashboard/page.tsx:22` calls `getProducts(token, { limit: 100 })` to derive stat card counts. For a seller with large inventory this is wasteful. The backend likely supports a `/stats` or count-only endpoint; at minimum this should be `limit: 20` and stats should come from the response `total`.

### H3: SKU creation is sequential in a loop, not atomic

`products/new/page.tsx:55-72` — SKUs are created one-by-one in a `for` loop. If SKU #3 of 5 fails, 2 orphaned SKUs are left attached to the product with no cleanup. There's no rollback and the user sees a generic error with no indication of partial state.

```ts
for (const s of skus) {
  await createSku(token, { ... }); // no rollback if this throws mid-loop
}
```

### H4: `getSellerOrders` total is calculated client-side from page results

`lib/api.ts:295-296` — `total` is set to `r.data.length` (the current page size), not the real backend total. The orders page pagination (`pages = Math.ceil(total / PAGE_SIZE)`) will always show 0 or 1 page regardless of actual order count, breaking pagination entirely.

```ts
// incorrect — total is page-result count, not server total
return { orders, total: orders.length, limit: params.limit ?? 20, offset: params.offset ?? 0 };
```

---

## Medium Priority Findings

### M1: No CSRF protection on state-mutating requests

All mutations go through `fetch` with `Authorization: Bearer` headers from JS. If the project migrates to `httpOnly` cookies (recommended above), CSRF tokens or `SameSite=Strict` must be enforced. The current setup is safe only because the token is not in a cookie, but that's a broken security model.

### M2: `useCallback` dependency causes reload on every keystroke

In `products/page.tsx` and `orders/page.tsx`, `load` is wrapped in `useCallback` and passed to `useEffect([load])`. Changing `search` triggers a new `load` reference on every keystroke, which triggers `useEffect`, which calls `load`. This fires an API request on every character typed with no debounce — both a performance issue and a UX issue (many concurrent requests, last-one-wins race).

### M3: Image upload throws bare `Error("No product ID")`

`products/new/page.tsx:79` — `handleImageUpload` throws `new Error("No product ID")` if called before `productId` is set. The `ImageDropzone` gets this as a Promise rejection with no user-visible recovery. The step flow should prevent this, but the guard is implicit.

### M4: `slug` field allows arbitrary user input without format validation

The slug field (`products/new/page.tsx:137`) accepts free text with no enforcement of slug format (lowercase, no spaces, etc.) beyond the auto-slugify from the name. A manually entered slug like `My Product!` will be sent to the backend as-is and either fail silently or be stored invalidly.

### M5: SKU index used as React key

`SKUEditor.tsx:76` — `key={si}` (array index) causes React reconciliation bugs when SKUs are reordered or deleted from the middle. Should use a stable `id` or generated UUID.

### M6: Revenue calculation ignores currency field

`dashboard/page.tsx:48-49` reduces `total_amount` across all CONFIRMED orders assuming USD, but `Order` has a `currency` field. Multi-currency sellers will see incorrect revenue totals.

---

## Low Priority Findings

### L1: Inline `inputStyle` object redeclared on every render

Both `products/new/page.tsx` and `SKUEditor.tsx` define `const inputStyle: React.CSSProperties = {...}` inside the component body. Should be hoisted to module level or replaced with a CSS class.

### L2: `sellerStatus` defaults to `"APPROVED"` on `getMe` failure

`layout.tsx:59-61` — if `getMe` throws (network error, 500), the seller is silently treated as approved and shown the full dashboard. An intermittent auth failure would appear as a successful login.

### L3: `cancelOrder` exported but never used in the UI

Dead export in `lib/api.ts` — no cancel order UI exists. Adds confusion.

### L4: Date filter inputs lack visible labels (WCAG 2.1 AA failure)

`orders/page.tsx:69-70` — `<input type="date" title="From date" />` with no visible `<label>` fails WCAG 2.1 AA (Success Criterion 1.3.1). Screen readers won't announce the purpose of the field.

### L5: `thisMonth` computed via mutable `Date` mutation

`dashboard/page.tsx:45` uses imperative mutation (`setDate(1)`, `setHours(...)`) instead of constructing a clean date: `new Date(new Date().getFullYear(), new Date().getMonth(), 1)`.

---

## Recommended Refactoring Plan

### Priority 1 — Before production
1. Replace `localStorage` JWT with BFF route + `httpOnly` cookie (mirrors admin-ui pattern)
2. Fix `getSellerOrders` `total` — use server-provided total or `X-Total-Count` header
3. Add 300ms debounce to search input in `products/page.tsx`

### Priority 2 — Next sprint
4. Add parallel SKU creation with `Promise.allSettled` and per-SKU error display, or implement server-side batch SKU endpoint
5. Move `getToken()` call to layout-level React context, provide token via context to child pages
6. Fix SKU `key={si}` → stable ID (e.g. `crypto.randomUUID()` in `emptyDraft`)

### Priority 3 — Polish
7. Add `<label>` elements to date filter inputs
8. Add slug format validation on blur
9. Hoist `inputStyle` to module scope
10. Handle `getMe` 401 specifically vs network error in layout — don't silently approve on errors

---

## Final Verdict

### APPROVED WITH RECOMMENDATIONS

The seller UI is well-built for an MVP. The component structure is clean, the multi-step product wizard is well-designed, responsive layout handles mobile correctly, and the design system integration is consistent. The critical `localStorage` token issue must be addressed before any real seller data is handled, but the rest of the codebase is solid enough to ship with the high-priority fixes in flight.
