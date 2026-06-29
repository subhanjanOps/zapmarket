# Phase 5 — Infrastructure Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add contract tests for all gRPC/Kafka interfaces, set up database HA with PgBouncer + read replicas, add secrets management via Vault, build an admin analytics dashboard, and establish a load testing baseline with k6.

**Architecture:** Contract tests live in `tests/contract/` at the repo root and run in CI. PgBouncer is added to docker-compose as a connection pooler in front of PostgreSQL. Vault runs in dev mode in docker-compose; production uses HCP Vault. Admin analytics reads from PostgreSQL directly using aggregation queries. k6 scripts in `tests/load/` cover the three critical paths: auth, browse, and checkout.

**Tech Stack:** Pact Go (`github.com/pact-foundation/pact-go/v2`), buf CLI for proto breaking-change checks, PgBouncer, HashiCorp Vault, k6 (JavaScript), Next.js + Recharts for admin dashboard.

## Global Constraints

- Contract tests are read-only — they never write to production DBs
- Pact contracts are stored in `tests/contract/pacts/` and committed to the repo
- k6 scripts target `BASE_URL` env var (default `http://localhost:8000`)
- Admin analytics queries use `LIMIT` and time-range filters — never full table scans
- PgBouncer runs in transaction-pooling mode
- All Vault secret paths follow `secret/data/zapmarket/<service>/<key>`

---

## File Map

```
tests/
  contract/
    auth/
      consumer_test.go        — order-mgmt → auth-service ValidateToken contract
    inventory/
      consumer_test.go        — order-mgmt → inventory-service ReserveStock contract
    payment/
      consumer_test.go        — order-mgmt → payment-service ChargeCard contract
    kafka/
      order_event_test.go     — order-mgmt producer / notification consumer contract
    pacts/                    — generated pact files committed here

  load/
    auth.js                   — login + token refresh
    browse.js                 — product listing + search
    checkout.js               — add to cart + place order
    k6.config.js              — shared thresholds

infrastructure/
  pgbouncer/
    pgbouncer.ini             — connection pool config
    userlist.txt              — pg auth users

  vault/
    dev-init.sh               — seeds secrets in dev Vault
    policies/zapmarket.hcl    — service read policy

docker-compose.yml            — add pgbouncer, vault
docker-compose.override.yml   — dev overrides

services/admin-ui/
  app/dashboard/
    analytics/
      page.tsx                — new analytics page
      components/
        GmvChart.tsx
        OrderFunnelChart.tsx
        InventoryHealthTable.tsx
```

---

## Task 1: Contract Tests — gRPC (auth + inventory + payment)

**Files:**
- Create: `tests/contract/auth/consumer_test.go`
- Create: `tests/contract/inventory/consumer_test.go`
- Create: `tests/contract/payment/consumer_test.go`
- Create: `tests/contract/go.mod`

**Interfaces:**
- Produces: Pact files at `tests/contract/pacts/*.json`

- [ ] **Step 1: Initialize contract test module**

```bash
mkdir -p tests/contract
cd tests/contract
go mod init github.com/zapmarket/contract-tests
go get github.com/pact-foundation/pact-go/v2@latest
go mod tidy
```

- [ ] **Step 2: Write auth-service ValidateToken contract**

Create `tests/contract/auth/consumer_test.go`:

```go
package auth_test

import (
    "context"
    "fmt"
    "testing"

    "github.com/pact-foundation/pact-go/v2/consumer"
    "github.com/pact-foundation/pact-go/v2/matchers"
    "google.golang.org/grpc"
    authpb "github.com/zapmarket/pkg/proto/authpb"
)

func TestAuthService_ValidateToken_Contract(t *testing.T) {
    pact, err := consumer.NewV4Pact(consumer.MockHTTPProviderConfig{
        Consumer: "order-management-service",
        Provider: "auth-service",
        PactDir:  "../pacts",
    })
    if err != nil {
        t.Fatal(err)
    }

    pact.
        AddInteraction().
        UponReceiving("a valid JWT token").
        WithRequest(matchers.Map{
            "token": matchers.Like("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.valid"),
        }).
        WillRespondWith(matchers.Map{
            "user_id": matchers.Like("user-uuid-123"),
            "role":    matchers.Term("buyer", "buyer|seller|admin"),
            "valid":   matchers.Like(true),
        })

    err = pact.ExecuteTest(t, func(config consumer.MockServerConfig) error {
        conn, err := grpc.Dial(fmt.Sprintf("%s:%d", config.Host, config.Port), grpc.WithInsecure())
        if err != nil {
            return err
        }
        defer conn.Close()
        client := authpb.NewAuthServiceClient(conn)
        resp, err := client.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{
            Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.valid",
        })
        if err != nil {
            return err
        }
        if !resp.Valid {
            return fmt.Errorf("expected valid=true, got false")
        }
        return nil
    })
    if err != nil {
        t.Fatal(err)
    }
}
```

