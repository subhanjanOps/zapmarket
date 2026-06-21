# Phase 7 — Seller & Buyer UX

**Goal:** Complete the seller dashboard and build the buyer-facing storefront. Depends on Phases 2 (events flowing), 3 (auth), and 6 (real payments).

---

## 7.1 Seller Dashboard Completions

### 7.1.1 Order Management

The seller-ui `/dashboard/orders` page exists but needs:

- **Order detail page** (`/dashboard/orders/[id]`): show buyer info (anonymized — only first name + order ID), line items, payment status, shipment status.
- **Fulfillment actions**: "Mark as Shipped" button → calls `PATCH /v1/orders/:id/ship` with `{tracking_number, carrier}`. Triggers `order.shipped` event → buyer notification.
- **Refund initiation**: seller can request a refund for a CONFIRMED order → creates a `REFUND_REQUESTED` state, admin approves → triggers `POST /v1/payments/:id/refund`.

### 7.1.2 SKU / Inventory Management

The `/dashboard/skus` page exists (basic toggle). Extend with:

- **Stock level display**: call `GET /v1/inventory/stock?sku_id=` for each SKU on the page.
- **Restock inline**: input + "Add Stock" button → `POST /v1/inventory/stock/add`.
- **Low-stock alerts**: badge on SKUs with `qty_available < low_stock_threshold`.
- **Bulk stock update**: upload CSV with `sku_id, qty` via import-service (Phase 4 extension).

### 7.1.3 Analytics Dashboard

Replace the current static stat cards with real data:

- **Revenue over time**: line chart — call `GET /v1/orders/seller/stats?from=&to=&group_by=day`.
- **Top products**: table of top 5 products by confirmed order count this month.
- **Order funnel**: CREATED → PENDING_PAYMENT → CONFIRMED → SHIPPED conversion rates.

Requires new analytics endpoints in `order-management-service` (simple GROUP BY queries, no OLAP needed at this scale).

### 7.1.4 Notifications Inbox

Real-time notification panel in the seller sidebar. The `notification-service` currently logs notifications but doesn't persist or deliver them to a UI.

Plan:
1. Add `notifications` table in a new `notifications` DB (or extend `notification-service`).
2. `notification-service` writes to this table in addition to (eventually) email.
3. New endpoint `GET /v1/notifications?unread=true&limit=20`.
4. Seller-ui polls every 30 seconds; bell icon shows unread count.
5. `POST /v1/notifications/:id/read` marks as read.

---

## 7.2 Buyer-UI (New Service)

A new `buyer-ui` Next.js service on port 3004. This is the public storefront.

### Pages

| Route | Description |
|-------|-------------|
| `/` | Homepage: featured products, top categories |
| `/search` | Search results page (Phase 5 search endpoint) |
| `/products/[slug]` | Product detail: images, description, SKU picker, Add to Cart |
| `/cart` | Cart (client-side state, persisted to localStorage + server on login) |
| `/checkout` | Address + payment flow (Razorpay widget from Phase 6) |
| `/orders` | Buyer's order history |
| `/orders/[id]` | Order detail + tracking |
| `/account` | Profile, password change |

### Auth for buyers

Buyers use the existing `auth-service` with `role=buyer`. The `buyer-ui` gets its own BFF pattern:
- `gw_token` / `gw_auth_hint` cookies (already partially present in admin-ui's hint cookie naming)
- `POST /api/auth/login`, `POST /api/auth/register`, `POST /api/auth/logout`, `POST /api/auth/refresh`
- middleware.ts: protect `/orders`, `/account`, `/checkout`

### Product detail page

```
GET /v1/products/slug/:slug            → product info + description
GET /v1/products/:id/skus              → SKU list with pricing
GET /v1/inventory/stock?sku_id=        → availability per SKU
GET /v1/products/:id/media             → image list from MinIO
```

### Cart

- Stored in localStorage (anonymous cart)
- On login: merge localStorage cart with server-side cart
- Server-side cart: `POST /v1/cart` (new endpoint, or handled fully client-side for v1)

### Checkout flow (with Razorpay)

```
1. Review cart → click "Place Order"
2. POST /api/proxy/v1/orders → {order_id, gateway_order_id, gateway_key, amount}
3. Load Razorpay JS, open modal
4. On success: poll GET /v1/orders/:id until CONFIRMED, show success page
5. On failure: show error, keep cart intact
```

---

## 7.3 Order Lifecycle Additions

New order statuses needed for full lifecycle:

```
CREATED → PENDING_PAYMENT → CONFIRMED → PROCESSING → SHIPPED → DELIVERED → COMPLETED
                         ↘ PAYMENT_FAILED
                         ↘ CANCELLED (before SHIPPED)
```

New events:
- `order.shipped` — seller marks as shipped (with tracking)
- `order.delivered` — delivery service webhook or manual confirm
- `order.completed` — buyer confirms receipt (or auto after 7 days)

New notification templates in `notification-service`:
- "Your order has shipped" (with tracking number)
- "Your order has been delivered"

---

## 7.4 Review & Ratings

Post-delivery, buyers can leave a review (1–5 stars + text) for products they've purchased.

**Schema:**

```sql
CREATE TABLE product_reviews (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  UUID NOT NULL REFERENCES products(id),
    buyer_id    UUID NOT NULL,
    order_id    UUID NOT NULL,
    rating      SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    body        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, product_id)   -- one review per purchased product per order
);
```

**Aggregate rating** (`avg_rating`, `review_count`) stored as denormalized columns on `products`, updated by trigger or by the review service after insert.

**Review moderation:** backoffice-ui admin can flag/remove reviews.

---

## 7.5 Shipment Integration

For v1, sellers enter tracking numbers manually. For v2, integrate with Shiprocket or Delhivery:

- After `order.shipped` event, call the courier's API to create a shipment.
- Webhook from courier on delivery → `order.delivered` event.
- Tracking URL displayed to buyer in order detail page.

---

## Acceptance criteria

- Buyer can discover, add to cart, and complete a purchase end-to-end without seller or admin involvement
- Seller receives `order.created` notification within 10 seconds of buyer checkout
- Seller can mark as shipped with tracking number; buyer receives email notification
- Review can only be left after order is DELIVERED; one review per ordered product
- Buyer order history loads in < 500ms

---

## Estimated effort

6–8 weeks (buyer-ui: 3 weeks; seller dashboard completions: 2 weeks; order lifecycle + reviews: 2 weeks; notifications inbox: 1 week).
