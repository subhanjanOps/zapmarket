# Seller UI — Planning Document

## Overview

A dedicated Next.js frontend for marketplace sellers. Sellers list products, manage variants (SKUs), upload images, and track orders placed against their inventory. This is distinct from the admin-ui (gateway operations) and the future buyer-facing storefront.

**Stack:** Next.js 15 App Router · TypeScript · Same CSS variable/theme system as admin-ui (Roboto, warm walnut palette, rounded cards)  
**Location:** `services/seller-ui`  
**Dev port:** 3002  
**Talks to:** API Gateway (`:8000`) → product-catalog-service, order-management-service  

---

## Design Decisions

### 1. Seller isolation on orders
The order-management-service `GET /v1/orders` returns orders by the authenticated user's `user_id` — meaning it returns orders a seller placed *as a buyer*, not orders placed *against their products*. 

**Decision:** Add `GET /v1/orders/seller` endpoint to order-management-service that joins orders → order_items → skus by `seller_id`. In v1 this is a single SQL query. The seller UI will use this new endpoint.

### 2. Inventory visibility
Inventory-service is gRPC-only (no REST). Stock quantities are not exposed via HTTP.

**Decision:** v1 shows SKU `is_active` and `price_amount` only. Stock count is out of scope until inventory-service grows an HTTP layer. Sellers manage availability by toggling `is_active` on SKUs.

### 3. Image upload UX
API supports multipart upload, up to 5MB, per image. Supports both product-level and SKU-specific images.

**Decision:** Drag-and-drop zone that accepts multiple files in one drop, uploads them sequentially with per-file progress indicators. Optional SKU association via a dropdown after upload. Position reordering via drag handles.

### 4. Registration / Auth
Same auth-service as everyone else. JWT role must be `seller`.

**Decision:** Seller UI has its own `/register` page (role=seller hardcoded in request). Login works the same as admin-ui. No admin approval gate in v1 — sellers self-register and immediately get access.

---

## Backend changes required

| Change | Service | Why |
|---|---|---|
| `GET /v1/orders/seller` | order-management-service | Sellers need to see orders against their products, not their own buyer orders |
| Route registered in api-gateway | api-gateway DB | New seller order endpoint needs a route entry |

---

## Pages

### `/` → redirect to `/dashboard` or `/login`

### `/login`
- Email + password form
- Redirects to `/dashboard` if already authenticated
- Link to `/register`

### `/register`  
- Name, email, password fields
- Calls `POST /v1/auth/register` with role=seller (if the auth API supports role in register body — check; otherwise admin seeds sellers)
- On success → redirect to `/login`

