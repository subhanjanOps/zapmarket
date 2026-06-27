# Smoke Test Bug Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all bugs and UX gaps found during the Stripe E2E payment smoke test.

**Architecture:** Six independent fixes across the Go inventory-service (seed SQL), Next.js buyer-ui (product name, add-to-cart, idempotency key, Buy Now qty), and one frontend-only cart cleanup. Each task is independently testable and deployable.

**Tech Stack:** Go 1.22, Next.js 14 (App Router), PostgreSQL, Zustand (cart store), Stripe.js

## Global Constraints

- No ORM — raw `database/sql` in Go services
- Buyer-UI: Next.js App Router, `"use client"` only where hooks required
- Cart store: `useCartStore` from `@/lib/cart` (Zustand + localStorage persist)
- API responses are always wrapped: `{ success: bool, data: T }` — never use top-level fields directly
- Do not add comments unless the reason is non-obvious

---

### Task 1: Seed inventory for all existing SKUs

**Why:** Every checkout fails with "no stock recorded for this sku" because the inventory DB only has 1 row. New SKUs created in product-catalog have no corresponding inventory row.

**Files:**
- Create: `services/inventory-service/db/seeds/001_seed_all_skus.sql`

**Interfaces:**
- Produces: all product-catalog SKUs have ≥100 units in warehouse `00000000-0000-0000-0000-000000000001`

- [ ] **Step 1: Create the seed SQL file**

```sql
-- services/inventory-service/db/seeds/001_seed_all_skus.sql
-- Inserts inventory rows for all SKUs that don't yet have one.
-- Safe to run repeatedly (ON CONFLICT DO NOTHING).
-- Run against the inventory DB: psql -U zapuser -d inventory -f 001_seed_all_skus.sql
--
-- NOTE: This requires the productcatalog DB to be on the same Postgres instance
-- so the cross-DB select works. Both DBs live in the same zapmarket-postgres container.

INSERT INTO inventory (sku_id, warehouse_id, qty_on_hand, qty_reserved, low_stock_threshold)
SELECT
    s.id::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    100,
    0,
    10
FROM dblink(
    'dbname=productcatalog user=zapuser password=zappass123 host=localhost',
    'SELECT id FROM skus WHERE deleted_at IS NULL'
) AS s(id text)
ON CONFLICT (sku_id, warehouse_id) DO NOTHING;
```

- [ ] **Step 2: Enable dblink extension and run the seed**

```bash
# Enable dblink (one-time, run inside the container)
docker exec zapmarket-postgres psql -U zapuser -d inventory -c "CREATE EXTENSION IF NOT EXISTS dblink;"

# Run the seed
docker exec -i zapmarket-postgres psql -U zapuser -d inventory < services/inventory-service/db/seeds/001_seed_all_skus.sql
```

Expected output: `INSERT 0 N` (where N = number of SKUs that had no inventory row)

- [ ] **Step 3: Verify**

```bash
docker exec zapmarket-postgres psql -U zapuser -d inventory \
  -c "SELECT COUNT(*) FROM inventory WHERE qty_on_hand = 100;"
```

Expected: a count matching the number of SKUs in product catalog.

- [ ] **Step 4: Commit**

```bash
git add services/inventory-service/db/seeds/001_seed_all_skus.sql
git commit -m "seed: add inventory rows for all existing SKUs (100 units each)"
```

---

### Task 2: Fix product name showing as "undefined" in cart

**Why:** The product detail page calls the catalog API which returns `{ success, data: { name, ... } }`, but the page uses `productRes.name` instead of `productRes.data?.name`. This means every cart item's name is `"undefined — <sku_code>"`.

**Files:**
- Modify: `services/buyer-ui/app/products/[id]/page.tsx` — unwrap `data` from API response

**Interfaces:**
- Consumes: `GET /v1/products/{id}` → `{ success: bool, data: { id, name, description, currency, ... } }`
- Produces: `productRes` variable holds the inner `data` object (not the envelope)

