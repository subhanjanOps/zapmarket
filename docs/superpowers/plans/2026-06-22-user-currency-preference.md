# Persistent User Currency Preference Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Store each user's preferred display currency server-side in auth-service so the preference persists across devices and browser sessions.

**Architecture:** A new `user_preferences` table in `userauth` DB holds key-value pairs per user. Two new endpoints on auth-service (`GET /v1/users/me/preferences` and `PUT /v1/users/me/preferences`) read and write these rows. The seller-ui reads the preference on mount (after rates load), saves it when the user changes currency, and falls back to the existing `localStorage` + locale-detection logic when the API is unavailable.

**Tech Stack:** Go 1.25, `database/sql`, `lib/pq`, existing auth-service middleware (JWT auth via `AuthMiddleware`), Next.js seller-ui.

## Global Constraints

- auth-service uses `net/http` ServeMux (NOT chi) — register routes on the existing mux
- No ORM — raw `database/sql` only
- JWT is already validated by `AuthMiddleware`; user ID is read from context via the existing helper (find in `internal/handler/http/handlers.go` how user claims are read from context — use the exact same method)
- Preference key for currency: `"display_currency"` (string constant)
- Currency code validation: must be 3 uppercase ASCII letters — reject anything else with 400
- All auth-service HTTP handlers use `internal/handler/http/base.go` helpers (`ErrorResponse`, `SuccessResponse`) — use those, do not hand-roll JSON responses
- seller-ui API calls go through `/api/proxy/` BFF route (see existing `fetchRates` call in `lib/currency.tsx` for the pattern)

---

### Task 1: auth-service — migration + repository + HTTP endpoints

**Files:**
- Create: `services/auth-service/migrations/NNNN_user_preferences.up.sql` (find the next migration number by listing `services/auth-service/migrations/`)
- Create: `services/auth-service/migrations/NNNN_user_preferences.down.sql`
- Create: `services/auth-service/internal/repository/preferences_repository.go`
- Modify: `services/auth-service/internal/domain/contracts/repositories.go` (add `PreferencesRepository` interface)
- Create: `services/auth-service/internal/handler/http/preferences_handler.go`
- Modify: `services/auth-service/cmd/main.go` or wherever routes are registered (find the mux setup) to register the two new routes

**Interfaces:**
- Produces:
  - `GET /v1/users/me/preferences` → `{"display_currency":"EUR"}` (200) or `{}` if no preference set
  - `PUT /v1/users/me/preferences` body `{"display_currency":"EUR"}` → `{"display_currency":"EUR"}` (200)
  - `PreferencesRepository` interface with `Get(ctx, userID, key) (string, error)` and `Set(ctx, userID, key, value string) error`

- [ ] **Step 1: Find the next migration number**

```bash
ls services/auth-service/migrations/ | sort | tail -5
```

Use the next sequential number (e.g. if last is `0005_...`, use `0006`).

- [ ] **Step 2: Write the migration**

```sql
-- services/auth-service/migrations/000N_user_preferences.up.sql
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key        VARCHAR(64) NOT NULL,
    value      TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, key)
);
```

```sql
-- services/auth-service/migrations/000N_user_preferences.down.sql
DROP TABLE IF EXISTS user_preferences;
```

- [ ] **Step 3: Add PreferencesRepository to domain contracts**

In `services/auth-service/internal/domain/contracts/repositories.go`, add:

```go
// PreferencesRepository stores per-user key-value preferences.
type PreferencesRepository interface {
    Get(ctx context.Context, userID uuid.UUID, key string) (string, bool, error)
    Set(ctx context.Context, userID uuid.UUID, key, value string) error
}
```

- [ ] **Step 4: Write the failing test for the handler**

```go
// services/auth-service/internal/handler/http/preferences_handler_test.go
package http_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/google/uuid"
    handler "github.com/zapmarket/zapmarket/services/auth-service/internal/handler/http"
    "github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

type fakePrefsRepo struct {
    stored map[string]string
}

func (f *fakePrefsRepo) Get(_ context.Context, userID uuid.UUID, key string) (string, bool, error) {
    v, ok := f.stored[key]
    return v, ok, nil
}

func (f *fakePrefsRepo) Set(_ context.Context, userID uuid.UUID, key, value string) error {
    f.stored[key] = value
    return nil
}

func TestGetPreferences_ReturnsStoredCurrency(t *testing.T) {
    repo := &fakePrefsRepo{stored: map[string]string{"display_currency": "EUR"}}
    h := handler.NewPreferencesHandler(repo)

    // Build a request with a user in context (look at how existing handler tests inject user context)
    ctx := context.WithValue(context.Background(), handler.ClaimsContextKey, &domain.Claims{
        UserID: uuid.New(),
    })
    req := httptest.NewRequest(http.MethodGet, "/v1/users/me/preferences", nil).WithContext(ctx)
    w := httptest.NewRecorder()

    h.GetPreferences(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
    var resp map[string]string
    json.NewDecoder(w.Body).Decode(&resp)
    if resp["display_currency"] != "EUR" {
        t.Errorf("expected EUR, got %q", resp["display_currency"])
    }
}

func TestSetPreferences_RejectsBadCurrencyCode(t *testing.T) {
    repo := &fakePrefsRepo{stored: map[string]string{}}
    h := handler.NewPreferencesHandler(repo)

    ctx := context.WithValue(context.Background(), handler.ClaimsContextKey, &domain.Claims{
        UserID: uuid.New(),
    })
    body := bytes.NewBufferString(`{"display_currency":"usd"}`) // lowercase — invalid
    req := httptest.NewRequest(http.MethodPut, "/v1/users/me/preferences", body).WithContext(ctx)
    w := httptest.NewRecorder()

    h.SetPreferences(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}
```

