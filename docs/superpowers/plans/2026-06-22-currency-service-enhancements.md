# Currency Service Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fallback rate provider, rate history storage + endpoint, and a Kafka event published after each successful rate ingestion to the existing `currency-service`.

**Architecture:** The fallback provider wraps the existing `RatesProvider` port — the worker tries the primary (Frankfurter) first and falls back to a secondary (Open Exchange Rates) on error. Rate history is stored in a new append-only `exchange_rates_history` table populated on every successful upsert. The Kafka event is published by `IngestRatesUseCase` after a successful DB write, decoupling downstream consumers from the worker's schedule.

**Tech Stack:** Go 1.25, `database/sql`, `pkg/kafka` (segmentio/kafka-go), existing Prometheus metrics.

## Global Constraints

- Module: `github.com/zapmarket/zapmarket/services/currency-service`
- No ORM — raw `database/sql` with `lib/pq` only
- All new files follow the existing clean-architecture layering: `domain` → `application` → `infrastructure` → `interfaces`
- Kafka topic name: `currency.rates.updated` (string constant in `pkg/kafka/topics.go`)
- Open Exchange Rates free endpoint: `https://open.er-api.com/v6/latest/{base}` — no API key required
- Migration files use sequential numbering: next is `0004_...`
- History table primary key: `(base, quote, as_of)` to allow re-runs without duplicates
- `pkg/kafka` replace directive already exists in `pkg/` — add it to currency-service go.mod replace block: `github.com/zapmarket/zapmarket/pkg/kafka => ../../pkg/kafka`

---

### Task 1: Fallback rate provider

**Files:**
- Create: `services/currency-service/infrastructure/external/openexchangerates_provider.go`
- Create: `services/currency-service/infrastructure/external/fallback_provider.go`
- Modify: `services/currency-service/interfaces/worker/rate_ingestor.go` (no change needed — `IngestRatesUseCase` takes a `ports.RatesProvider`)
- Modify: `services/currency-service/application/usecases/ingest_rates.go` (no change needed)
- Modify: `services/currency-service/cmd/main.go` (wire fallback provider)
- Test: `services/currency-service/infrastructure/external/fallback_provider_test.go`

**Interfaces:**
- Consumes: `ports.RatesProvider` interface (`FetchLatest(ctx, base) (RateSet, error)`)
- Produces: `FallbackProvider` struct implementing `ports.RatesProvider`; `NewFallbackProvider(primary, secondary ports.RatesProvider) *FallbackProvider`

- [ ] **Step 1: Write the failing test**

```go
// services/currency-service/infrastructure/external/fallback_provider_test.go
package external_test

import (
    "context"
    "errors"
    "testing"

    "github.com/zapmarket/zapmarket/services/currency-service/application/ports"
    "github.com/zapmarket/zapmarket/services/currency-service/infrastructure/external"
)

type stubProvider struct {
    result ports.RateSet
    err    error
}

func (s *stubProvider) FetchLatest(_ context.Context, base string) (ports.RateSet, error) {
    return s.result, s.err
}

func TestFallbackProvider_UsesSecondaryOnPrimaryError(t *testing.T) {
    primary   := &stubProvider{err: errors.New("primary down")}
    secondary := &stubProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.9}}}

    fp := external.NewFallbackProvider(primary, secondary)
    got, err := fp.FetchLatest(context.Background(), "USD")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if got.Rates["EUR"] != 0.9 {
        t.Errorf("expected EUR=0.9, got %v", got.Rates["EUR"])
    }
}

func TestFallbackProvider_UsesPrimaryWhenHealthy(t *testing.T) {
    primary   := &stubProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.85}}}
    secondary := &stubProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.9}}}

    fp := external.NewFallbackProvider(primary, secondary)
    got, err := fp.FetchLatest(context.Background(), "USD")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if got.Rates["EUR"] != 0.85 {
        t.Errorf("expected EUR=0.85 from primary, got %v", got.Rates["EUR"])
    }
}

func TestFallbackProvider_ErrorsWhenBothFail(t *testing.T) {
    primary   := &stubProvider{err: errors.New("primary down")}
    secondary := &stubProvider{err: errors.New("secondary down")}

    fp := external.NewFallbackProvider(primary, secondary)
    _, err := fp.FetchLatest(context.Background(), "USD")
    if err == nil {
        t.Fatal("expected error when both providers fail")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/currency-service
go test ./infrastructure/external/... -run TestFallbackProvider -v
```

