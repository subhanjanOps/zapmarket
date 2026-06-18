# Backoffice UI — Business Admin & Catalog Management

**Status: 📋 Planned (2026-06-18, revised)**

A new `services/backoffice-ui` Next.js app (port **3003**) for business operations:
product catalog management, content moderation, and eventually user/order/seller
oversight. This is explicitly **not** an extension of `services/admin-ui`.

---

## Service boundary decision

| Service | Port | Audience | Concerns |
|---|---|---|---|
| `admin-ui` | 3001 | Engineers / DevOps | Gateway ops: routes, registry, metrics, audit, blocklist |
| `seller-ui` | 3002 | Sellers | Own product listings, own orders |
| **`backoffice-ui`** | **3003** | Business ops / product managers / moderators | Categories, product moderation, SKU pricing, (later) users, orders |

Keeping these separate means:
- `admin-ui` stays a tightly-scoped infra tool — its `lib/api.ts` only ever talks to
  `/gateway/v1/*`. No catalog concern bleeds in.
- `backoffice-ui` can grow freely into all business-facing concerns without touching
  the infra panel.
- Different lockdown posture: `admin-ui` access = infra team only; `backoffice-ui`
  access = business ops team.

---

## What already exists (backend)

| Concern | Endpoint | RBAC |
|---|---|---|
| Auth login | `POST /v1/auth/login` → `access_token` | any role |
| Category reads | `GET /api/v1/categories[/{id}]` | public |
| Category writes | `POST/PUT/DELETE /api/v1/categories[/{id}]` | `admin` only |
| Product reads | `GET /api/v1/products[/{id}]` | public |
| Product writes | `PUT/DELETE /api/v1/products/{id}` | `seller` or `admin` |
| SKU reads | `GET /api/v1/skus[/{id}]` | public |
| SKU writes | `POST/PUT/DELETE /api/v1/skus[/{id}]` | `seller` or `admin` |
| Image list | `GET /api/v1/products/{id}/images` | public |
| Image delete | `DELETE /api/v1/products/{id}/images/{imageId}` | `seller` or `admin` |

All three gateway routes exist (`/api/v1/products`, `/api/v1/categories`,
`/api/v1/skus`). No new gateway rows needed.

---

## New service scaffold

```
services/backoffice-ui/
  package.json                 Next.js 16, same deps as seller-ui
  Dockerfile                   PORT 3003
  next.config.ts
  tsconfig.json
  .env.example                 NEXT_PUBLIC_GATEWAY_URL=http://localhost:8000
  lib/
    auth.ts                    saveToken / getToken / clearToken (localStorage key: bo_token)
    api.ts                     req() helper + all catalog API functions
  app/
    layout.tsx                 Root layout, font imports
    globals.css                Copy from seller-ui (identical design tokens + themes)
    page.tsx                   Redirect → /dashboard
    login/
      page.tsx                 Login form (role: admin)
    dashboard/
      layout.tsx               Sidebar + auth guard
      loading.tsx              Splash skeleton
      page.tsx                 Overview — stat cards (total products, pending, categories, SKUs)
      categories/
        page.tsx               Category tree — CRUD
        loading.tsx
      products/
        page.tsx               All-sellers product list
        loading.tsx
        [id]/
          page.tsx             Product detail + inline SKU editor + image moderation
          loading.tsx
      skus/
        page.tsx               Cross-product SKU browser
        loading.tsx
      moderation/
        page.tsx               DRAFT queue — approve / archive
        loading.tsx
```

---

## `lib/api.ts` — full function list

Uses `{ success, data }` unwrapping — identical pattern to seller-ui.

