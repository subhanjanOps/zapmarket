# seller-ui Review

_Principal Engineer production-grade review — ZapMarket `services/seller-ui` (Next.js App Router). Focus: recent multi-currency + BFF changes._

## Executive Summary

The seller-ui is a well-structured Next.js App Router application with a genuinely sound security posture: the JWT lives in an httpOnly cookie and every authenticated call is funneled through a server-side BFF proxy (`app/api/proxy/[...path]/route.ts`) that injects `Authorization` from the cookie. Client JS never touches the raw token. This is the correct pattern and is better than the localStorage-token approach common in many React dashboards. Middleware enforces auth at the edge, and `lib/api.ts` is a clean, typed, single-responsibility API client.

The recent currency work is functional and the UX (locale detection, persistent preference, stale-rate footnote, currency-aware cents conversion) is thoughtful. However it concentrates four distinct responsibilities into one `currency.tsx` file, duplicates `toCents`/`fromCents` across two pages, and the localStorage cache/SSR-safety patterns have correctness edges worth tightening.

The most material concerns are: (1) an **inconsistent proxy path convention** (`/api/proxy/v1/...` vs `/api/proxy/api/v1/...`) that means some new endpoints likely 404 against the gateway; (2) an **orphaned `app/api/fx-rates/route.ts`** staged in git but absent from disk and never referenced; (3) **zero tests** for currency conversion math that touches money; and (4) the BFF proxy performs **no allowlist validation** of forwarded paths.

The engineering-standards docs are Go/backend-oriented (SOLID, layering, no-SQL-in-handlers) and apply only by analogy here, but the spirit — single responsibility, remove duplication, testability — is directly violated in a few spots called out below.

Verdict: **APPROVED WITH RECOMMENDATIONS** (conditional on resolving the two Critical items, which are likely runtime breakages).

---

## Critical Findings

### C1. Inconsistent proxy path prefix — currency/preferences endpoints likely 404
Two prefix conventions coexist against the same gateway:

- `api/v1`: products, skus, images, categories, **currencies** (`/api/proxy/api/v1/currencies`, `/api/proxy/api/v1/currencies/rates`)
- `v1` (no `api`): auth register, orders, **user preferences** (`/api/proxy/v1/users/me/preferences`)

`lib/currency.tsx` mixes both: it fetches rates/list under `api/v1/currencies` but the display-currency preference under `v1/users/me/preferences`. At most one of these conventions can be correct for a given gateway. Either the preference sync silently fails (caught and swallowed in `fetchServerPreference`/`saveServerPreference`, so it would be invisible), or the currency list/rates fail. Because every failure path here is a silent `catch`, this will not surface as an error — it will manifest as "preference doesn't persist across devices" or "no currencies in the dropdown," which is exactly the kind of bug that ships unnoticed.

**Action:** Confirm the gateway's actual route prefixes (currency-service is port 8086, `/v1/...` per service convention; product-catalog uses `/api/v1/...`). Normalize the client to the real paths and add a thin typed helper so the prefix is declared once, not string-built at 12 call sites.

### C2. Orphaned BFF route `app/api/fx-rates/route.ts`
Git lists `services/seller-ui/app/api/fx-rates/route.ts` as staged-added, but it does not exist on disk and no code references `/api/fx-rates` (the only hit for "fx-rates" is the localStorage key `zap-fx-rates`). This is dead/abandoned scaffolding from an earlier iteration of the currency feature — rates now flow through the generic `[...path]` proxy. Ship a dead route and you carry an unowned, untested surface.

**Action:** Either delete the staged file (`git rm --cached`) or restore it and wire it in deliberately. Do not merge a phantom route.

---

## High Priority Findings

### H1. BFF proxy forwards arbitrary paths with no allowlist (SSRF/abuse surface)
`app/api/proxy/[...path]/route.ts` forwards `GW/<any path the client supplies>` with the seller's bearer token attached, for all of GET/POST/PUT/PATCH/DELETE. Auth-wise it is gated by middleware (cookie required), so it is not anonymous SSRF — but any authenticated seller can reach **every** gateway route with their token, including endpoints the UI never intended to expose (admin routes, other services' v1 surfaces). The BFF's value is partly to be a narrow, intentional seam; today it is a wide-open passthrough.