- [ ] **Step 1: Locate the three fetch calls and fix the unwrap**

In `services/buyer-ui/app/products/[id]/page.tsx`, lines 93–103, change:

```ts
// BEFORE
const [productRes, skusRes, imagesRes] = await Promise.all([
  fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } })
    .then((r) => (r.ok ? r.json() : null))
    .catch(() => null),
  fetch(`${GW}/v1/skus?product_id=${id}`, { next: { revalidate: 300 } })
    .then((r) => (r.ok ? r.json() : { data: [] }))
    .catch(() => ({ data: [] })),
  fetch(`${GW}/v1/products/${id}/images`, { next: { revalidate: 300 } })
    .then((r) => (r.ok ? r.json() : { data: [] }))
    .catch(() => ({ data: [] })),
]);
```

```ts
// AFTER
const [productEnv, skusRes, imagesRes] = await Promise.all([
  fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } })
    .then((r) => (r.ok ? r.json() : null))
    .catch(() => null),
  fetch(`${GW}/v1/skus?product_id=${id}`, { next: { revalidate: 300 } })
    .then((r) => (r.ok ? r.json() : { data: [] }))
    .catch(() => ({ data: [] })),
  fetch(`${GW}/v1/products/${id}/images`, { next: { revalidate: 300 } })
    .then((r) => (r.ok ? r.json() : { data: [] }))
    .catch(() => ({ data: [] })),
]);
const productRes = productEnv?.data ?? productEnv;
```

- [ ] **Step 2: Also fix `generateMetadata` which has the same issue**

In `services/buyer-ui/app/products/[id]/page.tsx`, lines 36–41, change:

```ts
// BEFORE
const r = await fetch(`${GW}/v1/products/${id}`, {
  next: { revalidate: 300 },
});
if (!r.ok) return { title: "Product" };
const p = await r.json();
return { title: p.name ?? "Product", description: p.description };
```

```ts
// AFTER
const r = await fetch(`${GW}/v1/products/${id}`, {
  next: { revalidate: 300 },
});
if (!r.ok) return { title: "Product" };
const env = await r.json();
const p = env?.data ?? env;
return { title: p.name ?? "Product", description: p.description };
```

- [ ] **Step 3: Verify in browser**

Navigate to `http://localhost:3000/products/121fcf1c-21b9-44b5-bbf6-3fdcc2f443ba`, click "Buy Now", check cart — item name should be `"Dhokra Metal Art Figurine Elephant — ARJ-N-030"` (not `"undefined — ARJ-N-030"`).

- [ ] **Step 4: Commit**

```bash
git add services/buyer-ui/app/products/\[id\]/page.tsx
git commit -m "fix(buyer-ui): unwrap data envelope from product API response so product name shows correctly in cart"
```

---

### Task 3: Fix "Add to cart" button on product listing page

**Why:** `ProductCard.tsx` renders the "+ Add to cart" button but its `onClick` only calls `e.preventDefault()` — no cart logic. Clicking it silently does nothing.

The fix: the listing page doesn't have a selected SKU (only product-level data), so add-to-cart needs to navigate to the product detail page where the user can pick a SKU. Change the button to navigate to `/products/${product.id}` instead of pretending to add to cart.

**Files:**
- Modify: `services/buyer-ui/components/ProductCard.tsx`

- [ ] **Step 1: Replace the dead button with a navigation link**

In `services/buyer-ui/components/ProductCard.tsx`, replace lines 227–234:

```tsx
// BEFORE
<motion.button
  onClick={e => e.preventDefault()}
  className="w-full h-8 mt-2.5 border border-[#E8E8E8] rounded-md text-xs font-medium text-[#111111] hover:bg-[#111111] hover:text-white hover:border-[#111111] transition-all duration-200 flex items-center justify-center gap-1.5"
  whileTap={{ scale: 0.97 }}
>
  <ShoppingCart className="h-3.5 w-3.5" />
  + Add to cart
</motion.button>
```