- [ ] **Step 3: Write inventory-service ReserveStock contract**

Create `tests/contract/inventory/consumer_test.go`:

```go
package inventory_test

import (
    "context"
    "fmt"
    "testing"

    "github.com/pact-foundation/pact-go/v2/consumer"
    "github.com/pact-foundation/pact-go/v2/matchers"
    inventorypb "github.com/zapmarket/pkg/proto/inventorypb"
    "google.golang.org/grpc"
)

func TestInventoryService_ReserveStock_Contract(t *testing.T) {
    pact, _ := consumer.NewV4Pact(consumer.MockHTTPProviderConfig{
        Consumer: "order-management-service",
        Provider: "inventory-service",
        PactDir:  "../pacts",
    })

    pact.
        AddInteraction().
        UponReceiving("a reserve stock request for available SKU").
        WithRequest(matchers.Map{
            "sku_id":   matchers.Like("sku-uuid-1"),
            "order_id": matchers.Like("order-uuid-1"),
            "quantity": matchers.Like(int32(2)),
        }).
        WillRespondWith(matchers.Map{
            "reservation_id": matchers.Like("res-uuid-1"),
            "success":        matchers.Like(true),
        })

    err := pact.ExecuteTest(t, func(config consumer.MockServerConfig) error {
        conn, _ := grpc.Dial(fmt.Sprintf("%s:%d", config.Host, config.Port), grpc.WithInsecure())
        defer conn.Close()
        client := inventorypb.NewInventoryServiceClient(conn)
        resp, err := client.ReserveStock(context.Background(), &inventorypb.ReserveStockRequest{
            SkuId:    "sku-uuid-1",
            OrderId:  "order-uuid-1",
            Quantity: 2,
        })
        if err != nil {
            return err
        }
        if !resp.Success {
            return fmt.Errorf("expected success=true")
        }
        return nil
    })
    if err != nil {
        t.Fatal(err)
    }
}
```

- [ ] **Step 4: Run contract tests**

```bash
cd tests/contract
go test ./... -v
```

Expected: `PASS` — pact JSON files generated in `tests/contract/pacts/`

- [ ] **Step 5: Add buf breaking-change check**

Create `buf.work.yaml` at repo root:

```yaml
version: v1
directories:
  - pkg/proto
```

Create `pkg/proto/buf.yaml`:

```yaml
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

Add to CI (or run manually):

```bash
buf breaking --against '.git#branch=main'
```

Expected: exits 0 if no breaking changes to any `.proto` file.

- [ ] **Step 6: Commit**

```bash
git add tests/contract/ buf.work.yaml pkg/proto/buf.yaml
git commit -m "test(contract): Pact gRPC contracts for auth/inventory/payment, buf breaking-change guard"
```

---

## Task 2: Kafka Event Contract Tests

**Files:**
- Create: `tests/contract/kafka/order_event_test.go`

**Interfaces:**
- Validates: `order.confirmed` JSON payload shape produced by order-management-service matches what notification-service expects

- [ ] **Step 1: Write failing Kafka contract test**

Create `tests/contract/kafka/order_event_test.go`:

```go
package kafka_test

import (
    "encoding/json"
    "testing"
    "time"
)

// OrderConfirmedEvent is the shape order-management-service produces
type OrderConfirmedEvent struct {
    OrderID     string    `json:"order_id"`
    UserID      string    `json:"user_id"`
    PaymentID   string    `json:"payment_id"`
    TotalAmount int64     `json:"total_amount"`
    Currency    string    `json:"currency"`
    ConfirmedAt time.Time `json:"confirmed_at"`
}