**Before writing the test**, look at:
- `services/auth-service/internal/handler/http/handlers.go` to find the exact context key used to store `*domain.Claims` — use that same key as `handler.ClaimsContextKey`
- Existing handler tests (if any) for the context injection pattern

- [ ] **Step 5: Run test to verify it fails**

```bash
cd services/auth-service
go test ./internal/handler/http/... -run TestGetPreferences -v
go test ./internal/handler/http/... -run TestSetPreferences -v
```

Expected: FAIL with `undefined: handler.NewPreferencesHandler`

- [ ] **Step 6: Implement preferences_repository.go**

```go
// services/auth-service/internal/repository/preferences_repository.go
package repository

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
)

type PreferencesRepository struct {
    db *sql.DB
}

func NewPreferencesRepository(db *sql.DB) *PreferencesRepository {
    return &PreferencesRepository{db: db}
}

func (r *PreferencesRepository) Get(ctx context.Context, userID uuid.UUID, key string) (string, bool, error) {
    var value string
    err := r.db.QueryRowContext(ctx,
        `SELECT value FROM user_preferences WHERE user_id = $1 AND key = $2`,
        userID, key).Scan(&value)
    if errors.Is(err, sql.ErrNoRows) {
        return "", false, nil
    }
    if err != nil {
        return "", false, fmt.Errorf("get preference: %w", err)
    }
    return value, true, nil
}

func (r *PreferencesRepository) Set(ctx context.Context, userID uuid.UUID, key, value string) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO user_preferences (user_id, key, value, updated_at)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (user_id, key) DO UPDATE
           SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
        userID, key, value, time.Now().UTC())
    if err != nil {
        return fmt.Errorf("set preference: %w", err)
    }
    return nil
}
```

- [ ] **Step 7: Implement preferences_handler.go**

First read `services/auth-service/internal/handler/http/handlers.go` to find: (a) the context key for claims, (b) the `ErrorResponse`/`SuccessResponse` helper signatures. Then write:

```go
// services/auth-service/internal/handler/http/preferences_handler.go
package http

import (
    "encoding/json"
    "net/http"
    "regexp"

    "github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
)

var validCurrencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

type PreferencesHandler struct {
    repo contracts.PreferencesRepository
}

func NewPreferencesHandler(repo contracts.PreferencesRepository) *PreferencesHandler {
    return &PreferencesHandler{repo: repo}
}

// GetPreferences handles GET /v1/users/me/preferences
func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
    claims := claimsFromContext(r.Context()) // use the exact helper from handlers.go
    if claims == nil {
        ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    val, ok, err := h.repo.Get(r.Context(), claims.UserID, "display_currency")
    if err != nil {
        ErrorResponse(w, http.StatusInternalServerError, "internal error")
        return
    }
    resp := map[string]string{}
    if ok {
        resp["display_currency"] = val
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}

// SetPreferences handles PUT /v1/users/me/preferences
func (h *PreferencesHandler) SetPreferences(w http.ResponseWriter, r *http.Request) {
    claims := claimsFromContext(r.Context())
    if claims == nil {
        ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    var body struct {
        DisplayCurrency string `json:"display_currency"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        ErrorResponse(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if !validCurrencyCode.MatchString(body.DisplayCurrency) {
        ErrorResponse(w, http.StatusBadRequest, "display_currency must be a 3-letter uppercase ISO code (e.g. EUR)")
        return
    }
    if err := h.repo.Set(r.Context(), claims.UserID, "display_currency", body.DisplayCurrency); err != nil {
        ErrorResponse(w, http.StatusInternalServerError, "internal error")
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"display_currency": body.DisplayCurrency})
}
```

Replace `claimsFromContext` with the actual function/helper used in `handlers.go` to read claims from context.

- [ ] **Step 8: Run tests to verify they pass**

```bash
cd services/auth-service
go test ./internal/handler/http/... -run TestGetPreferences -v
go test ./internal/handler/http/... -run TestSetPreferences -v
```

Expected: both PASS

- [ ] **Step 9: Register routes**

Find where the HTTP mux is set up in auth-service (likely `cmd/main.go` or a `routes.go` file). Register:

```go
prefsRepo    := repository.NewPreferencesRepository(db)
prefsHandler := httphandler.NewPreferencesHandler(prefsRepo)

