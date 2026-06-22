# Admin UI Currency Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Currencies page to the admin-ui dashboard that lists all currencies and lets an admin enable/disable each one via a toggle.

**Architecture:** Follows the exact same pattern as the existing `Routes` page — a `lib/api.ts` addition for type + fetch functions, a new `app/dashboard/currencies/page.tsx` client component with a table and toggle buttons, a loading skeleton, and a nav entry in `app/dashboard/layout.tsx`. All API calls go through the Next.js BFF proxy at `/api/gateway/...` which attaches the admin JWT cookie automatically.

**Tech Stack:** Next.js (App Router, "use client"), TypeScript, existing `lib/api.ts` + `lib/hooks.ts` patterns, Lucide React icons.

## Global Constraints

- All API calls use the `/api/gateway/` BFF prefix — never call the gateway directly from the browser
- Follow the exact coding style of `app/dashboard/routes/page.tsx`: `useDataFetch`, modal pattern, `btn btn-primary`/`btn btn-ghost`/`btn btn-danger` classes
- Nav entry index: `"07"` (routes=02, registry=03, metrics=04, audit=05, blocklist=06)
- Currency toggle endpoint: `PUT /api/gateway/api/v1/admin/currencies/{code}` with body `{"enabled": true|false}`
- Currency list endpoint: `GET /api/gateway/api/v1/currencies`
- This is NOT standard Next.js — read `node_modules/next/dist/docs/` if unsure about any API. Heed deprecation notices.
- No new npm packages — use only what's already installed

---

### Task 1: API types and client functions

**Files:**
- Modify: `services/admin-ui/lib/api.ts`

**Interfaces:**
- Produces:
  - `Currency` type
  - `getCurrencies(): Promise<Currency[]>`
  - `toggleCurrency(code: string, enabled: boolean): Promise<void>`

- [ ] **Step 1: Read the existing api.ts to find where to insert**

Read `services/admin-ui/lib/api.ts` fully to understand the existing pattern before making changes.

- [ ] **Step 2: Add Currency type and functions**

At the end of `services/admin-ui/lib/api.ts`, add:

```typescript
export type Currency = {
  code: string;
  name: string;
  flag: string;
  decimals: number;
  enabled: boolean;
};

export async function getCurrencies(): Promise<Currency[]> {
  const res = await apiFetch("/api/gateway/api/v1/currencies");
  if (!res.ok) throw new Error(`getCurrencies: ${res.status}`);
  return res.json();
}

export async function toggleCurrency(code: string, enabled: boolean): Promise<void> {
  const res = await apiFetch(`/api/gateway/api/v1/admin/currencies/${code}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) throw new Error(`toggleCurrency: ${res.status}`);
}
```

Where `apiFetch` is whatever the existing fetch helper is in that file (look at how `getRoutes` or `createRoute` fetches — use the exact same helper).

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd services/admin-ui
npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add services/admin-ui/lib/api.ts
git commit -m "feat(admin-ui): add Currency type and toggle API functions"
```

---

### Task 2: Currencies page

**Files:**
- Create: `services/admin-ui/app/dashboard/currencies/page.tsx`

**Interfaces:**
- Consumes: `getCurrencies`, `toggleCurrency` from `@/lib/api`; `useDataFetch` from `@/lib/hooks`
- Produces: `default function CurrenciesPage()` — renders a table of currencies with toggle buttons

- [ ] **Step 1: Write the page**