Expected: FAIL with `undefined: external.NewFallbackProvider`

- [ ] **Step 3: Write the OpenExchangeRates provider**

```go
// services/currency-service/infrastructure/external/openexchangerates_provider.go
package external

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/zapmarket/zapmarket/services/currency-service/application/ports"
)

type oerResponse struct {
    Base  string             `json:"base"`
    Rates map[string]float64 `json:"rates"`
}

// OpenExchangeRatesProvider fetches exchange rates from open.er-api.com (free tier, no key).
type OpenExchangeRatesProvider struct {
    baseURL    string
    httpClient *http.Client
}

func NewOpenExchangeRatesProvider(baseURL string) *OpenExchangeRatesProvider {
    return &OpenExchangeRatesProvider{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (p *OpenExchangeRatesProvider) FetchLatest(ctx context.Context, base string) (ports.RateSet, error) {
    url := fmt.Sprintf("%s/v6/latest/%s", p.baseURL, base)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return ports.RateSet{}, fmt.Errorf("oer: build request: %w", err)
    }

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return ports.RateSet{}, fmt.Errorf("oer: fetch: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return ports.RateSet{}, fmt.Errorf("oer: upstream status %d", resp.StatusCode)
    }

    var r oerResponse
    if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
        return ports.RateSet{}, fmt.Errorf("oer: decode: %w", err)
    }

    if r.Rates == nil {
        r.Rates = make(map[string]float64)
    }
    r.Rates[base] = 1.0

    return ports.RateSet{
        Base:  base,
        AsOf:  time.Now().UTC().Truncate(24 * time.Hour),
        Rates: r.Rates,
    }, nil
}
```

- [ ] **Step 4: Write the FallbackProvider**

```go
// services/currency-service/infrastructure/external/fallback_provider.go
package external

import (
    "context"
    "fmt"

    "github.com/zapmarket/zapmarket/services/currency-service/application/ports"
)

// FallbackProvider tries the primary provider first, then the secondary on error.
type FallbackProvider struct {
    primary   ports.RatesProvider
    secondary ports.RatesProvider
}

func NewFallbackProvider(primary, secondary ports.RatesProvider) *FallbackProvider {
    return &FallbackProvider{primary: primary, secondary: secondary}
}

func (f *FallbackProvider) FetchLatest(ctx context.Context, base string) (ports.RateSet, error) {
    rs, err := f.primary.FetchLatest(ctx, base)
    if err == nil {
        return rs, nil
    }
    rs2, err2 := f.secondary.FetchLatest(ctx, base)
    if err2 != nil {
        return ports.RateSet{}, fmt.Errorf("both providers failed: primary=%w; secondary=%v", err, err2)
    }
    return rs2, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/currency-service
go test ./infrastructure/external/... -run TestFallbackProvider -v
```

Expected: all 3 tests PASS

- [ ] **Step 6: Wire fallback provider in cmd/main.go**

In `services/currency-service/cmd/main.go`, find where `NewFrankfurterProvider` is instantiated and wrap it:

```go
// replace:
provider := external.NewFrankfurterProvider(cfg.RateProviderURL)

// with:
primaryProvider   := external.NewFrankfurterProvider(cfg.RateProviderURL)
secondaryProvider := external.NewOpenExchangeRatesProvider("https://open.er-api.com")
provider          := external.NewFallbackProvider(primaryProvider, secondaryProvider)
```

- [ ] **Step 7: Build to verify no compile errors**

```bash
cd services/currency-service
go build ./...
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add services/currency-service/infrastructure/external/openexchangerates_provider.go \
        services/currency-service/infrastructure/external/fallback_provider.go \
        services/currency-service/infrastructure/external/fallback_provider_test.go \
        services/currency-service/cmd/main.go
git commit -m "feat(currency): add OpenExchangeRates fallback provider"
```

---

### Task 2: Rate history storage + endpoint