```tsx
// AFTER
<Link
  href={`/products/${product.id}`}
  onClick={e => e.stopPropagation()}
  className="w-full h-8 mt-2.5 border border-[#E8E8E8] rounded-md text-xs font-medium text-[#111111] hover:bg-[#111111] hover:text-white hover:border-[#111111] transition-all duration-200 flex items-center justify-center gap-1.5"
>
  <ShoppingCart className="h-3.5 w-3.5" />
  + View &amp; Add
</Link>
```

Note: The outer `<Link href={/products/${product.id}}>` already wraps the card; nesting another link inside is invalid HTML. Remove the outer `<Link>` wrapper and make the entire card use `onClick` navigation instead, keeping the inner link for the button only. The full updated return for the card body:

```tsx
return (
  <motion.div
    className={cn(
      "bg-white border border-[#E8E8E8] rounded-lg overflow-hidden transition-shadow duration-300 cursor-pointer",
      hovered
        ? "shadow-[0_4px_12px_rgba(0,0,0,0.08)]"
        : "shadow-sm"
    )}
    onMouseEnter={handleMouseEnter}
    onMouseLeave={handleMouseLeave}
    whileHover={{ y: -3 }}
    transition={{ duration: 0.25, ease: [0.25, 0.1, 0.25, 1] }}
    onClick={() => window.location.href = `/products/${product.id}`}
  >
    {/* Image area — square aspect ratio */}
    <div className="relative bg-[#F6F6F6] overflow-hidden" style={{ aspectRatio: "1/1" }}>
      {allImages[0] ? (
        <Image
          src={allImages[imgIdx]?.url ?? allImages[0].url}
          alt={product.name}
          fill
          sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
          className={cn(
            "object-contain transition-transform duration-300",
            hovered ? "scale-[1.04]" : "scale-100"
          )}
          priority={priority}
        />
      ) : (
        <div className="absolute inset-0 flex items-center justify-center">
          <Zap className="h-10 w-10 text-[#D0D0D0]" />
        </div>
      )}

      {/* Top-left badges */}
      <div className="absolute top-2 left-2 flex flex-col gap-1">
        {product.discount_percent && product.discount_percent > 0 && (
          <span className="px-1.5 py-0.5 bg-[#E91E8C] text-white text-[10px] font-bold rounded-sm tabular-nums leading-none">
            -{product.discount_percent}%
          </span>
        )}
        {product.is_new && (
          <span className="px-1.5 py-0.5 bg-[#111111] text-white text-[10px] font-bold rounded-sm leading-none">
            NEW
          </span>
        )}
      </div>

      {/* Wishlist button */}
      <motion.button
        onClick={e => {
          e.stopPropagation();
          setWishlist(w => !w);
        }}
        className="absolute top-2 right-2 h-7 w-7 rounded-full bg-white border border-[#E8E8E8] flex items-center justify-center shadow-sm"
        whileTap={{ scale: 0.9 }}
        aria-label={wishlist ? "Remove from wishlist" : "Add to wishlist"}
      >
        <Heart
          className={cn(
            "h-3.5 w-3.5 transition-colors",
            wishlist ? "fill-[#E91E8C] text-[#E91E8C]" : "text-[#999999]"
          )}
        />
      </motion.button>

      {/* Image indicator dots */}
      {hasMultiple && hovered && (
        <div className="absolute bottom-2 left-1/2 -translate-x-1/2 flex items-center gap-1">
          {allImages.slice(0, 5).map((_, i) => (
            <button
              key={i}
              onMouseEnter={e => {
                e.stopPropagation();
                if (hoverTimer.current) clearInterval(hoverTimer.current);
                setImgIdx(i);
              }}
              className={cn(
                "rounded-full transition-all duration-200",
                i === imgIdx ? "h-1.5 w-4 bg-white" : "h-1.5 w-1.5 bg-white/60"
              )}
            />
          ))}
        </div>
      )}
    </div>

    {/* Card body */}
    <div className="px-3 pt-3 pb-3.5">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-[#999999] leading-none truncate">
        {product.brand ?? product.category_name ?? ""}
      </p>
      <h3 className="text-sm font-medium text-[#111111] line-clamp-2 leading-snug mt-1">
        {product.name}
      </h3>

      {product.rating !== undefined && (
        <div className="flex items-center gap-1 mt-1.5">
          {Array.from({ length: 5 }).map((_, i) => (
            <Star
              key={i}
              className={cn(
                "h-3 w-3",
                i < Math.round(product.rating!)
                  ? "fill-[#D97706] text-[#D97706]"
                  : "fill-[#E8E8E8] text-[#E8E8E8]"
              )}
            />
          ))}
          {product.review_count !== undefined && (
            <span className="text-[10px] text-[#999999] ml-0.5">
              ({product.review_count.toLocaleString("en-IN")})
            </span>
          )}
        </div>
      )}

      <div className="flex items-baseline gap-1.5 mt-2">
        <span className="text-base font-bold text-[#111111] tabular-nums">
          {formatPrice(discountedPrice)}
        </span>
        {product.discount_percent && product.discount_percent > 0 && (
          <span className="text-xs text-[#999999] line-through tabular-nums">
            {formatPrice(resolvedPrice)}
          </span>
        )}
      </div>

      <p className="mt-1 text-[11px] leading-none">
        {product.free_shipping ? (
          <span className="text-[#16A34A]">Free delivery</span>
        ) : (
          <span className="text-[#999999]">Delivery charges apply</span>
        )}
      </p>

      <motion.button
        onClick={e => {
          e.stopPropagation();
          window.location.href = `/products/${product.id}`;
        }}
        className="w-full h-8 mt-2.5 border border-[#E8E8E8] rounded-md text-xs font-medium text-[#111111] hover:bg-[#111111] hover:text-white hover:border-[#111111] transition-all duration-200 flex items-center justify-center gap-1.5"
        whileTap={{ scale: 0.97 }}
      >
        <ShoppingCart className="h-3.5 w-3.5" />
        + Add to cart
      </motion.button>
    </div>
  </motion.div>
);
```