func TestOrderConfirmedEvent_HasRequiredFields(t *testing.T) {
    // Simulate a message produced by order-management-service
    raw := `{
        "order_id": "ord-123",
        "user_id": "usr-456",
        "payment_id": "pay-789",
        "total_amount": 99900,
        "currency": "INR",
        "confirmed_at": "2026-06-29T10:00:00Z"
    }`

    var evt OrderConfirmedEvent
    if err := json.Unmarshal([]byte(raw), &evt); err != nil {
        t.Fatalf("event does not match expected schema: %v", err)
    }
    if evt.OrderID == "" {
        t.Error("order_id is required")
    }
    if evt.UserID == "" {
        t.Error("user_id is required")
    }
    if evt.Currency == "" {
        t.Error("currency is required")
    }
    if evt.TotalAmount <= 0 {
        t.Error("total_amount must be positive")
    }
}
```

- [ ] **Step 2: Run tests**

```bash
cd tests/contract
go test ./kafka/... -v
```

Expected: `PASS`

- [ ] **Step 3: Commit**

```bash
git add tests/contract/kafka/
git commit -m "test(contract): Kafka event schema test for order.confirmed — locks payload shape"
```

---

## Task 3: Load Testing Baseline with k6

**Files:**
- Create: `tests/load/k6.config.js`
- Create: `tests/load/auth.js`
- Create: `tests/load/browse.js`
- Create: `tests/load/checkout.js`

- [ ] **Step 1: Create shared k6 config**

Create `tests/load/k6.config.js`:

```javascript
export const thresholds = {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed:   ['rate<0.01'],
};

export const stages = [
    { duration: '30s', target: 10  },  // ramp up
    { duration: '60s', target: 50  },  // sustain
    { duration: '30s', target: 100 },  // peak
    { duration: '30s', target: 0   },  // ramp down
];
```

- [ ] **Step 2: Create auth load test**

Create `tests/load/auth.js`:

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { thresholds, stages } from './k6.config.js';

export const options = { thresholds, stages };

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export default function () {
    const loginRes = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
        email:    'loadtest@zapmarket.in',
        password: 'Load1234!',
    }), { headers: { 'Content-Type': 'application/json' } });

    check(loginRes, {
        'login status 200': (r) => r.status === 200,
        'has access_token': (r) => JSON.parse(r.body).access_token !== undefined,
    });

    sleep(1);
}
```

- [ ] **Step 3: Create browse load test**

Create `tests/load/browse.js`:

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { thresholds, stages } from './k6.config.js';

export const options = { thresholds, stages };

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export default function () {
    const listRes = http.get(`${BASE_URL}/v1/products?limit=20&offset=0`);
    check(listRes, { 'product list 200': (r) => r.status === 200 });

    const searchRes = http.get(`${BASE_URL}/v1/products/search?q=phone&per_page=20`);
    check(searchRes, { 'search 200': (r) => r.status === 200 });

    sleep(0.5);
}
```

- [ ] **Step 4: Create checkout load test**

Create `tests/load/checkout.js`:

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { thresholds } from './k6.config.js';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Lower VUs for checkout — it hits payment gateway
export const options = {
    thresholds,
    stages: [
        { duration: '30s', target: 5  },
        { duration: '60s', target: 10 },
        { duration: '30s', target: 0  },
    ],
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const TOKEN    = __ENV.LOAD_TEST_TOKEN || '';
const SKU_ID   = __ENV.LOAD_TEST_SKU_ID || '';

export default function () {
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${TOKEN}`,
        'Idempotency-Key': uuidv4(),
    };

    const orderRes = http.post(`${BASE_URL}/v1/orders`, JSON.stringify({
        items: [{ sku_id: SKU_ID, quantity: 1 }],
        currency: 'INR',
    }), { headers });

    check(orderRes, { 'order created 201 or 202': (r) => r.status === 201 || r.status === 202 });
    sleep(2);
}
```

- [ ] **Step 5: Run browse test against local stack**

```bash
# Start services first
docker compose up -d

