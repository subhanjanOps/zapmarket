# ZapMarket — Feature Guide for All Users

This is a single reference of everything ZapMarket currently does, organized
by who uses it: **buyers**, **sellers**, **delivery agents**, **admins**, and
**backoffice/support staff**. For step-by-step buyer/seller instructions see
`docs/user-guide.md` and `docs/new-features-guide.md`; this document is the
full feature inventory across every app.

---

## Buyers (`buyer-ui`)

### Account
- Sign up with email/password (4-step wizard: basic info, email/phone OTP
  verification, profile, terms acceptance) or sign in with Google/Facebook
- Sign in with email/password or OAuth
- View profile: name, email, verification status

### Browsing & search
- Home page with featured products and deals
- Browse all products, filter by category
- Product detail pages: images, description, variants (size/colour/etc via
  SKUs), price, stock indicator, and buyer reviews with a verified-purchase
  badge and average rating

### Cart
- Add a specific SKU to cart; cart persists in the browser for guests
- Once signed in, the cart is also saved server-side and merges automatically
  across devices (cart-service)
- Update quantities, remove items (up to 50 line items)

### Wishlist
- Save products for later from any product page
- View, and (via the API) remove or clear saved items

### Checkout & payment
- Review order summary, enter a coupon/promo code for a discount
- Pay by card via Stripe (card details never touch ZapMarket's servers)
- Place order; see confirmation or a clear error on decline

### Orders
- View order history with status: `PENDING → RESERVED → CONFIRMED →
  CANCELLED`
- Cancel an order while it's `PENDING`/`RESERVED`
- Track a shipped order: carrier, tracking link, delivery timeline, and proof
  of delivery once delivered

### Reviews & returns
- Write a star rating (1–5) + written review + photos for delivered items
  (goes through moderation before appearing publicly)
- Request a return on a delivered, in-window item with a reason and photos
- Track return status: `Requested → Approved/Rejected → picked up →
  refunded`; a reverse pickup is scheduled automatically once approved
- Refunds are issued to the original payment method

### Notifications
- Email notifications for order confirmed/cancelled, payment
  successful/failed, and refund processed

---

## Sellers (`seller-ui`)

### Account
- Seller registration (same 4-step wizard as buyers, plus store details);
  new seller accounts enter a **pending review** state until approved

### Product catalog
- Create/edit products: name, description, category, custom attributes
- Add one or more SKUs per product (price, variant attributes, stock-keeping
  code); deactivate a SKU without deleting it
- Upload and reorder product images
- Move a product between `DRAFT` and `ACTIVE` status
- Bulk import products/categories via CSV upload (in progress — backend
  scaffold exists, not yet exposed in the dashboard)

### Orders
- View orders containing the seller's products, with buyer details and line
  items

### Coupons & promotions
- Coupon codes and time-boxed flash sales apply automatically at buyer
  checkout when eligible

### Earnings & payouts
- Every sale credits the seller's running balance net of platform commission,
  TDS, and GST on commission; refunds debit it back
- Add and verify a bank account or UPI ID for payouts
- Payouts run on a weekly schedule once the balance and verified bank details
  are in place; each payout is tracked through a pending → completed/failed
  state so a failure never results in a silent double-payment

### Returns
- Approve or reject buyer return requests (staff-role gated); approving
  automatically schedules a reverse pickup via Logistics

---

## Delivery agents & logistics (internal, via `logistics-service`)

- Agents are assigned to shipments by staff (`admin`/`seller` role)
- Record delivery attempts (success, or scheduled reattempt after failure, up
  to 3 attempts before a shipment is marked undelivered)
- Capture proof of delivery (signature or photo) and, for COD orders, record
  cash collected and reconcile it
- Reverse pickups are created automatically when a return is approved
- Carrier tracking updates (e.g. from Shiprocket) flow in via a
  signature-verified webhook and are visible to the buyer on their order page

---

## Admins (`admin-ui`)

- Platform-wide dashboard: currency management (enable/disable currencies),
  blocklist management (block abusive users/IPs), general operational
  controls
- Full staff-role access to settlement (any seller's balance/bank details),
  return approvals, and logistics operations alongside sellers

---

## Backoffice / support staff (`backoffice-ui`)

- Manage products, orders, and users across the platform
- Content moderation (e.g. review moderation queue)
- Bulk CSV import/export tooling for operational data
- Command palette and dashboards for day-to-day support operations

---

## Cross-cutting platform features

- **Authentication:** JWT-based sessions with OAuth2 (Google/Facebook) login,
  RBAC roles (`buyer`, `seller`, `admin`); every mutation endpoint across
  cart, wishlist, reviews/returns, promotions, settlement, and logistics
  requires a valid token, with staff-only actions additionally role-gated
- **Idempotency:** checkout, payment capture, coupon redemption, and seller
  ledger postings are all protected against duplicate execution (client
  idempotency keys, DB transactions with row locking, or unique constraints
  that make retries and at-least-once event delivery safe)
- **Async, event-driven fulfilment:** order placement, payment, inventory
  reservation, settlement, and notifications are connected via Kafka so a
  slow or failing downstream service doesn't block the buyer's checkout
- **Multi-currency support:** prices and payouts are currency-aware
  (`currency-service` manages exchange rates and enabled currencies)

---

## Known in-progress areas

- Seller bulk import (CSV upload UI) is not yet wired to the dashboard
- Buyer profile editing (name/address/photo after registration) is not yet
  available
- Distributed tracing across services and TLS-by-default on internal
  connections are platform hardening work still in progress (see
  `reviews/2026-07-01-new-services-review.md`)
