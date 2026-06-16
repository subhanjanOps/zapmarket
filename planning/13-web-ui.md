# Stage 13 — Web UI (Storefront, Seller, Admin, Warehouse)

**Status: 📋 Planned (2026-06-17), not started**

Not part of the original checklist (which is backend-only) — new scope, added at user
request. Decisions made before writing this plan (asked, not guessed):

| Question | Decision |
|---|---|
| Surfaces | Customer storefront + Seller dashboard + Admin panel + Warehouse management |
| Stack | Next.js (React) |
| Backend wiring | Same monorepo, with a **minimal BFF layer** built first (not calling services directly from the browser, not a full Stage 8 API Gateway either) |

## Goal

A single Next.js app (`apps/web`) serving four role-gated surfaces, talking to the
existing Go services through a thin server-side BFF layer — so the browser never talks
to `auth:8080`/`catalog:8081`/etc. directly, but we also don't build the full Stage 8
gateway (rate limiting, circuit breakers, multi-client routing) just to unblock the UI.

## Architecture decision: BFF = Next.js Route Handlers, not a separate service

Next.js's own `app/api/**/route.ts` handlers run server-side, inside the same process as
the pages. Using them as the BFF means:
- No new deployable service, no new port, no new Dockerfile to maintain just to proxy.
- The browser only ever calls same-origin `/api/*` routes — backend hostnames/ports
  (`zapmarket-auth-service:50051`, etc.) never leak to client code, same security
  property a dedicated gateway would give for this purpose.
- Route handlers can hold a JWT in an `httpOnly` cookie and attach it as
  `Authorization: Bearer` when calling backend services — the browser never sees the
  token.

This is "minimal BFF" in the literal sense: it's the smallest thing that satisfies "don't
call services directly from the browser" without building Stage 8 early. **If Stage 8's
real API Gateway is built later, these route handlers either get deleted (gateway takes
over) or stay as a thin pass-through — not a wasted investment either way.**

### How the BFF reaches each backend service
- `auth-service`, `product-catalog-service`: already expose REST (`/v1/auth/*`,
  `/api/v1/*`) — route handlers call these directly with `fetch`, no codegen needed.
- `inventory-service`, `payment-service`: **gRPC only**, no REST surface (by design, per
  `design.md`). Route handlers need a Node gRPC client. Use `@grpc/grpc-js` +
  `ts-proto`-generated TypeScript stubs from the *same* `.proto` files already in
  `pkg/proto/inventory` and `pkg/proto/payment` — single source of truth, no manually
  re-typed request/response shapes drifting from the Go side.

## Preconditions
- Auth, Catalog, Inventory, Payment all exist and are healthy (Stages 1-4 ✅ — confirmed
  live in earlier sessions).
- Order Management does **not** exist yet (Stage 5, not started) — this gates which
  storefront features are buildable now vs. later (see phasing below).

## Repo layout

```
apps/
  web/
    app/
      (storefront)/            # customer-facing routes
      (seller)/                 # seller dashboard, behind role guard
      (admin)/                  # admin panel, behind role guard
      (warehouse)/              # warehouse management, behind role guard
      api/
        auth/...                 # BFF routes wrapping auth-service REST
        catalog/...               # BFF routes wrapping catalog-service REST
        inventory/...             # BFF routes wrapping inventory-service gRPC
        payment/...               # BFF routes wrapping payment-service gRPC (later)
    lib/
      grpc-clients/              # generated ts-proto stubs + thin client wrappers
      auth/                      # session cookie helpers, role-guard middleware
    middleware.ts                 # Next.js middleware: route-level role gating
    package.json
    next.config.ts
```

Lives at repo root as `apps/web`, sibling to `services/`, **not** inside the Go
workspace (`go.work` is untouched — this is a separate Node project, no Go build
implications).

## Tasks, phased by backend readiness

### Phase 1 — Foundation (buildable now)
- [ ] Scaffold `apps/web` with Next.js (App Router, TypeScript, Tailwind — confirm
      Tailwind is wanted or if a component library like shadcn/ui is preferred; default
      to Tailwind + shadcn/ui as the common modern pairing unless told otherwise).
- [ ] `lib/grpc-clients`: set up `ts-proto` codegen reading directly from
      `pkg/proto/inventory/inventory.proto` and `pkg/proto/payment/payment.proto` (path
      relative to repo root, e.g. `../../pkg/proto/...` from `apps/web`) — add an npm
      script `gen:proto` so these stay in sync with the Go-side `.proto` files instead of
      hand-copied.
- [ ] Auth flow: login/register pages calling `POST /api/auth/login` →
      `auth-service:8080/v1/auth/login`, storing the returned JWT in an `httpOnly`,
      `secure`, `sameSite=lax` cookie set by the route handler (never exposed to client
      JS). Add a `/api/auth/me` route the rest of the app uses to read the current
      session.
- [ ] `middleware.ts`: decode the session cookie's role claim (no signature
      verification needed client-side-equivalent here — the BFF route handlers
      re-validate via `auth-service`'s `ValidateToken` gRPC on every request that needs
      it, same as `product-catalog-service`'s existing `AuthMiddleware` pattern) and
      redirect unauthenticated/wrong-role users away from `(seller)`, `(admin)`,
      `(warehouse)` route groups.