Also remove the `Link` import if it's no longer used elsewhere in the file (check — it was only used for the outer wrapper).

- [ ] **Step 2: Verify**

Go to `http://localhost:3000/products`, click "+ Add to cart" on any card. Should navigate to that product's detail page.

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/components/ProductCard.tsx
git commit -m "fix(buyer-ui): product listing add-to-cart now navigates to product detail page instead of silently no-op"
```

---

### Task 4: Fix "Buy Now" accumulating cart quantity across clicks

**Why:** `handleBuyNow` in `AddToCartButton.tsx` calls `addItem` which increments qty if the item already exists. Clicking "Buy Now" twice (or after a failed checkout) doubles the quantity. Buy Now should ensure exactly `qty` units of this SKU are in cart, not add on top.

**Files:**
- Modify: `services/buyer-ui/components/AddToCartButton.tsx`
- Consumes: `useCartStore` from `@/lib/cart` — needs `updateQty` and `items` in addition to `addItem`

- [ ] **Step 1: Update `AddToCartButton` to use set-not-add for Buy Now**

Replace lines 39–51 in `services/buyer-ui/components/AddToCartButton.tsx`:

```tsx
// BEFORE
function handleBuyNow() {
  if (!sku) return;
  for (let i = 0; i < qty; i++) {
    addItem({
      skuId: sku.id,
      name: `${productName} — ${sku.sku_code}`,
      image: productImage,
      price: sku.price_amount,
      currency: sku.currency,
    });
  }
  router.push("/checkout");
}
```

```tsx
// AFTER
const { addItem, updateQty, items } = useCartStore((s) => ({
  addItem: s.addItem,
  updateQty: s.updateQty,
  items: s.items,
}));