**Action:** Add a prefix allowlist (e.g. permit only `api/v1/products`, `api/v1/skus`, `api/v1/products/*/images`, `api/v1/categories`, `v1/orders/seller*`, `v1/users/me/preferences`, `api/v1/currencies*`) and reject anything else with 404. This also makes C1's path set explicit and self-documenting.

### H2. No tests anywhere — money math is untested
There are no `*.test.*`/`*.spec.*` files and no test runner wired. The highest-risk logic in the app is currency conversion, and it is duplicated and subtle:

- `toCents`/`fromCents` (zero-decimal handling: JPY round vs ×100)
- `format` (USD cents → display) and `formatFrom` (arbitrary currency → USD → display) in `currency.tsx`
- the dashboard revenue rollup: `o.total_amount / (rates[o.currency] ?? 1)` then `format(Math.round(...))`

A single off-by-100 or a wrong zero-decimal branch silently misprices products or misreports revenue. This is the textbook case the standards call out as "must be testable." 

**Action:** Add Vitest + a `currency.test.ts` covering: 2-decimal and 0-decimal round-trips (`fromCents(toCents(x))`), `formatFrom` cross-currency, and unknown-currency fallback (rate missing → `?? 1`). This is low effort, high leverage.

### H3. `toCents`/`fromCents` duplicated across pages (DRY / standards "remove duplication")
`toCents` is defined identically in `products/new/page.tsx` and `products/[id]/page.tsx`; `fromCents` lives in `[id]` and the decimals lookup is re-implemented a third time in `SKUEditor.decimalsFor` and a fourth in `currency.getDecimals`. Four copies of "how many decimals does this currency have." When the rule changes (e.g. a 3-decimal currency like BHD/KWD — none are in `ZERO_DECIMAL` but they exist), you must find all four.

**Action:** Export `toCents`, `fromCents`, and `decimalsFor(code, currencies)` from `lib/currency.tsx` (or a small `lib/money.ts`) and consume them everywhere. The provider already holds `currencies`; expose the converters off `useCurrency()` so they're always rate/meta-consistent.

### H4. `currency.tsx` has four responsibilities in one module (SRP)
The file is the context provider **and** the rates fetcher **and** the currency-list fetcher **and** the server-preference client **and** locale detection **and** formatting. Per the standards' Single Responsibility rule, this is "more than one reason to change." It's readable today only because each piece is small, but it will accrete.

**Action:** Split into `lib/currency/rates.ts` (fetch+cache), `lib/currency/preferences.ts` (server sync), `lib/currency/detect.ts` (locale), and keep `currency.tsx` as the thin provider/hook. No behavior change, large readability/testability win (each fetcher becomes independently unit-testable, addressing H2).

---

## Medium Priority Findings

### M1. localStorage cache: no shared TTL helper, partial corruption handling
`fetchRates` and `fetchCurrencyList` both hand-roll the "read JSON, check `Date.now() - ts < TTL`, fall back to stale on error" dance with slightly different shapes. A malformed cache entry (e.g. schema change between deploys) is handled inconsistently — `fetchRates`'s outer `catch` re-reads and assumes the old shape; if the shape changed you get `undefined` fields rather than a clean refetch. Centralize a `cachedFetch(key, ttl, fetcher)` helper.

### M2. SSR-safety relies on effect timing, not guards
`fetchRates`/`fetchCurrencyList` call `localStorage` directly. They're safe today only because they're invoked from inside `useEffect` (client-only) in `CurrencyProvider`. But they're `export`ed (`fetchRates` is exported) and have no `typeof window` guard, so any future server-side import would throw at module-eval/first-call. The `try/catch` around `navigator.language` in `detectCurrency` is correct defensive style; apply the same discipline (or an explicit `typeof window === "undefined"` early-return) to the storage functions so they're safe by construction, not by call-site luck.

### M3. `CurrencyProvider` initial flash / double-set of currency
The provider initializes `currency` to `"USD"`, then in the effect sets it from localStorage/locale, then potentially again from the server preference — up to three state transitions on mount, each a re-render of every currency consumer (layout, dashboard, SKUEditor, both product pages). Because the whole context value object is rebuilt each render and `format`/`formatFrom` are recreated when `rates` changes, consumers re-render more than necessary. Consider `useMemo` on the context value and seeding `currency` synchronously via a lazy `useState(() => ...)` reading localStorage (guarded) to cut the flash.