```typescript
const GW = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

// auth
login(email, password)                 → { token: string }   // maps access_token → token

// categories
getCategories()                        GET  /api/v1/categories
getCategory(id)                        GET  /api/v1/categories/{id}
createCategory(token, body)            POST /api/v1/categories
updateCategory(token, id, body)        PUT  /api/v1/categories/{id}
deleteCategory(token, id)              DELETE /api/v1/categories/{id}

// products (admin does not CREATE — sellers do)
getProducts(token?, params)            GET  /api/v1/products
getProduct(token?, id)                 GET  /api/v1/products/{id}
updateProduct(token, id, body)         PUT  /api/v1/products/{id}
deleteProduct(token, id)               DELETE /api/v1/products/{id}

// skus
getSkus(token?, params)                GET  /api/v1/skus
getSku(token?, id)                     GET  /api/v1/skus/{id}
createSku(token, body)                 POST /api/v1/skus
updateSku(token, id, body)             PUT  /api/v1/skus/{id}
deleteSku(token, id)                   DELETE /api/v1/skus/{id}

// images
getImages(productId)                   GET  /api/v1/products/{id}/images
deleteImage(token, productId, imgId)   DELETE /api/v1/products/{id}/images/{imgId}
```

List responses: `{ success, data: T[], total, page, page_size }` — unwrap to internal
shape. Single responses: `{ success, data: T }` — return `r.data`.

---

## Sidebar navigation

```
ZapMarket Backoffice

── Overview ─────────
Dashboard

── Catalog ──────────
Categories            FolderTree icon
Products              Package icon
SKUs                  Tag icon
Moderation            ClipboardCheck icon

── (future) ─────────
Users                 Users icon      (Phase 2)
Orders                ShoppingCart icon (Phase 2)
```