### `/dashboard` — Overview
**Cards (top row):**
- Total products (active / total)
- Active SKUs
- Orders this month (count)
- Revenue this month (sum of confirmed order totals for seller's SKUs)

**Below cards:**
- Recent orders table (last 10) — order ID truncated, buyer (anonymous/uuid), SKUs purchased, total, status badge
- Product status breakdown — a small list: X active, Y draft, Z archived

### `/dashboard/products` — Product list
- Search bar (filters by name)
- Status filter tabs: All · Active · Draft · Archived
- Table: thumbnail | name | category | SKUs | status | created | actions
- "New Product" button → opens create flow
- Row actions: Edit · Archive · Delete (with confirm)
- Pagination

### `/dashboard/products/new` — Create product
Step-by-step within a single page (not a wizard, just logical sections):
1. **Basic info** — name, slug (auto-generated from name, editable), description, category dropdown
2. **SKUs** — inline table to add variants. Each SKU: sku_code, variant attributes (key-value pairs), price, compare price, weight, active toggle
3. **Images** — drag-and-drop zone. After upload, each image shows a thumbnail with "Assign to SKU" dropdown and position drag handle
4. **Publish** — status selector (DRAFT / ACTIVE) + Save button

### `/dashboard/products/[id]` — Edit product
Same layout as create, pre-filled. SKU section shows existing SKUs with edit-in-place. Images section shows existing images with delete/reorder.

### `/dashboard/orders` — Order list
- Status filter: All · Pending · Confirmed · Cancelled
- Date range filter
- Table: order ID | date | items (SKU names, qty) | total | status | actions
- Row action: View detail
- Pagination (offset-based, matches API)

### `/dashboard/orders/[id]` — Order detail
- Header: order ID, status badge, created date
- Buyer: UUID (no PII in seller view)
- Line items table: SKU name | SKU code | qty | unit price | subtotal
- Order total
- Status timeline (visual stepper): PENDING → RESERVED → CONFIRMED / CANCELLED
- If status is PENDING or RESERVED: Cancel Order button (calls order service cancel endpoint — seller should be able to cancel too)

---

## Component inventory

| Component | Used on |
|---|---|
| `<StatCard>` | Dashboard overview |
| `<ProductTable>` | Products list |
| `<SKURow>` (inline editable) | Product create/edit |
| `<ImageDropzone>` | Product create/edit |
| `<ImageThumbnail>` (with drag handle) | Product create/edit |
| `<OrderTable>` | Orders list |
| `<StatusTimeline>` | Order detail |
| `<StatusBadge>` | Products, Orders |
| `<Sidebar>` + `<Layout>` | All dashboard pages |
| `<ThemePicker>` | Sidebar footer (same as admin-ui) |
| `<Skeleton>` variants | All pages during load |

---

## API calls (`lib/api.ts`)

```
// Auth
login(email, password) → token
register(name, email, password) → void

// Products (seller-scoped — API filters by seller_id from JWT)
getProducts(token, {status?, search?, limit?, offset?}) → Product[]
getProduct(token, id) → Product
createProduct(token, data) → Product
updateProduct(token, id, data) → Product
deleteProduct(token, id) → void

// SKUs
getSkus(token, productId) → SKU[]
createSku(token, data) → SKU
updateSku(token, id, data) → SKU
deleteSku(token, id) → void

// Images
getImages(token, productId) → ProductImage[]
uploadImage(token, productId, file, skuId?) → ProductImage
setImagePosition(token, productId, imageId, position) → void
deleteImage(token, productId, imageId) → void

// Categories (read-only for sellers)
getCategories() → Category[]

// Orders (new seller endpoint)
getSellerOrders(token, {status?, from?, to?, limit?, offset?}) → Order[]
getSellerOrder(token, id) → {order: Order, items: OrderItem[]}
cancelOrder(token, id) → Order
```

---

## Visual design direction

Same design language as admin-ui (so they feel like a coherent product family):
- Roboto + Roboto Mono
- Walnut warm dark default, same 5-theme switcher
- 10px card radius, shadow-sm elevation
- Accent: `#e08c42` amber

**Differentiator from admin-ui:** seller-ui has a more content-heavy, editorial feel — products have images and rich metadata. The product edit page in particular needs a two-column layout (form left, image/preview right) rather than the single-column table-heavy admin UI.

---

## File structure

```
services/seller-ui/
  app/
    login/page.tsx
    register/page.tsx
    dashboard/
      layout.tsx          ← sidebar, theme, auth guard
      page.tsx            ← overview
      products/
        page.tsx          ← product list
        new/page.tsx      ← create product
        [id]/page.tsx     ← edit product
        loading.tsx
      orders/
        page.tsx          ← order list
        [id]/page.tsx     ← order detail
        loading.tsx
    components/
      Skeleton.tsx
      StatusBadge.tsx
      StatCard.tsx
      ImageDropzone.tsx
      SKUEditor.tsx
      StatusTimeline.tsx
  lib/
    api.ts
    auth.ts
  app/globals.css
  package.json
  next.config.ts
  tsconfig.json
```

---

## Build sequence

1. Scaffold Next.js app (`create-next-app`)
2. Copy globals.css + auth.ts pattern from admin-ui
3. Build `lib/api.ts` (all seller API calls)
4. Layout + login + register
5. Dashboard overview (StatCards + skeleton)
6. Products list page
7. Product create/edit page (the complex one — SKU editor + image dropzone)
8. Orders list + order detail
9. Backend: add `GET /v1/orders/seller` to order-management-service
10. Register the new route in api-gateway DB migration

---

## Open questions (deferred)

- Should sellers see buyer PII (name/email) in order detail? Currently the order has only `user_id`. Requires a cross-service lookup to auth-service. **Decision: defer to v2 — show user_id UUID only in v1.**
- Low-stock notification threshold per SKU — out of scope until inventory HTTP API exists.
- Seller analytics (revenue charts) — out of scope for v1, add after order data accumulates.