function handleBuyNow() {
  if (!sku) return;
  const existing = items.find((i) => i.skuId === sku.id);
  if (existing) {
    updateQty(sku.id, qty);
  } else {
    addItem({
      skuId: sku.id,
      name: `${productName} — ${sku.sku_code}`,
      image: productImage,
      price: sku.price_amount,
      currency: sku.currency,
      qty,
    });
  }
  router.push("/checkout");
}
```

Also update the existing `addItem` selector at line 17 to remove the standalone declaration (it's now part of the destructure above):

```tsx
// Remove this line:
const addItem = useCartStore((s) => s.addItem);

// Replace with (at line 17):
const { addItem, updateQty, items } = useCartStore((s) => ({
  addItem: s.addItem,
  updateQty: s.updateQty,
  items: s.items,
}));
```

- [ ] **Step 2: Verify**

Navigate to the product detail page. Click "Buy Now" twice. Cart badge should show `1` (not `2`). Checkout summary should show qty 1.

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/components/AddToCartButton.tsx
git commit -m "fix(buyer-ui): Buy Now sets cart qty instead of incrementing, preventing duplicate quantities"
```

---

### Task 5: Clear idempotency key after non-retryable checkout errors

**Why:** When an order fails (e.g. "failed to reserve stock"), the idempotency key is stored in `sessionStorage`. The next "Place Order" click replays the same failed order from DB (idempotent replay returns PENDING), making retries silently fail. The key should be cleared so a fresh order is created on retry.

**Files:**
- Modify: `services/buyer-ui/app/checkout/page.tsx`

**Interfaces:**
- Consumes: `POST /api/proxy/v1/orders` — returns 201 on success, 409 on idempotent replay, 5xx on failure
- The idempotency key is in `sessionStorage` under key `"checkout_idempotency_key"` and in `idempotencyKey.current`

- [ ] **Step 1: Find the error handling block and clear the key on non-retryable failure**

In `services/buyer-ui/app/checkout/page.tsx`, after line 253 (`const err = await res.json()...`), add key clearing before the `setError` call:

```tsx
// BEFORE (lines 253–259):
const err = await res.json().catch(() => ({}));
setError(
  (err as Record<string, string>).error ??
    (err as Record<string, string>).message ??
    "Failed to place order. Please try again."
);
```

```tsx
// AFTER:
const err = await res.json().catch(() => ({}));
// Clear the idempotency key so the user can retry with a fresh order
// rather than replaying the same failed one.
sessionStorage.removeItem("checkout_idempotency_key");
idempotencyKey.current = "";
setError(
  (err as Record<string, string>).error ??
    (err as Record<string, string>).message ??
    "Failed to place order. Please try again."
);
```

Also clear in the `catch` block (line 260–262):

```tsx
// BEFORE:
} catch {
  setError("Network error. Please check your connection.");
}
```

```tsx
// AFTER:
} catch {
  sessionStorage.removeItem("checkout_idempotency_key");
  idempotencyKey.current = "";
  setError("Network error. Please check your connection.");
}
```

- [ ] **Step 2: Verify**

In checkout, submit an order with an invalid setup (e.g. temporarily change the API URL). Observe error toast. Without refreshing, fill correct details and resubmit — a new order ID should appear in backend logs (not an idempotent replay).

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/app/checkout/page.tsx
git commit -m "fix(buyer-ui): clear idempotency key on checkout failure so retries create a new order"
```

---

### Task 6: Fix order detail page redirect (orders page shows 404 after successful order)

**Why:** After a successful order, the checkout page redirects to `/account/orders/${orderId}?new=1`. The order detail page calls `apiFetch` which uses `GATEWAY_URL` (docker-internal URL `http://zapmarket-api-gateway:8000`) from env. In local dev (`next dev`), this env var is only set in docker-compose, not in the local `.env.local`. If `GATEWAY_URL` is unset, it falls back to `NEXT_PUBLIC_GATEWAY_URL` or `http://localhost:8000`. Check that the right URL is being used from the server-side render.