**Files:**
- Create: `services/currency-service/migrations/0004_exchange_rates_history.up.sql`
- Create: `services/currency-service/migrations/0004_exchange_rates_history.down.sql`
- Modify: `services/currency-service/infrastructure/postgres/rates_repository.go` (add `InsertHistory` + `HistoryByBase`)
- Modify: `services/currency-service/domain/repositories/rates_repository.go` (extend interface)
- Create: `services/currency-service/application/usecases/get_rates_history.go`
- Modify: `services/currency-service/interfaces/http/handler.go` (add `GetRatesHistory`)
- Modify: `services/currency-service/interfaces/http/router.go` (register route)
- Test: `services/currency-service/application/usecases/get_rates_history_test.go`

**Interfaces:**
- Consumes: `RatesRepository` extended with `HistoryByBase(ctx, base, date time.Time) ([]entities.ExchangeRate, error)`
- Produces:
  - `GET /v1/currencies/rates/history?date=YYYY-MM-DD` → `{"base":"USD","date":"2026-06-01","rates":{"EUR":0.92,...}}`
  - `GetRatesHistoryUseCase` struct with `Execute(ctx, base, date) (RatesHistoryDTO, error)`

- [ ] **Step 1: Write the migration**

```sql
-- services/currency-service/migrations/0004_exchange_rates_history.up.sql
CREATE TABLE IF NOT EXISTS exchange_rates_history (
    base       CHAR(3)        NOT NULL REFERENCES currencies(code),
    quote      CHAR(3)        NOT NULL REFERENCES currencies(code),
    rate       NUMERIC(18, 8) NOT NULL,
    as_of      DATE           NOT NULL,
    fetched_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (base, quote, as_of)
);

CREATE INDEX IF NOT EXISTS idx_erh_base_asof ON exchange_rates_history (base, as_of DESC);
```

```sql
-- services/currency-service/migrations/0004_exchange_rates_history.down.sql
DROP TABLE IF EXISTS exchange_rates_history;
```

- [ ] **Step 2: Write the failing test**

```go
// services/currency-service/application/usecases/get_rates_history_test.go
package usecases_test

import (
    "context"
    "testing"
    "time"

    "github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
    "github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type fakeHistoryRepo struct {
    rows []entities.ExchangeRate
}

func (f *fakeHistoryRepo) HistoryByBase(_ context.Context, base string, date time.Time) ([]entities.ExchangeRate, error) {
    return f.rows, nil
}

func TestGetRatesHistory_ReturnsDTO(t *testing.T) {
    date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
    repo := &fakeHistoryRepo{rows: []entities.ExchangeRate{
        {Base: "USD", Quote: "EUR", Rate: 0.92, AsOf: date},
        {Base: "USD", Quote: "GBP", Rate: 0.79, AsOf: date},
    }}

    uc := usecases.NewGetRatesHistoryUseCase(repo)
    dto, err := uc.Execute(context.Background(), "USD", date)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if dto.Base != "USD" {
        t.Errorf("expected base USD, got %s", dto.Base)
    }
    if dto.Rates["EUR"] != 0.92 {
        t.Errorf("expected EUR=0.92, got %v", dto.Rates["EUR"])
    }
    if len(dto.Rates) != 2 {
        t.Errorf("expected 2 rates, got %d", len(dto.Rates))
    }
}

func TestGetRatesHistory_EmptyOnNoData(t *testing.T) {
    repo := &fakeHistoryRepo{rows: nil}
    uc := usecases.NewGetRatesHistoryUseCase(repo)
    date := time.Now().UTC()
    dto, err := uc.Execute(context.Background(), "USD", date)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(dto.Rates) != 0 {
        t.Errorf("expected empty rates, got %v", dto.Rates)
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd services/currency-service
go test ./application/usecases/... -run TestGetRatesHistory -v
```

Expected: FAIL with `undefined: usecases.NewGetRatesHistoryUseCase`

- [ ] **Step 4: Extend the RatesRepository interface**

Add to `services/currency-service/domain/repositories/rates_repository.go`:

```go
// Add to existing RatesRepository interface:
HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error)
```

Full updated file:

```go
package repositories

import (
    "context"
    "time"

    "github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type RatesRepository interface {
    LatestByBase(ctx context.Context, base string) ([]entities.ExchangeRate, time.Time, error)
    UpsertLatest(ctx context.Context, rates []entities.ExchangeRate, asOf time.Time) error
    HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error)
}
```

- [ ] **Step 5: Implement HistoryByBase in postgres repo**

Add to `services/currency-service/infrastructure/postgres/rates_repository.go`:

```go
func (r *RatesRepository) HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error) {
    rows, err := r.db.QueryContext(ctx,
        `SELECT base, quote, rate, as_of, fetched_at
         FROM exchange_rates_history
         WHERE base = $1 AND as_of = $2::date
         ORDER BY quote`,
        base, date.Format("2006-01-02"))
    if err != nil {
        return nil, fmt.Errorf("history rates: %w", err)
    }
    defer rows.Close()

    var out []entities.ExchangeRate
    for rows.Next() {
        var e entities.ExchangeRate
        if err := rows.Scan(&e.Base, &e.Quote, &e.Rate, &e.AsOf, &e.FetchedAt); err != nil {
            return nil, fmt.Errorf("history rates scan: %w", err)
        }
        out = append(out, e)
    }
    return out, rows.Err()
}
```

Also modify `UpsertLatest` to also insert into history table. Add after the existing upsert loop in the transaction:

```go
// After upsert loop, before tx.Commit():
for _, rate := range rates {
    _, err := tx.ExecContext(ctx, `
        INSERT INTO exchange_rates_history (base, quote, rate, as_of, fetched_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (base, quote, as_of) DO NOTHING`,
        rate.Base, rate.Quote, rate.Rate, asOf, now)
    if err != nil {
        return fmt.Errorf("history insert %s/%s: %w", rate.Base, rate.Quote, err)
    }
}
```

- [ ] **Step 6: Create the DTO and use case**

```go
// services/currency-service/application/usecases/get_rates_history.go
package usecases

import (
    "context"
    "time"

    "github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type RatesHistoryDTO struct {
    Base  string             `json:"base"`
    Date  string             `json:"date"`
    Rates map[string]float64 `json:"rates"`
}

type HistoryRepository interface {
    HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error)
}

type GetRatesHistoryUseCase struct {
    repo HistoryRepository
}

func NewGetRatesHistoryUseCase(repo HistoryRepository) *GetRatesHistoryUseCase {
    return &GetRatesHistoryUseCase{repo: repo}
}

func (uc *GetRatesHistoryUseCase) Execute(ctx context.Context, base string, date time.Time) (RatesHistoryDTO, error) {
    rows, err := uc.repo.HistoryByBase(ctx, base, date)
    if err != nil {
        return RatesHistoryDTO{}, err
    }
    rates := make(map[string]float64, len(rows))
    for _, r := range rows {
        rates[r.Quote] = r.Rate
    }
    return RatesHistoryDTO{
        Base:  base,
        Date:  date.Format("2006-01-02"),
        Rates: rates,
    }, nil
}
```

Also update the test's fake to implement `HistoryRepository` (uses `[]entities.ExchangeRate`):

```go
// Update fakeHistoryRepo in the test file to implement HistoryRepository:
type fakeHistoryRepo struct {
    rows []entities.ExchangeRate
}

func (f *fakeHistoryRepo) HistoryByBase(_ context.Context, base string, date time.Time) ([]entities.ExchangeRate, error) {
    return f.rows, nil
}
```

Add the import to the test file:
```go
import (
    "context"
    "testing"
    "time"

    "github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
    "github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd services/currency-service
go test ./application/usecases/... -run TestGetRatesHistory -v
```

Expected: both tests PASS

- [ ] **Step 8: Add HTTP handler**

Add to `services/currency-service/interfaces/http/handler.go`:

```go
// Add field to Handler struct:
getRatesHistory *usecases.GetRatesHistoryUseCase

// Add to NewHandler parameters:
func NewHandler(
    listCurrencies *usecases.ListCurrenciesUseCase,
    getRates       *usecases.GetRatesUseCase,
    toggleCurrency *usecases.ToggleCurrencyUseCase,
    getRatesHistory *usecases.GetRatesHistoryUseCase,
    log            *slog.Logger,
) *Handler {
    return &Handler{
        listCurrencies: listCurrencies,
        getRates:       getRates,
        toggleCurrency: toggleCurrency,
        getRatesHistory: getRatesHistory,
        log:            log,
    }
}

// Add handler method:
// GetRatesHistory handles GET /v1/currencies/rates/history?date=YYYY-MM-DD
func (h *Handler) GetRatesHistory(w http.ResponseWriter, r *http.Request) {
    dateStr := r.URL.Query().Get("date")
    if dateStr == "" {
        h.writeError(w, http.StatusBadRequest, "date query parameter is required (YYYY-MM-DD)")
        return
    }
    date, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        h.writeError(w, http.StatusBadRequest, "date must be in YYYY-MM-DD format")
        return
    }
    dto, err := h.getRatesHistory.Execute(r.Context(), "USD", date)
    if err != nil {
        h.log.Error("get rates history", "error", err)
        h.writeError(w, http.StatusInternalServerError, "internal error")
        return
    }
    w.Header().Set("Cache-Control", "public, max-age=86400")
    h.writeJSON(w, http.StatusOK, dto)
}
```

Add `"time"` to imports in handler.go.

- [ ] **Step 9: Register route in router.go**

In `services/currency-service/interfaces/http/router.go`, add:

```go
mux.HandleFunc("GET /v1/currencies/rates/history", h.GetRatesHistory)
```

Place it before the `GET /v1/currencies/rates` line so the more specific path matches first.

- [ ] **Step 10: Wire use case in cmd/main.go**

In `services/currency-service/cmd/main.go`, instantiate and pass the new use case:

```go
getRatesHistoryUC := usecases.NewGetRatesHistoryUseCase(ratesRepo)
// Pass to NewHandler:
handler := httphandler.NewHandler(listCurrenciesUC, getRatesUC, toggleCurrencyUC, getRatesHistoryUC, log)
```

- [ ] **Step 11: Build to verify**

```bash
cd services/currency-service
go build ./...
```

Expected: no errors

- [ ] **Step 12: Commit**

```bash
git add services/currency-service/migrations/0004_exchange_rates_history.up.sql \
        services/currency-service/migrations/0004_exchange_rates_history.down.sql \
        services/currency-service/domain/repositories/rates_repository.go \
        services/currency-service/infrastructure/postgres/rates_repository.go \
        services/currency-service/application/usecases/get_rates_history.go \
        services/currency-service/application/usecases/get_rates_history_test.go \
        services/currency-service/interfaces/http/handler.go \
        services/currency-service/interfaces/http/router.go \
        services/currency-service/cmd/main.go
git commit -m "feat(currency): rate history table and GET /v1/currencies/rates/history endpoint"
```

---

### Task 3: Kafka event on rate refresh

**Files:**
- Modify: `services/currency-service/go.mod` (add `pkg/kafka` dependency)
- Modify: `services/currency-service/go.sum` (auto-updated by tidy)
- Create: `services/currency-service/application/ports/event_publisher.go`
- Modify: `services/currency-service/application/usecases/ingest_rates.go` (inject publisher, publish after successful upsert)
- Modify: `services/currency-service/cmd/main.go` (wire Kafka producer)
- Test: update `services/currency-service/application/usecases/ingest_rates_test.go`

**Interfaces:**
- Produces:
  - `EventPublisher` port interface: `Publish(ctx, event string, payload []byte) error`
  - Kafka topic: `currency.rates.updated`
  - Event payload: `{"base":"USD","as_of":"2026-06-22","rate_count":24}`

- [ ] **Step 1: Add pkg/kafka to go.mod**

Edit `services/currency-service/go.mod`:

```
// In require block, add:
github.com/zapmarket/zapmarket/pkg/kafka v0.0.0

// In replace block, add:
github.com/zapmarket/zapmarket/pkg/kafka => ../../pkg/kafka
```

Then run:

```bash
cd services/currency-service
go mod tidy
```

Expected: go.mod and go.sum updated, no errors.

- [ ] **Step 2: Create the EventPublisher port**

```go
// services/currency-service/application/ports/event_publisher.go
package ports

import "context"

// EventPublisher publishes domain events to an async transport (e.g. Kafka).
type EventPublisher interface {
    Publish(ctx context.Context, event string, payload []byte) error
}
```

- [ ] **Step 3: Write the failing test**

Find the existing ingest rates test file at `services/currency-service/application/usecases/ingest_rates_test.go` and add:

```go
type capturePublisher struct {
    event   string
    payload []byte
}

func (c *capturePublisher) Publish(_ context.Context, event string, payload []byte) error {
    c.event = event
    c.payload = payload
    return nil
}

func TestIngestRates_PublishesEventOnSuccess(t *testing.T) {
    provider  := &fakeProvider{result: ports.RateSet{Base: "USD", Rates: map[string]float64{"EUR": 0.9}}}
    repo      := &fakeRatesRepo{}
    cache     := &fakeCache{}
    publisher := &capturePublisher{}

    uc := usecases.NewIngestRatesUseCase(provider, repo, cache, publisher)
    if err := uc.Execute(context.Background(), "USD"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if publisher.event != "currency.rates.updated" {
        t.Errorf("expected event currency.rates.updated, got %q", publisher.event)
    }
    if len(publisher.payload) == 0 {
        t.Error("expected non-empty payload")
    }
}

func TestIngestRates_DoesNotPublishOnProviderError(t *testing.T) {
    provider  := &fakeProvider{err: errors.New("provider down")}
    repo      := &fakeRatesRepo{}
    cache     := &fakeCache{}
    publisher := &capturePublisher{}

    uc := usecases.NewIngestRatesUseCase(provider, repo, cache, publisher)
    _ = uc.Execute(context.Background(), "USD")
    if publisher.event != "" {
        t.Errorf("expected no event published on provider error, got %q", publisher.event)
    }
}
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd services/currency-service
go test ./application/usecases/... -run TestIngestRates_Publish -v
```

Expected: FAIL (NewIngestRatesUseCase signature mismatch or method not found)

- [ ] **Step 5: Update IngestRatesUseCase to accept and use publisher**

Read the existing `services/currency-service/application/usecases/ingest_rates.go` first, then update:

```go
package usecases

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/zapmarket/zapmarket/services/currency-service/application/ports"
    "github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
    "github.com/zapmarket/zapmarket/services/currency-service/domain/repositories"
)

type IngestRatesUseCase struct {
    provider  ports.RatesProvider
    repo      repositories.RatesRepository
    cache     ports.RatesCache
    publisher ports.EventPublisher
}

func NewIngestRatesUseCase(
    provider  ports.RatesProvider,
    repo      repositories.RatesRepository,
    cache     ports.RatesCache,
    publisher ports.EventPublisher,
) *IngestRatesUseCase {
    return &IngestRatesUseCase{provider: provider, repo: repo, cache: cache, publisher: publisher}
}

func (uc *IngestRatesUseCase) Execute(ctx context.Context, base string) error {
    rs, err := uc.provider.FetchLatest(ctx, base)
    if err != nil {
        return fmt.Errorf("ingest: fetch: %w", err)
    }

    rates := make([]entities.ExchangeRate, 0, len(rs.Rates))
    for quote, rate := range rs.Rates {
        if quote == base {
            continue
        }
        rates = append(rates, entities.ExchangeRate{
            Base: base, Quote: quote, Rate: rate, AsOf: rs.AsOf, FetchedAt: time.Now().UTC(),
        })
    }

    if err := uc.repo.UpsertLatest(ctx, rates, rs.AsOf); err != nil {
        return fmt.Errorf("ingest: upsert: %w", err)
    }

    cached := ports.CachedRates{Base: base, AsOf: rs.AsOf, Rates: rs.Rates}
    _ = uc.cache.Set(ctx, base, cached, 2*time.Hour)

    // Publish event after successful persistence.
    payload, _ := json.Marshal(map[string]interface{}{
        "base":       base,
        "as_of":      rs.AsOf.Format("2006-01-02"),
        "rate_count": len(rates),
    })
    _ = uc.publisher.Publish(ctx, "currency.rates.updated", payload)

    return nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/currency-service
go test ./application/usecases/... -v
```

Expected: all tests PASS

- [ ] **Step 7: Create Kafka publisher adapter**

```go
// services/currency-service/infrastructure/kafka/publisher.go
package kafka

import (
    "context"

    pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
)

// KafkaPublisher adapts pkg/kafka.Producer to the ports.EventPublisher interface.
type KafkaPublisher struct {
    producer *pkgkafka.Producer
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
    return &KafkaPublisher{producer: pkgkafka.NewProducer(brokers, topic)}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event string, payload []byte) error {
    return p.producer.Publish(ctx, pkgkafka.Message{
        Key:     []byte(event),
        Value:   payload,
        Headers: map[string]string{"event_type": event},
    })
}

func (p *KafkaPublisher) Close() error {
    return p.producer.Close()
}
```

Create `services/currency-service/infrastructure/kafka/` directory if it doesn't exist.

- [ ] **Step 8: Create no-op publisher for when Kafka is not configured**

```go
// services/currency-service/infrastructure/kafka/noop_publisher.go
package kafka

import "context"

// NoopPublisher discards all events. Used when KAFKA_BROKERS is not set.
type NoopPublisher struct{}

func (n *NoopPublisher) Publish(_ context.Context, _ string, _ []byte) error { return nil }
```

- [ ] **Step 9: Wire publisher in cmd/main.go**

```go
// In cmd/main.go, after config load:
import (
    kafkainfra "github.com/zapmarket/zapmarket/services/currency-service/infrastructure/kafka"
    "strings"
)

// Wire publisher (optional — skip if KAFKA_BROKERS not set):
var publisher ports.EventPublisher = &kafkainfra.NoopPublisher{}
if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
    kp := kafkainfra.NewKafkaPublisher(strings.Split(brokers, ","), "currency.rates.updated")
    defer kp.Close()
    publisher = kp
}

// Pass publisher to use case:
ingestRatesUC := usecases.NewIngestRatesUseCase(provider, ratesRepo, redisCache, publisher)
```

Also add `KAFKA_BROKERS` env var to `docker-compose.yml` under `currency-service`:

```yaml
- KAFKA_BROKERS=kafka:9092
```

And add `depends_on` for kafka:

```yaml
depends_on:
  postgres:
    condition: service_healthy
  redis:
    condition: service_healthy
  kafka:
    condition: service_healthy
```

- [ ] **Step 10: Build to verify**

```bash
cd services/currency-service
go build ./...
```

Expected: no errors

- [ ] **Step 11: Commit**

```bash
git add services/currency-service/go.mod \
        services/currency-service/go.sum \
        services/currency-service/application/ports/event_publisher.go \
        services/currency-service/application/usecases/ingest_rates.go \
        services/currency-service/application/usecases/ingest_rates_test.go \
        services/currency-service/infrastructure/kafka/ \
        services/currency-service/cmd/main.go \
        docker-compose.yml
git commit -m "feat(currency): publish currency.rates.updated Kafka event after ingestion"
```

---

### Task 4: protoc-generated proto stubs

**Files:**
- Modify: `services/currency-service/proto/currencypb/currency.pb.go` (replace hand-written stubs with protoc output)
- Modify: `services/currency-service/proto/currency.proto` (ensure it matches the hand-written stubs)

**Note:** This task requires `protoc` and `protoc-gen-go` / `protoc-gen-go-grpc` installed on the machine. Verify first:

```bash
protoc --version
protoc-gen-go --version
protoc-gen-go-grpc --version
```

If not installed, skip this task and leave a TODO comment in `currency.pb.go`.

- [ ] **Step 1: Check proto toolchain**

```bash
protoc --version && protoc-gen-go --version && protoc-gen-go-grpc --version
```

If any command fails, install:

```bash
# Install protoc-gen-go and protoc-gen-go-grpc:
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

`protoc` itself must be installed via OS package manager (e.g. `choco install protoc` on Windows).

- [ ] **Step 2: Verify proto file matches current hand-written stubs**

Read `services/currency-service/proto/currency.proto` and confirm it has:
- `service CurrencyService` with `ListCurrencies` and `GetRates` RPCs
- `message CurrencyProto` with fields: `code`, `name`, `flag`, `decimals`, `enabled`
- `message GetRatesRequest` with `base` field
- `message GetRatesResponse` with `base`, `as_of`, `stale`, and `map<string,double> rates`

If proto file is missing fields, add them before running protoc.

- [ ] **Step 3: Generate stubs**

```bash
cd services/currency-service/proto
protoc --go_out=. --go-grpc_out=. currency.proto
```

Expected: `currencypb/currency.pb.go` and `currencypb/currency_grpc.pb.go` generated.

- [ ] **Step 4: Update grpc/server.go to use generated registration**

Replace the `grpc.ServiceDesc` approach with the generated `RegisterCurrencyServiceServer`:

```go
// In interfaces/grpc/server.go, replace manual ServiceDesc with:
import currencypb "github.com/zapmarket/zapmarket/services/currency-service/proto/currencypb"

currencypb.RegisterCurrencyServiceServer(grpcServer, &CurrencyServer{...})
```

Update handler method signatures to match generated interfaces.

- [ ] **Step 5: Build and test**

```bash
cd services/currency-service
go build ./...
go test ./...
```

Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add services/currency-service/proto/
git commit -m "feat(currency): replace hand-written proto stubs with protoc-generated output"
```