### Phase 2 — Customer storefront (read-only parts buildable now)
- [ ] Category browse page → `GET /api/catalog/categories` → catalog's
      `/api/v1/categories` (already paginated/filterable from Stage 2).
- [ ] Product listing + detail pages → `GET /api/catalog/products`,
      `GET /api/catalog/products/[id]` → catalog's existing endpoints, including images
      (already real MinIO-backed URLs from Stage 2b — render directly, no extra proxying
      needed since those URLs are already public).
- [ ] Register/login pages (shared with Phase 1's auth flow).
- [ ] **Blocked until Stage 5 (Order Management) exists**: cart, checkout, order
      history, order status tracking. Don't stub these with fake data — leave them as a
      clearly-marked "coming soon" or simply omit the nav entries until Stage 5 lands, so
      the UI never lies about what the platform can actually do yet.

### Phase 3 — Seller dashboard (buildable now)
- [ ] Product CRUD (create/update/delete) → catalog's seller-role-protected endpoints.
      The BFF route handler must forward the session's JWT as `Authorization: Bearer` —
      catalog's own `AuthMiddleware` does the real role check; the BFF doesn't duplicate
      that logic, just passes the token through.
- [ ] SKU CRUD per product.
- [ ] Image upload — this is the one BFF route that's more than a thin proxy: it must
      accept the browser's `multipart/form-data`, then re-stream it server-side to
      catalog's `POST /api/v1/products/{id}/images` (Stage 2b's real upload endpoint,
      MinIO-backed) — confirm Next.js route handlers can re-stream a multipart body
      without fully buffering it first (Node's `fetch` with a `ReadableStream` body
      should work; verify, since this matters for the 5MB cap already enforced
      server-side at the catalog layer).
- [ ] Seller's own product list (filtered by `seller_id` — already supported by
      catalog's `ProductFilters.SellerID` from Stage 2).

### Phase 4 — Admin panel (buildable now)
- [ ] Category CRUD → catalog's admin-role-protected category endpoints.
- [ ] **Blocked until later stages**: order management view (Stage 5), payment
      reconciliation view (Stage 4 exists but has no list/search endpoint yet — only
      `GetTransaction` by ID; a `ListTransactions` RPC would need to be added to
      `payment-service` first if this is wanted — flagging as a gap, not building it
      speculatively here).

### Phase 5 — Warehouse management (buildable now, via gRPC BFF routes)
- [ ] Stock lookup → `GET /api/inventory/stock/[skuId]` → inventory's `GetStock` gRPC.
- [ ] Add stock → `POST /api/inventory/stock/[skuId]/add` → inventory's `AddStock` gRPC
      (the only "create" operation inventory currently exposes — see Stage 3's notes on
      this being a deliberately minimal CRUD).
- [ ] Reservation visibility: inventory has no "list active reservations" RPC yet — only
      per-reservation operations tied to a specific `reservation_id` a caller already
      has. A warehouse dashboard showing *all* open reservations would need a new
      `ListReservations`-style RPC added to `inventory-service` first. Flagging as a gap
      rather than building around it with a workaround query.

### Phase 6 — Role-based access control wiring
- [ ] Confirm the three non-customer surfaces map to the existing `auth-service` roles
      (`buyer`, `seller`, `admin`) from `db-design.md` — note that "warehouse management"
      isn't one of the three existing roles. Decide: is warehouse management an
      **admin-only** surface (simplest, no schema change), or does it need its own role
      (`warehouse_staff`)? **This needs a decision before Phase 5 ships** — recommend
      admin-only for now since there's no requirement yet for a distinct warehouse
      operator role, and adding one later is a small, additive migration.

## Out of scope (this stage)
- Checkout/cart/order-history UI — hard-blocked on Stage 5.
- Payment management UI beyond what Phase 4 already flags as gapped.
- Mobile app / React Native — not requested.
- SSR-driven SEO optimization, sitemap generation, structured data — can follow once the
  storefront's core pages exist; premature before there's real product data.
- Replacing this BFF with the real Stage 8 API Gateway — that's still Stage 8's job
  later; this stage's BFF is intentionally disposable/replaceable.

## Definition of done
- `apps/web` runs locally (`npm run dev`) against the existing Docker Compose backend
  stack (auth/catalog/inventory/payment all already running) with no CORS issues (since
  the browser only ever talks to the Next.js app's own origin).
- A seller can log in, create a product, add a SKU, and upload a real image, and see it
  appear correctly on the (read-only) storefront product page — full round trip through
  every existing backend capability this stage touches.
- An admin can log in and create/edit a category.
- A logged-in user with role `seller` is redirected away from `(admin)` routes, and
  vice versa — role gating verified live, not just written.
- Warehouse stock add/lookup works end-to-end through the gRPC BFF routes — verifies the
  `ts-proto` codegen path actually works, not just that it compiles.
- `go.work` / any Go service is **untouched** by this stage — confirm `go build ./...`
  across all services still passes after `apps/web` is added (it should be invisible to
  the Go toolchain entirely).
