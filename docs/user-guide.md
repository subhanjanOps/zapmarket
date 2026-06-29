# ZapMarket User Guide

---

## What is ZapMarket?

ZapMarket is an online marketplace where buyers can browse products across
categories, add them to a cart, and pay by card. Sellers list products and
fulfil orders through a separate seller dashboard.

---

## Signing up

### For buyers

1. Go to **zapmarket.com → Sign up → Buyer**.
2. **Step 1 — Basic info**: enter your name, email address, phone number,
   and a password (minimum 8 characters). Click **Continue**.
3. **Step 2 — Verification**: a 6-digit code is sent to your email address
   automatically. Enter it to verify your email. A separate code can be sent
   to your phone number by clicking **Send SMS code**.
   - You may click **Skip for now** and complete verification later, but some
     features may be restricted until your email is verified.
4. **Step 3 — Profile**: optionally add your date of birth (must be 18 or
   older to create an account), gender, a profile photo (JPEG or PNG, max
   5 MB), and a delivery address.
5. **Step 4 — Review**: check your details and tick the box to accept the
   Terms of Service and Privacy Policy, then click **Complete registration**.

You are logged in immediately after step 4 and taken to the home page.

> If you leave the process before step 4, your account is created but login
> is blocked until you complete the remaining steps. Return to
> `/register/buyer` to continue.

### For sellers

The seller registration follows the same 4-step flow, with an additional
**Store details** section in step 3 (store name, tagline, and category).
After step 4 your account enters a **pending review** state. You will be
notified once the ZapMarket team approves your seller account.

### Sign in with Google or Facebook

Click **Continue with Google** or **Continue with Facebook** on the login
page. You will be redirected to complete OAuth consent and returned to
ZapMarket automatically. These flows bypass the registration wizard.

---

## Logging in

Go to **Sign in**, enter your email and password, and click **Sign in**.

If your account was created via Google or Facebook, use those buttons instead
of email/password.

---

## Buyer: browsing and search

- The **home page** shows featured products and deals.
- Use the **Products** page to browse all available items. Filter by
  category using the sidebar.
- Click any product to see its detail page: images, description, available
  variants (sizes, colours), price, and a stock indicator.

---

## Buyer: cart

- On any product detail page, select a variant (SKU) and click **Add to cart**.
- Your cart is saved in your browser. It persists across page refreshes but
  is not synced across devices.
- Go to **Cart** to review items, adjust quantities, or remove items.
- The cart holds a maximum of 50 line items.

---

## Buyer: checkout and payment

1. From the cart, click **Checkout**.
2. Review your order summary.
3. Enter your card details in the card form. ZapMarket uses **Stripe** to
   process payments — your card number is handled directly by Stripe and
   never touches ZapMarket's servers.
4. Click **Place order**.

If payment is successful, you are shown an order confirmation and the order
appears in your account.

**If payment fails**, you will see an error message. Your cart is unchanged
and you can try again with a different card.

> **Note on test/dev environments**: if the site is running without real
> Stripe credentials, any card number is accepted except amounts ending in
> ₹X.13 (or the equivalent in your currency), which are deliberately failed
> to test the error path.

---

## Buyer: order history

Go to **Account → Orders** to see all your past and current orders.

Each order shows its current status:

| Status | Meaning |
|---|---|
| PENDING | Order placed, processing |
| RESERVED | Stock held, payment being processed |
| CONFIRMED | Payment captured, order confirmed |
| CANCELLED | Order was cancelled |

Click an order to see its line items, amounts, and full status.

**Cancelling an order**: click **Cancel** on an order that is in PENDING or
RESERVED status. Orders in CONFIRMED status cannot be self-cancelled from
the UI at this time.

---

## Buyer: account and profile

Go to **Account** to view your profile. You can see your registered name,
email address, and verification status.

Profile editing (updating name, address, or photo after registration) is in
progress and not yet available.

---

## Seller: dashboard

Once your seller account is approved, log in at `seller-ui` (port 3002 on
self-hosted installs, or the seller subdomain on the hosted version).

### Listing products

1. Go to **Dashboard → Products → New product**.
2. Fill in the product name, description, category, and attributes.
3. Add at least one SKU with a price and variant attributes (e.g. size, colour).
4. Upload product images.
5. Save. The product starts in **DRAFT** status. Change it to **ACTIVE** to
   make it visible to buyers.

### Managing SKUs and images

- Open a product from the products list to edit its SKUs or images.
- Images can be reordered by position.
- A SKU can be deactivated without deleting it.

### Viewing orders

Go to **Dashboard → Orders** to see orders that include your products. Click
an order to see buyer details and line items.

Order fulfilment workflow (shipping updates, tracking) is in progress and
not yet available in the dashboard.

### Inventory

Stock management is not yet available through the seller dashboard. Inventory
levels are managed internally. Contact your ZapMarket administrator to
adjust stock quantities.

---

## Payments

ZapMarket uses **Stripe** for card payments. Accepted card types are
determined by your Stripe account configuration (typically Visa, Mastercard,
American Express).

**Refunds**: if an order is cancelled after payment is captured, a refund is
initiated automatically. Refunds appear in your bank account within 3–5
business days depending on your bank.

Refund status can be seen on the order detail page once the flow is
confirmed end-to-end. (The refund processing itself is implemented; the
buyer-facing status display is in progress.)

---

## Email notifications

ZapMarket sends emails for:

- Order confirmed
- Order cancelled
- Payment successful
- Payment failed
- Refund processed

Email delivery requires the platform operator to have configured an SMTP or
Resend API connection. If you are not receiving emails, contact your
platform administrator.

---

## Troubleshooting

**"Registration incomplete — please finish setting up your account"**
You started registration but did not complete all four steps. Return to
the registration page and continue from where you left off. Your email
and password are already saved.

**"Invalid OTP" during verification**
OTP codes expire after a short time. Click **Resend** to get a new code
and enter it within a minute.

**Card declined at checkout**
Check that the card number, expiry, and CVC are correct. If the problem
persists, try a different card or contact your bank.

**Cart is empty after returning to the site**
The cart is stored in your browser's local storage. Clearing your browser
data or switching to a different browser will clear the cart.

**Page shows an error after logging in**
If you see an error immediately after logging in, your session may have
expired. Click **Sign in** to log in again.
