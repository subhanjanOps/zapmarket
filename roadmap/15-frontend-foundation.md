# Plan 15 — Frontend Foundation

**Effort:** M | **Impact:** M | **Depends on:** none

## Context
- No form validation in any UI (no react-hook-form, no zod)
- No data fetching/caching layer (no react-query or SWR) — every navigation triggers a full refetch
- seller-ui and backoffice-ui have near-zero UI primitives (no modals, tables, selects)
- buyer-ui is on Next.js 15; all others are on Next.js 16 — divergent upgrade paths
- No shared component package across 4 apps

## Scope
- Install react-hook-form + zod across all 4 apps
- Install @tanstack/react-query in buyer-ui and seller-ui
- Add shadcn/ui to seller-ui and admin-ui (components.json + init)
- Upgrade buyer-ui to Next.js 16

## Out of scope
- Full UI redesign
- Shared monorepo package (requires turborepo setup — separate plan)
- i18n

## Tasks

### seller-ui
- [ ] `npm install react-hook-form zod @hookform/resolvers @tanstack/react-query shadcn-ui`
- [ ] Run `npx shadcn@latest init` — use same config as buyer-ui (Tailwind v4, CSS vars)
- [ ] Add `QueryClientProvider` wrapper in `app/layout.tsx`
- [ ] Replace raw `<form>` in login/register with react-hook-form + zod schema
- [ ] Add shadcn `Table`, `Dialog`, `Select`, `Toast` components (most needed for order management)

### admin-ui
- [ ] `npm install react-hook-form zod @hookform/resolvers`
- [ ] Replace raw forms with react-hook-form; add zod validation schemas
- [ ] Replace raw Radix `Dialog` usage with shadcn `Dialog` wrapper for consistency

### backoffice-ui
- [ ] `npm install react-hook-form zod @hookform/resolvers @tanstack/react-query`
- [ ] Add `QueryClientProvider` in layout
- [ ] Add shadcn init

### buyer-ui
- [ ] Upgrade Next.js 15 → 16: `npm install next@16 react@19.2 react-dom@19.2`
- [ ] Fix any breaking changes (App Router API changes between 15 and 16)
- [ ] Add `@tanstack/react-query` if not already present; wrap product listing and cart calls in `useQuery`

## Done criteria
- All 4 apps compile on Next.js 16
- Login and register forms in all apps use react-hook-form + zod with inline validation errors
- Product listing in buyer-ui uses react-query (cached, no redundant fetches on tab switch)
- seller-ui has working Table and Dialog components from shadcn