```tsx
// services/admin-ui/app/dashboard/currencies/page.tsx
"use client";
import { useState } from "react";
import { getCurrencies, toggleCurrency, Currency } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function CurrenciesPage() {
  const { data, loading, error: fetchError, refresh } = useDataFetch<Currency[]>(getCurrencies);
  const currencies = data ?? [];
  const [actionError, setActionError] = useState("");
  const [togglingCode, setTogglingCode] = useState<string | null>(null);

  const error = actionError || fetchError;

  async function handleToggle(c: Currency) {
    setTogglingCode(c.code);
    setActionError("");
    try {
      await toggleCurrency(c.code, !c.enabled);
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    } finally {
      setTogglingCode(null);
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Currencies</h1>
          <p className="page-subtitle">
            {currencies.filter((c) => c.enabled).length} of {currencies.length} enabled
          </p>
        </div>
        <button className="btn btn-ghost" onClick={refresh}>Refresh</button>
      </div>

      {error && (
        <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>
      )}

      {loading && currencies.length === 0 ? (
        <SkeletonTableCard cols={5} rows={10} />
      ) : null}

      <div
        className="card"
        style={{
          padding: 0,
          display: loading && currencies.length === 0 ? "none" : undefined,
          overflowX: "auto",
        }}
      >
        <table style={{ minWidth: "32rem" }}>
          <thead>
            <tr>
              <th style={{ width: "4rem" }}>Flag</th>
              <th style={{ width: "6rem" }}>Code</th>
              <th>Name</th>
              <th style={{ width: "7rem" }}>Decimals</th>
              <th style={{ width: "9rem" }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {currencies.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="empty-state">
                    <p className="empty-state-title">No currencies found</p>
                    <p className="empty-state-body">Run migrations to seed the currencies table</p>
                  </div>
                </td>
              </tr>
            ) : (
              currencies.map((c) => (
                <tr key={c.code} data-status={c.enabled ? "ok" : "off"}>
                  <td style={{ fontSize: "1.25rem", textAlign: "center" }}>{c.flag}</td>
                  <td className="mono" style={{ fontWeight: 600 }}>{c.code}</td>
                  <td style={{ color: "var(--text-2)" }}>{c.name}</td>
                  <td style={{ color: "var(--muted)", textAlign: "center" }}>{c.decimals}</td>
                  <td>
                    <button
                      onClick={() => handleToggle(c)}
                      disabled={togglingCode === c.code}
                      style={{ background: "none", border: "none", cursor: togglingCode === c.code ? "wait" : "pointer", padding: 0 }}
                      title={c.enabled ? "Click to disable" : "Click to enable"}
                    >
                      <span className={`badge ${c.enabled ? "badge-green" : "badge-red"}`}>
                        {togglingCode === c.code ? "…" : c.enabled ? "enabled" : "disabled"}
                      </span>
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd services/admin-ui
npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add services/admin-ui/app/dashboard/currencies/page.tsx
git commit -m "feat(admin-ui): add currencies management page"
```

---

### Task 3: Nav entry + loading skeleton

**Files:**
- Create: `services/admin-ui/app/dashboard/currencies/loading.tsx`
- Modify: `services/admin-ui/app/dashboard/layout.tsx`

**Interfaces:**
- Consumes: `SkeletonTableCard` from `@/app/components/Skeleton`
- Produces: nav item `{ href: "/dashboard/currencies", label: "Currencies", Icon: Coins, idx: "07" }` added to `NAV` array in layout

- [ ] **Step 1: Create loading skeleton**

```tsx
// services/admin-ui/app/dashboard/currencies/loading.tsx
import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function Loading() {
  return <SkeletonTableCard cols={5} rows={10} />;
}
```

- [ ] **Step 2: Add nav entry to layout**

In `services/admin-ui/app/dashboard/layout.tsx`:

1. Add `Coins` to the lucide-react import line:
```tsx
import {
  LayoutDashboard, Route, Network, ScrollText,
  BarChart2, ShieldOff, LogOut, Menu, X, Coins,
} from "lucide-react";
```

2. Add to the `NAV` array after the blocklist entry:
```tsx
{ href: "/dashboard/currencies", label: "Currencies", Icon: Coins, idx: "07" },
```

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd services/admin-ui
npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add services/admin-ui/app/dashboard/currencies/ \
        services/admin-ui/app/dashboard/layout.tsx
git commit -m "feat(admin-ui): add Currencies nav entry and loading skeleton"
```
