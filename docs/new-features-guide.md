# ZapMarket New Features Guide

This supplements `docs/user-guide.md` with features from the services added
on 2026-07-01: server-side cart, wishlist, coupons, reviews & returns, and
order tracking. Some of these are still being finished on the backend (see
the note at the end of each section) — behaviour may change before release.

---

## Buyer: cart that follows you

Previously the cart lived only in your browser. It now also saves to your
account once you're logged in:

- Add items to your cart while signed out — they're kept on your device.
- Sign in, and those items are merged into your account cart automatically.
- From then on, your cart is the same whether you're on your phone or your
  laptop.

> **In progress:** the merge and cross-device sync are implemented on the
> backend, but the service does not yet require you to be signed in to read
> or change a cart — treat cart contents as provisional until this is
> hardened.

---

## Buyer: wishlist

- On any product page, click **Save for later** (heart icon) to add it to
  your wishlist.
- View your saved items under **Account → Wishlist**.
- Saved items are not reserved — if stock runs out, remove-from-wishlist is
  the only action available; add-to-cart still depends on live stock.

> **In progress:** only "add to wishlist" is available right now; viewing,
> removing, and clearing your wishlist are not yet exposed in the UI.

---

## Buyer: coupons and flash sales

- At checkout, enter a coupon code in the **Promo code** field and click
  **Apply**. Valid codes reduce your order total before payment.
- Flash sales are shown directly on eligible product listings with a
  countdown and a struck-through original price — no code needed.
- Each coupon has its own rules: a minimum order amount, an expiry date, and
  a limit on how many times it can be used in total or per customer. If a
  code doesn't apply, the checkout page will tell you why.

> **In progress:** redemption limits are enforced per request but not yet
> atomically, so under very high concurrency a coupon could be used slightly
> more than its stated limit. This does not affect the price you're charged.

---

## Buyer: reviews

- After an order is marked **Delivered**, go to **Account → Orders → [order]**
  and click **Write a review** next to any item.
- Reviews from verified purchases are marked **Verified Purchase**.
- Star rating (1–5) is required; a written review and photos are optional.
- New reviews go through moderation before appearing on the product page.

---

## Buyer: returns and refunds

1. From **Account → Orders → [order]**, click **Request return** on an
   eligible item (only available for delivered orders, within the return
   window).
2. Choose a reason, add a description, and optionally attach photos.
3. Track the status of your request: **Requested → Approved/Rejected →
   (if approved) picked up → refunded**.
4. Once a return is approved, a reverse pickup is scheduled automatically —
   you'll be notified of the pickup window.
5. Refunds are issued to your original payment method once the returned item
   is received and inspected.

> **In progress:** the buyer-facing return status screen currently reads
> directly from the service; approvals are still a manual seller/admin
> action without extra confirmation steps, so double-check before approving
> or rejecting a request if you're on the seller side.

---

## Buyer: order tracking

Once a seller ships your order, the order detail page shows:

- Carrier name and tracking link (if the carrier is integrated)
- A timeline of tracking events (dispatched, in transit, out for delivery,
  delivered)
- Proof of delivery (signature or photo) once delivered, where available

## Seller: getting paid

- Every sale credits your seller balance, shown under
  **Dashboard → Earnings**, net of ZapMarket's commission.
- Refunds on your orders debit the same balance.
- Add and verify a bank account or UPI ID under
  **Dashboard → Earnings → Payout details** to receive payouts.
- Payouts run on a weekly schedule once your balance and verified bank
  details are in place.

> **In progress:** payout initiation does not yet have duplicate-payment
> protection at the infrastructure level — if you notice a payout appear
> twice, contact support immediately rather than spending it, and do not
> rely on the payout history as final until this is fixed.

## Seller: bulk product import

Instead of adding products one at a time, sellers can upload a CSV of
products/categories under **Dashboard → Products → Bulk import**. A
background job processes the file and reports how many rows succeeded or
failed.

> **In progress:** this feature is still under active development on the
> backend and is not yet available in the seller dashboard.