### M4. Silent failures hide real problems (observability standard)
Every network failure in currency.tsx is swallowed (`catch { /* */ }` or silent return). The standards require structured logging and "never ignore errors." On the client you obviously can't use the Go logger, but silently dropping a preference-save or a rates-fetch failure means the C1 path bug is undetectable. At minimum `console.warn` with context, ideally surface stale/failed state to the user (the stale footnote partially does this for rates, but not for list/preference failures).

### M5. Dashboard revenue rollup is fragile
`page.tsx` computes month revenue from only the most recent 10 orders (`getSellerOrders({ limit: 10 })`) — so "Revenue This Month" is wrong whenever a seller has >10 orders in the window. The label says "confirmed only" but the denominator is "last 10 fetched." Either fetch with a server-side `from` filter and a real total, or compute revenue server-side. As written it's a correctness bug masquerading as a stat.

### M6. SKU currency selector UX nested inside `<label>`
In `SKUEditor`, the price-currency `<select>` is rendered *inside* the `<label>Price *</label>` for the price input. Clicking the select toggles focus to the associated input via the label, and screen readers will read the select as part of the price field's label. The currency selector should be a sibling control with its own accessible name, not nested in another field's label.

### M7. `formatFrom` / `format` default context values diverge from real behavior
The `createContext` default (used before the provider mounts or if a consumer is mis-nested) hardcodes `$` formatting and a `USD:1` rate. A consumer rendered outside the provider would silently get dollar-sign output regardless of locale. Low real-world risk given the root layout wraps everything, but the default should at least be honest (e.g. throw in dev if used outside provider) per Liskov-style "never surprise callers."

---

## Low Priority Findings

- **L1.** Inline `React.CSSProperties` style objects are duplicated across nearly every component (`inputStyle`, `skuInputStyle`, the danger-banner div). Extract to CSS classes (a `globals.css` already exists) to cut noise and enable theming consistency.
- **L2.** `publish()` in `new/page.tsx` does `await import("@/lib/api")` for `updateProduct` even though the module is already statically imported at the top. The dynamic import is pointless — use the existing static import.
- **L3.** `[id]/page.tsx` imports `Trash2` twice (once in the grouped lucide import line 5 region, again at line 12). Dedupe.
- **L4.** Attribute rows in `SKUEditor` use array index as React `key` (`key={ai}`); reordering/removing mid-list can mis-associate inputs. Use a stable id per attribute row.
- **L5.** `detectCurrency` maps region→currency with a hardcoded 40-entry table that duplicates information the server's currency list already implies. Acceptable as a first-paint heuristic, but note it's a second source of truth.
- **L6.** Magic timeout `setTimeout(() => setNavigating(false), 500)` in the layout is a guessed duration unrelated to actual navigation completion; fine for a progress shimmer but worth a comment.
- **L7.** `seller_auth_hint` cookie is set but `lib/auth.ts` no longer reads it (only `logout` remains). Confirm it's still consumed somewhere or drop it.

---

## Recommended Refactoring Plan

Ordered by risk-reduction per unit effort:

1. **Resolve C1 + C2 first (blocking):** verify gateway prefixes, normalize all currency/preference paths, delete or wire the orphaned `fx-rates` route.
2. **Extract money utilities (H3):** create `lib/money.ts` with `decimalsFor`, `toCents`, `fromCents`; replace all four duplicated copies. Mechanical, low risk.
3. **Add tests (H2):** Vitest + `money.test.ts` (round-trips, zero-decimal, missing-rate fallback) + a `formatFrom` cross-currency case. Gate CI on it.
4. **Split `currency.tsx` (H4):** move rates/preferences/detect into sibling modules; provider becomes ~60 lines. Now each fetcher is unit-testable and the silent-catch problem (M4) is isolated.
5. **Harden the proxy (H1):** add path allowlist; this formalizes the route set from step 1.
6. **Tighten provider rendering + SSR guards (M2, M3):** lazy-init currency from guarded localStorage, `useMemo` the context value.
7. **Fix the revenue stat (M5)** and the nested-label a11y issue (M6); sweep the Low items opportunistically.

---

## Final Verdict

**APPROVED WITH RECOMMENDATIONS**

The architecture is right (httpOnly cookie + server-side BFF + typed API client + edge middleware), and the feature works. But two items (C1 path inconsistency, C2 orphaned route) are very likely live defects and should be resolved before merge, and the absence of any test around money math (H2) is a standards violation on the highest-risk code in the app. None require redesign — they're verify-and-tidy plus a small test file. Treat C1/C2 as merge blockers; H1–H4 as fast-follows.