Section labels rendered as non-link dividers between groups (same visual style as
admin-ui's existing sidebar but with group labels).

---

## Page specs

### Overview — `/dashboard`

Stat cards row: Total Products · Pending Moderation · Total Categories · Total SKUs.
Below: table of the 10 most recent DRAFT products (shortcut into the moderation queue).

```
┌──────────┐  ┌───────────────────┐  ┌────────────────┐  ┌──────────┐
│ Products │  │ Pending Moderation│  │  Categories    │  │   SKUs   │
│   1,240  │  │        18  ⚠      │  │      32        │  │   4,801  │
└──────────┘  └───────────────────┘  └────────────────┘  └──────────┘

Recent pending listings
┌────────────────────────────────────────────────────────────────────────┐
│  Name               Seller           Submitted       Action            │
│  iPhone 15 Pro Max  1a03bea2…        Jun 18 09:14    [Review →]        │
└────────────────────────────────────────────────────────────────────────┘
```

---

### Categories — `/dashboard/categories`

Full tree built client-side from flat `parent_id` array. Depth-indented rows.

```
[Page header] Categories                          [+ New Category]

┌─────────────────────────────────────────────────────────────────┐
│  Name              Slug               Children   Actions        │
│  Electronics       electronics        3          [Edit] [Del]   │
│    ├─ Phones       phones             0          [Edit] [Del]   │
│    ├─ Laptops      laptops            0          [Edit] [Del]   │
│    └─ Accessories  accessories        0          [Edit] [Del]   │
│  Clothing          clothing           2          [Edit] [Del]   │
└─────────────────────────────────────────────────────────────────┘
```

Create/Edit modal fields: Name, Slug (auto from name), Parent (select; "None" = root).
Delete guard: if category has children or products, show warning before confirming.

---

### Products — `/dashboard/products`

All sellers, all products. Status filter tabs + search. Inline status toggle.

```
[Page header] Products            [🔍 search]   All | ACTIVE | DRAFT | ARCHIVED

┌────────────────────────────────────────────────────────────────────────────────┐
│  Name           Category    Seller        Status    Created    Actions         │
│  iPhone 15      Phones      1a03bea2…     ACTIVE    Jun 18     [Edit]          │
│                                                                [Archive][Del]  │
└────────────────────────────────────────────────────────────────────────────────┘
                                           Page 1 of 3   [← Prev]  [Next →]
```

Row actions:
- **Edit** → `/dashboard/products/{id}`
- **Archive** → `PUT /api/v1/products/{id}` `{ status: "ARCHIVED" }` (inline, no nav)
- **Activate** → same with `{ status: "ACTIVE" }` (shown when DRAFT or ARCHIVED)
- **Delete** → confirm modal

---

### Product Detail — `/dashboard/products/[id]`

Two-column. Left: metadata form. Right: SKU table + images.

```
← Products    iPhone 15 Pro Max                         [Save] [Archive]

LEFT (55%)                           RIGHT (45%)
┌──────────────────────────────┐    ┌──────────────────────────────────────┐
│ Name     [__________________]│    │ SKUs                                 │
│ Slug     [__________________]│    │  Code       Price   Active  Actions  │
│ Desc     [__________________]│    │  IP15-128   $999    ✓       [▼ Edit] │
│          [__________________]│    │  IP15-256   $1099   ✓       [▼ Edit] │
│ Category [select ▼          ]│    │  ────────────────────────────────── │
│ Status   [select ▼          ]│    │  [+ Add SKU]                         │
└──────────────────────────────┘    └──────────────────────────────────────┘
                                    ┌──────────────────────────────────────┐
                                    │ Images                               │
                                    │  [img thumbnail] [img] [img]         │
                                    │  hover → [✕ Delete]                  │
                                    └──────────────────────────────────────┘
```

SKU edit: expandable inline row (not modal) — sku_code, price (dollars), currency,
is_active toggle, attributes as key-value pairs.

---

### SKUs — `/dashboard/skus`

Cross-product browser. Search by sku_code. Filter by product_id.

```
[Page header] SKUs         [🔍 sku_code…]  [product_id filter]

┌────────────────────────────────────────────────────────────────────────────┐
│  SKU Code      Product        Price     Currency  Active  Actions          │
│  IP15-128-BLK  iPhone 15 Pro  $999.00   INR       ✓       [→ Product]      │
│                                                            [Edit] [Delete] │
└────────────────────────────────────────────────────────────────────────────┘
```

Inline expand to edit price, toggle active, edit attributes.

---

### Moderation — `/dashboard/moderation`

DRAFT product review queue. Card layout — one card per product.

```
[Page header] Moderation         18 pending              [↻ Refresh]

┌─────────────────────────────────────────────────────────────────────────────┐
│  iPhone 15 Pro Max                              DRAFT    Jun 18, 09:14 AM   │
│  Phones · Seller: 1a03bea2-6e31…                                            │
│  "A premium smartphone with a Pro camera system and titanium design."       │
│  SKUs: 3   Images: 5                                                        │
│                                                [✓ Approve] [✕ Archive]      │
└─────────────────────────────────────────────────────────────────────────────┘
```

Approve → `PUT .../status: ACTIVE`. Archive → `PUT .../status: ARCHIVED`.
Empty state: "All caught up — no listings pending review." (green tint).

---

## Backend checks needed before building

| Check | Why |
|---|---|
| `GET /api/v1/products?status=DRAFT` works | Moderation queue depends on it |
| `GET /api/v1/products?seller_id=...` works | Products page "filter by seller" |
| `GET /api/v1/skus?sku_code=...` works | SKU search |

If any of these query params are not handled in the handler, add them first.

---

## `docker-compose.yml` addition

```yaml
backoffice-ui:
  build: ./services/backoffice-ui
  container_name: zapmarket-backoffice-ui
  ports:
    - "3003:3003"
  environment:
    - NEXT_PUBLIC_GATEWAY_URL=http://zapmarket-api-gateway:8000
  depends_on:
    - api-gateway
  networks:
    - zapmarket-network
```

---

## Implementation order

1. Scaffold service (`package.json`, `next.config.ts`, `Dockerfile`, `globals.css`,
   `lib/auth.ts`, `lib/api.ts`)
2. Login page + dashboard layout (sidebar, auth guard, theme switcher)
3. Verify backend query params (`?status=`, `?seller_id=`, `?sku_code=`)
4. Categories page — first real test of admin token + write flow
5. Overview page — stat cards from category/product/SKU counts
6. Moderation page — highest business value, simple card layout
7. Products list + detail (with inline SKU editor)
8. SKUs browser

---

## Out of scope (Phase 2, separate docs)

- **User management** — needs `GET/PUT /v1/admin/users` in auth-service
- **Order oversight** — needs admin endpoints in order-management-service
- **Seller verification** — `seller_verified` flag + email flow in auth-service
- **Bulk import/export** — CSV for categories/products
