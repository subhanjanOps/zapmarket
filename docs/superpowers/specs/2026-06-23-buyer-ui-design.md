# Buyer UI — Product-Focused Content Storefront

**Date:** 2026-06-23
**Status:** Approved
**Service:** `services/buyer-ui`
**Port:** 3000

---

## Overview

A Next.js 15 App Router storefront for ZapMarket buyers. The guiding principle is **product-first, content-driven**: every blog post, glossary term, and promotional flyer exists to serve the product. Buyers can browse, read, and research without creating an account; authentication is only required at checkout and for order history.

This mirrors the Amazon pattern: public by default, content-rich for SEO, account-gated only at the point of transaction.

---

## Architecture

### Stack
- **Framework:** Next.js 15, App Router, TypeScript
- **Styling:** Tailwind CSS (matches seller-ui/admin-ui conventions)
- **Content:** MDX via `next-mdx-remote` for blogs and glossary
- **Cart state:** Zustand, persisted to `localStorage` — no backend cart service
- **Auth:** JWT in httpOnly cookie; BFF proxy at `/api/auth/*` (same pattern as seller-ui)
- **API access:** BFF proxy at `/api/proxy/[...path]` → API gateway at `GATEWAY_URL`
- **Port:** 3000 in docker-compose

### docker-compose entry
New service `buyer-ui` depending on `api-gateway`. Env vars: `NEXT_PUBLIC_GATEWAY_URL`, `GATEWAY_URL`, `PORT=3000`, `HOSTNAME=0.0.0.0`. Build arg `NEXT_PUBLIC_GATEWAY_URL` for client-side fetch.

### Content storage
```
services/buyer-ui/
  content/
    blog/          # MDX files, one per post
    glossary/      # MDX files, one per term
```

MDX frontmatter schema:

**Blog post:**
```yaml
title: string
slug: string
excerpt: string
category: string          # e.g. "Buying Guide", "Trends", "How-to"
product_slugs: string[]   # links to live catalog products
published_at: YYYY-MM-DD
featured: bool
```

**Glossary term:**
```yaml
term: string
slug: string
category: string          # e.g. "Electronics", "Fashion", "Marketplace"
related_terms: string[]
product_slugs: string[]
```

---

## Campaigns (Flyers)

### Data model
New table `campaigns` in the `apigateway` PostgreSQL database:

```sql
CREATE TABLE campaigns (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug          TEXT UNIQUE NOT NULL,
  title         TEXT NOT NULL,
  headline      TEXT NOT NULL,
  banner_image_url TEXT,
  product_ids   UUID[] NOT NULL DEFAULT '{}',
  valid_from    TIMESTAMPTZ NOT NULL,
  valid_until   TIMESTAMPTZ NOT NULL,
  active        BOOLEAN NOT NULL DEFAULT false,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### API gateway endpoints
- `GET /v1/campaigns` — list active campaigns (where `active = true AND valid_until > NOW()`)
- `GET /v1/campaigns/:slug` — single campaign; 404 if not found or expired
- `POST /v1/campaigns` — admin only; create campaign
- `PUT /v1/campaigns/:id` — admin only; update
- `PATCH /v1/campaigns/:id/activate` — admin only; toggle active

These are implemented inside `services/api-gateway/internal/` following existing handler patterns.

---

## Page Map

| Route | Auth required | Data source |
|---|---|---|
| `/` | No | Catalog API + campaigns API |
| `/blog` | No | MDX index (static) |
| `/blog/[slug]` | No | MDX + catalog API (related products) |
| `/glossary` | No | MDX A–Z index (static) |
| `/glossary/[term]` | No | MDX + catalog API (related products) |
| `/deals/[slug]` | No | Campaigns API + catalog API |
| `/products` | No | Catalog API |
| `/products/[slug]` | No | Catalog API |
| `/cart` | No | Zustand / localStorage |
| `/checkout` | Yes | OMS API |
| `/account/orders` | Yes | OMS API |
| `/account/orders/[id]` | Yes | OMS API |
| `/login` | No | Auth API |
| `/register` | No | Auth API (role locked to `buyer`) |

---

## Home Page Sections

1. **Hero carousel** — rotates through active campaigns; each slide links to `/deals/[slug]`
2. **Shop by category** — grid of top-level categories from `GET /v1/categories`
3. **Today's deals** — up to 8 products from the most recently active campaign
4. **New arrivals** — 8 products sorted by `created_at desc` from `GET /v1/products`
5. **From the blog** — 3 most recent MDX posts (resolved at build time / ISR)
6. **Browse the glossary** — 6 featured terms (frontmatter `featured: true`)

---

## Product Listing (`/products`)

- Left sidebar: category tree (nested), price range slider (`min_price`/`max_price` query params), in-stock toggle
- Main grid: 20-per-page product cards — image (MinIO URL), name, price with currency, SKU count badge
- Sort controls: relevance, price asc/desc, newest
- URL-driven filters so pages are shareable and crawlable
- Pagination via `limit`/`offset` query params

---

## Product Detail (`/products/[slug]`)

- Image gallery (carousel of product images from `GET /v1/products/{id}/images`)
- SKU selector: variant attributes rendered as button groups (colour, size, etc.)
- Price: shown in user's preferred currency if logged in; falls back to default currency
- Stock badge: "In stock" / "Low stock" / "Out of stock" derived from inventory gRPC response cached at gateway
- "Add to cart" — always enabled; stores `{skuId, name, image, price, currency, qty}` in Zustand
- "Buy now" — adds to cart and navigates to `/checkout`

---

## Cart (`/cart`)

- Entirely client-side (Zustand + localStorage); no backend cart API
- Line items show: image, name, variant, unit price, quantity stepper, remove button
- Order summary: subtotal, estimated tax line (placeholder), total
- "Proceed to checkout" → `/checkout` (redirects to `/login?next=/checkout` if unauthenticated)

---

## Checkout (`/checkout`) — auth required

1. Order review (read-only cart summary)
2. Shipping address form (collected locally; passed in order notes — no address microservice)
3. "Place order" button → `POST /v1/orders` with `Idempotency-Key: <uuid>` header (generated via `crypto.randomUUID()` on checkout page mount, stored in component state) and cart items as body
4. On 201: clear cart, redirect to `/account/orders/[id]?new=1` with confirmation banner
5. On 409 (idempotency replay): same redirect with existing order ID

---

## My Orders (`/account/orders`)

- Requires login; redirects to `/login?next=/account/orders` if not authenticated
- Lists orders from `GET /v1/orders` with status badge, total, date
- Detail page `/account/orders/[id]`: line items, payment status, "Cancel" button (calls `POST /v1/orders/:id/cancel`) shown only when status is `PENDING`

---

## Auth Flow

- Register: role field is not shown; hardcoded to `buyer` in the request body
- Login: `POST /v1/auth/login` via BFF; JWT stored in httpOnly cookie
- `next` query param preserved through login redirect so buyers return to where they were
- Google OAuth: "Continue with Google" on login/register pages

---

## Blog (`/blog`, `/blog/[slug]`)

- `/blog`: paginated card grid; filter by `category` query param; sorted by `published_at desc`
- `/blog/[slug]`: full MDX render; "Related Products" section at bottom fetches live data for `product_slugs` from the frontmatter using `GET /v1/products?slugs=...` (or individual `GET /v1/products/slug/:slug` calls in parallel)
- ISR with `revalidate: 3600` so product data stays fresh without a full rebuild

---

## Glossary (`/glossary`, `/glossary/[term]`)

- `/glossary`: A–Z tabbed index; entries grouped by first letter; search input filters client-side
- `/glossary/[term]`: term definition in MDX; "Related Terms" sidebar; "Related Products" row (same live-fetch pattern as blog)
- Fully static at build time; `generateStaticParams` from MDX file list

---

## Campaign Flyers (`/deals/[slug]`)

- Full-width hero with `banner_image_url` (MinIO)
- Headline + validity countdown ("Deal ends in X hours")
- Curated product grid from campaign's `product_ids` — fetched live from catalog API
- 404 if campaign not found, inactive, or expired

---

## Backoffice: Campaign Manager

New page at `/dashboard/campaigns` in `services/backoffice-ui`. Follows the same patterns as existing backoffice pages (table + modal form). Provides:
- List all campaigns with status badge and validity dates
- Create campaign: slug, title, headline, banner upload (MinIO presigned URL), product picker (search by name), date range, activate toggle
- Edit / delete existing campaigns

No new service — campaigns are managed through the API gateway's admin endpoints.

---

## Error handling

- Product not found → custom 404 page with "Back to shopping" CTA
- API unreachable → static error boundary with retry button; never a blank white screen
- Cart checkout failure → inline error on the checkout page; cart is not cleared

---

## SEO

- All public pages use Next.js `generateMetadata` with title, description, og:image
- Blog and glossary pages have canonical URLs
- Product pages include structured data (`Product` schema.org JSON-LD) for Google Shopping
- `/sitemap.xml` generated at build time from MDX files + catalog slugs via `next-sitemap`

---

## Initial sample content

Seed files to include in the repo:

**Blog (5 posts):**
1. "How to choose the right SKU for your next purchase" — category: Buying Guide
2. "Top 10 electronics deals this season" — links to 10 product slugs
3. "Understanding GST-inclusive pricing in India" — category: Finance
4. "What is a return window and how does it protect you?" — category: Policy
5. "ZapMarket seller verification: what buyers need to know" — category: Trust & Safety

**Glossary (10 terms):**
SKU, GST, COD (Cash on Delivery), Return Window, Idempotency Key, Inventory Reservation, Seller Rating, Buy Box, Flash Sale, Category Tree

**Campaigns (2 seed SQL rows):**
- "Summer Sale" — valid for 30 days from deploy, active
- "New Arrivals Week" — valid for 7 days from deploy, active