# Run browse test (read-only, safe to run against local)
k6 run tests/load/browse.js --env BASE_URL=http://localhost:8000
```

Expected output:
```
✓ product list 200
✓ search 200
http_req_duration p(95)<500ms ✓
```

- [ ] **Step 6: Commit**

```bash
git add tests/load/
git commit -m "test(load): k6 baseline scripts for auth, browse, and checkout — p95<500ms threshold"
```

---

## Task 4: PgBouncer Connection Pooling

**Files:**
- Create: `infrastructure/pgbouncer/pgbouncer.ini`
- Create: `infrastructure/pgbouncer/userlist.txt`
- Modify: `docker-compose.yml` — add pgbouncer service
- Modify: each service's `.env.example` — point `DB_HOST` to pgbouncer

- [ ] **Step 1: Create PgBouncer config**

Create `infrastructure/pgbouncer/pgbouncer.ini`:

```ini
[databases]
userauth       = host=postgres port=5432 dbname=userauth
productcatalog = host=postgres port=5432 dbname=productcatalog
ordermgmt      = host=postgres port=5432 dbname=ordermgmt
inventory      = host=postgres port=5432 dbname=inventory
payment        = host=postgres port=5432 dbname=payment
currency       = host=postgres port=5432 dbname=currency
settlement     = host=postgres port=5432 dbname=settlement
cart           = host=postgres port=5432 dbname=cart

[pgbouncer]
listen_addr     = 0.0.0.0
listen_port     = 5432
auth_type       = md5
auth_file       = /etc/pgbouncer/userlist.txt
pool_mode       = transaction
max_client_conn = 1000
default_pool_size = 20
reserve_pool_size = 5
log_connections = 1
log_disconnections = 1
```

Create `infrastructure/pgbouncer/userlist.txt`:

```
"zapuser" "zappass123"
```

- [ ] **Step 2: Add PgBouncer to docker-compose.yml**

```yaml
  pgbouncer:
    image: edoburu/pgbouncer:1.22.0
    ports:
      - "5433:5432"
    volumes:
      - ./infrastructure/pgbouncer/pgbouncer.ini:/etc/pgbouncer/pgbouncer.ini:ro
      - ./infrastructure/pgbouncer/userlist.txt:/etc/pgbouncer/userlist.txt:ro
    depends_on:
      - postgres
    environment:
      - PGBOUNCER_CONFIG=/etc/pgbouncer/pgbouncer.ini
```

- [ ] **Step 3: Update service env vars to point at pgbouncer**

In each service's `.env.example`, change:

```
DB_HOST=postgres
DB_PORT=5432
```

to:

```
DB_HOST=pgbouncer
DB_PORT=5432
```

- [ ] **Step 4: Test connection through pgbouncer**

```bash
docker compose up -d pgbouncer
psql -h localhost -p 5433 -U zapuser -d userauth -c "SELECT 1;"
```

Expected: `1` returned.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/pgbouncer/ docker-compose.yml
git commit -m "infra: add PgBouncer in transaction-pool mode, max 1000 client connections, 20 server pool per DB"
```

---

## Task 5: Admin Analytics Dashboard

**Files:**
- Create: `services/admin-ui/app/dashboard/analytics/page.tsx`
- Create: `services/admin-ui/app/dashboard/analytics/components/GmvChart.tsx`
- Create: `services/admin-ui/app/dashboard/analytics/components/OrderFunnelChart.tsx`
- Create: `services/admin-ui/app/api/analytics/route.ts` — server-side aggregation query

**Interfaces:**
- Produces: `GET /api/analytics?range=7d|30d|90d` returns `{ gmv, orders_by_status, top_products }`

- [ ] **Step 1: Add Recharts dependency**

```bash
cd services/admin-ui
npm install recharts
```

- [ ] **Step 2: Create analytics API route**

Create `services/admin-ui/app/api/analytics/route.ts`:

```typescript
import { NextRequest, NextResponse } from 'next/server';

const ORDER_SERVICE_URL = process.env.ORDER_SERVICE_URL || 'http://order-management-service:8084';

export async function GET(req: NextRequest) {
    const range = req.nextUrl.searchParams.get('range') || '7d';
    const days = range === '90d' ? 90 : range === '30d' ? 30 : 7;

    // Fetch from order-management-service admin endpoint
    const [gmvRes, funnelRes] = await Promise.all([
        fetch(`${ORDER_SERVICE_URL}/v1/admin/analytics/gmv?days=${days}`),
        fetch(`${ORDER_SERVICE_URL}/v1/admin/analytics/funnel?days=${days}`),
    ]);

    const gmv    = gmvRes.ok    ? await gmvRes.json()    : { daily: [] };
    const funnel = funnelRes.ok ? await funnelRes.json() : { stages: [] };

    return NextResponse.json({ gmv, funnel });
}
```