**Files:**
- Modify: `services/buyer-ui/.env.local` — add `GATEWAY_URL`
- Modify: `services/buyer-ui/.env.example` — document `GATEWAY_URL`

**Interfaces:**
- `apiFetch` in `services/buyer-ui/lib/api.ts` uses `process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000"`
- The order detail page at `services/buyer-ui/app/account/orders/[id]/page.tsx` calls `apiFetch('/v1/orders/${id}', { headers: { Authorization: Bearer ${token} } })`

- [ ] **Step 1: Check what the order detail page actually receives**

```bash
# Grab the buyer token from the browser (after login), then:
TOKEN="<paste token from browser devtools Application > Cookies > buyer_token>"
curl -v "http://localhost:8000/v1/orders/6e318dab-941e-4f45-bf6a-5fe5d19e1184" \
  -H "Authorization: Bearer $TOKEN"
```

Expected: `200 { success: true, data: { id, status: "CONFIRMED", ... } }`

- [ ] **Step 2: Add `GATEWAY_URL` to `.env.local` for local dev**

In `services/buyer-ui/.env.local`, add:

```
GATEWAY_URL=http://localhost:8000
```

- [ ] **Step 3: Add to `.env.example`**

In `services/buyer-ui/.env.example`, add:

```
# Server-side gateway URL (used in RSC fetches — set to http://localhost:8000 for local dev)
GATEWAY_URL=http://localhost:8000
```

- [ ] **Step 4: Fix order detail page to unwrap the API envelope**

The order API also returns `{ success, data: { id, status, items, ... } }`. In `services/buyer-ui/app/account/orders/[id]/page.tsx`, line 60–64:

```ts
// BEFORE
order = await apiFetch<Record<string, unknown>>(
  `/v1/orders/${id}`,
  { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" }
);
```

```ts
// AFTER
const env = await apiFetch<{ data: Record<string, unknown> }>(
  `/v1/orders/${id}`,
  { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" }
);
order = env?.data ?? null;
```

- [ ] **Step 5: Restart Next.js dev server and verify**

```bash
# In the buyer-ui directory:
# Stop the dev server (Ctrl+C) and restart:
cd services/buyer-ui
npm run dev
```

Then place a fresh order and confirm the browser lands on `/account/orders/${orderId}?new=1` with the green "Order placed successfully!" banner and status CONFIRMED.

- [ ] **Step 6: Commit**

```bash
git add services/buyer-ui/.env.local services/buyer-ui/.env.example
git add services/buyer-ui/app/account/orders/\[id\]/page.tsx
git commit -m "fix(buyer-ui): add GATEWAY_URL to env.local and unwrap order API envelope so order detail page renders after checkout"
```

---

## Self-Review

**Spec coverage:**
1. ✅ No inventory seeded → Task 1
2. ✅ Product name undefined → Task 2
3. ✅ Add to cart on listing page broken → Task 3
4. ✅ Buy Now qty accumulation → Task 4
5. ✅ Idempotency key not cleared on failure → Task 5
6. ✅ Order detail page redirect issue → Task 6

**Skipped (low priority, no code change warranted):**
- Footer 404s — placeholder pages, not bugs
- Stripe HTTPS warning — expected in local dev
- `payment_method` field in request body — unused but harmless

**Placeholder scan:** None found. All steps have exact code.

**Type consistency:** `useCartStore` destructure pattern consistent across Tasks 4. `apiFetch<{ data: T }>` pattern in Task 6 matches the helper's generic signature.