// Wrap with AuthMiddleware (look at how existing protected routes are wrapped):
mux.Handle("GET /v1/users/me/preferences",  authMW.Authenticate(http.HandlerFunc(prefsHandler.GetPreferences)))
mux.Handle("PUT /v1/users/me/preferences",  authMW.Authenticate(http.HandlerFunc(prefsHandler.SetPreferences)))
```

- [ ] **Step 10: Build to verify**

```bash
cd services/auth-service
go build ./...
```

Expected: no errors

- [ ] **Step 11: Commit**

```bash
git add services/auth-service/migrations/ \
        services/auth-service/internal/domain/contracts/repositories.go \
        services/auth-service/internal/repository/preferences_repository.go \
        services/auth-service/internal/handler/http/preferences_handler.go \
        services/auth-service/internal/handler/http/preferences_handler_test.go
git commit -m "feat(auth): user preferences table and GET/PUT /v1/users/me/preferences endpoints"
```

---

### Task 2: seller-ui — sync preference with server

**Files:**
- Modify: `services/seller-ui/lib/currency.tsx`

**Interfaces:**
- Consumes: `GET /api/proxy/v1/users/me/preferences` and `PUT /api/proxy/v1/users/me/preferences` (via BFF — check that the gateway has a route for `/v1/users/me` pointing to `auth-service`)
- Produces: `CurrencyProvider` reads preference from server on mount; `setCurrency` saves to server + localStorage

**Note on gateway route:** Before implementing, check if the gateway's `gateway_routes` table has a row for `/v1/users/me` pointing to `auth-service` with `auth_mode=required`. If not, add a migration `services/api-gateway/migrations/000N_auth_me_routes.up.sql`:

```sql
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
VALUES ('/v1/users/me', 'auth-service', 'required', false)
ON CONFLICT (path_prefix) DO NOTHING;
```

Check the existing gateway migrations to find the next number.

- [ ] **Step 1: Add preference fetch/save helpers to lib/currency.tsx**

Add these two functions after the existing `fetchCurrencyList` function:

```typescript
async function fetchServerPreference(): Promise<string | null> {
  try {
    const res = await fetch("/api/proxy/v1/users/me/preferences");
    if (!res.ok) return null;
    const data = await res.json() as { display_currency?: string };
    return data.display_currency ?? null;
  } catch {
    return null;
  }
}

async function saveServerPreference(code: string): Promise<void> {
  try {
    await fetch("/api/proxy/v1/users/me/preferences", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ display_currency: code }),
    });
  } catch {
    // Silently ignore — localStorage is the source of truth on failure
  }
}
```

- [ ] **Step 2: Update CurrencyProvider to read preference from server on mount**

In the `useEffect` inside `CurrencyProvider`, change the currency initialization:

```typescript
// Replace:
const saved = localStorage.getItem("zap-currency");
setCurrencyState(saved ?? detectCurrency());

// With:
const saved = localStorage.getItem("zap-currency");
const initial = saved ?? detectCurrency();
setCurrencyState(initial);

// After Promise.all([fetchRates(), fetchCurrencyList()]) resolves, also fetch server preference:
Promise.all([fetchRates(), fetchCurrencyList(), fetchServerPreference()]).then(
  ([ratesResult, list, serverCurrency]) => {
    setRates(ratesResult.rates);
    setStale(ratesResult.stale);
    setRatesLoading(false);
    if (ratesResult.asOf) {
      try {
        setRatesDate(
          new Date(ratesResult.asOf).toLocaleDateString(undefined, {
            month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
          })
        );
      } catch { /* */ }
    }
    if (list.length > 0) setCurrencies(list);
    // Server preference wins over localStorage/locale detection:
    if (serverCurrency) {
      setCurrencyState(serverCurrency);
      localStorage.setItem("zap-currency", serverCurrency);
    }
  }
);
```

- [ ] **Step 3: Update setCurrency to save preference to server**

```typescript
// Replace:
const setCurrency = useCallback((c: string) => {
  setCurrencyState(c);
  localStorage.setItem("zap-currency", c);
}, []);

// With:
const setCurrency = useCallback((c: string) => {
  setCurrencyState(c);
  localStorage.setItem("zap-currency", c);
  saveServerPreference(c); // fire-and-forget
}, []);
```

- [ ] **Step 4: Verify TypeScript compilation**

```bash
cd services/seller-ui
npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add services/seller-ui/lib/currency.tsx
git commit -m "feat(seller-ui): sync currency preference with auth-service on mount and change"
```