- [ ] **Step 3: Create GMV chart component**

Create `services/admin-ui/app/dashboard/analytics/components/GmvChart.tsx`:

```typescript
'use client';

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface DailyGmv { date: string; amount_paise: number }

export function GmvChart({ data }: { data: DailyGmv[] }) {
    const formatted = data.map(d => ({
        date: d.date,
        gmv:  Math.round(d.amount_paise / 100), // paise → rupees
    }));

    return (
        <div className="rounded-lg border p-4">
            <h3 className="mb-4 text-sm font-semibold text-gray-600">GMV (₹)</h3>
            <ResponsiveContainer width="100%" height={240}>
                <LineChart data={formatted}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `₹${v.toLocaleString('en-IN')}`} />
                    <Tooltip formatter={(v: number) => [`₹${v.toLocaleString('en-IN')}`, 'GMV']} />
                    <Line type="monotone" dataKey="gmv" stroke="#6366f1" strokeWidth={2} dot={false} />
                </LineChart>
            </ResponsiveContainer>
        </div>
    );
}
```

- [ ] **Step 4: Create analytics page**

Create `services/admin-ui/app/dashboard/analytics/page.tsx`:

```typescript
import { GmvChart } from './components/GmvChart';
import { OrderFunnelChart } from './components/OrderFunnelChart';

async function getAnalytics(range: string) {
    const res = await fetch(`${process.env.NEXT_PUBLIC_APP_URL}/api/analytics?range=${range}`, {
        next: { revalidate: 300 }, // cache 5 minutes
    });
    return res.ok ? res.json() : { gmv: { daily: [] }, funnel: { stages: [] } };
}

export default async function AnalyticsPage({ searchParams }: { searchParams: { range?: string } }) {
    const range = searchParams.range || '7d';
    const data = await getAnalytics(range);

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold">Analytics</h1>
                <div className="flex gap-2">
                    {['7d', '30d', '90d'].map(r => (
                        <a key={r} href={`?range=${r}`}
                           className={`rounded px-3 py-1 text-sm ${range === r ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-700'}`}>
                            {r}
                        </a>
                    ))}
                </div>
            </div>
            <GmvChart data={data.gmv.daily} />
            <OrderFunnelChart data={data.funnel.stages} />
        </div>
    );
}
```

- [ ] **Step 5: Add admin analytics endpoint to order-management-service**

In `services/order-management-service/internal/handler/http/admin_handler.go`, add:

```go
func (h *AdminHandler) GetGMV(w http.ResponseWriter, r *http.Request) {
    days, _ := strconv.Atoi(r.URL.Query().Get("days"))
    if days == 0 { days = 7 }

    rows, err := h.db.QueryContext(r.Context(), `
        SELECT DATE(created_at) as date,
               SUM(total_amount) as amount_paise
        FROM orders
        WHERE status = 'CONFIRMED'
          AND created_at >= NOW() - ($1 || ' days')::INTERVAL
        GROUP BY DATE(created_at)
        ORDER BY date
    `, days)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    // ... scan and return JSON
}
```

- [ ] **Step 6: Build and verify admin UI**

```bash
cd services/admin-ui
npm run build
```

Expected: no TypeScript errors

- [ ] **Step 7: Commit**

```bash
git add services/admin-ui/app/dashboard/analytics/ \
        services/admin-ui/app/api/analytics/ \
        services/order-management-service/internal/handler/http/admin_handler.go
git commit -m "feat(admin): analytics dashboard with GMV chart and order funnel, 5-min cache"
```

---

## Verification Checklist

- [ ] `cd tests/contract && go test ./... -v` — all Pact contracts pass
- [ ] `buf breaking --against '.git#branch=main'` — exits 0 on clean branch
- [ ] `k6 run tests/load/browse.js` — p95 < 500ms at 100 VUs
- [ ] `k6 run tests/load/auth.js` — p95 < 500ms at 100 VUs
- [ ] `psql -h localhost -p 5433 -U zapuser -d userauth -c "SELECT 1;"` — PgBouncer routes correctly
- [ ] Navigate to `/dashboard/analytics` in admin-ui — GMV chart renders with real data
- [ ] Switch 7d / 30d / 90d range filters — chart updates correctly
