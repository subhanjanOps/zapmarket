package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type RatesRepository struct{ db *sql.DB }

func NewRatesRepository(db *sql.DB) *RatesRepository {
	return &RatesRepository{db: db}
}

// LatestByBase returns all exchange rates for the given base currency.
// The second return value is MAX(as_of) across all returned rows, computed in SQL.
func (r *RatesRepository) LatestByBase(ctx context.Context, base string) ([]entities.ExchangeRate, time.Time, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT base, quote, rate, as_of, fetched_at, MAX(as_of) OVER () AS latest_as_of
		 FROM exchange_rates WHERE base = $1 ORDER BY quote`, base)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("latest rates: %w", err)
	}
	defer rows.Close()

	var out []entities.ExchangeRate
	var latestAsOf time.Time
	for rows.Next() {
		var e entities.ExchangeRate
		if err := rows.Scan(&e.Base, &e.Quote, &e.Rate, &e.AsOf, &e.FetchedAt, &latestAsOf); err != nil {
			return nil, time.Time{}, fmt.Errorf("latest rates scan: %w", err)
		}
		out = append(out, e)
	}
	return out, latestAsOf, rows.Err()
}

// UpsertLatest persists a batch of exchange rates using unnest() for a single-statement
// upsert — avoiding N round-trips per ingest cycle.
func (r *RatesRepository) UpsertLatest(ctx context.Context, rates []entities.ExchangeRate, asOf time.Time) error {
	if len(rates) == 0 {
		return nil
	}

	bases := make([]string, len(rates))
	quotes := make([]string, len(rates))
	rateVals := make([]float64, len(rates))
	for i, rate := range rates {
		bases[i] = rate.Base
		quotes[i] = rate.Quote
		rateVals[i] = rate.Rate
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("upsert rates begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO exchange_rates (base, quote, rate, as_of, fetched_at)
		SELECT unnest($1::text[]), unnest($2::text[]), unnest($3::float8[]), $4, $5
		ON CONFLICT (base, quote) DO UPDATE
		  SET rate       = EXCLUDED.rate,
		      as_of      = EXCLUDED.as_of,
		      fetched_at = EXCLUDED.fetched_at`,
		pq.Array(bases), pq.Array(quotes), pq.Array(rateVals), asOf, now)
	if err != nil {
		return fmt.Errorf("upsert rates: %w", err)
	}

	// Record history; idempotent — duplicate (base, quote, as_of) rows are silently skipped.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO exchange_rates_history (base, quote, rate, as_of, fetched_at)
		SELECT unnest($1::text[]), unnest($2::text[]), unnest($3::float8[]), $4, $5
		ON CONFLICT (base, quote, as_of) DO NOTHING`,
		pq.Array(bases), pq.Array(quotes), pq.Array(rateVals), asOf, now)
	if err != nil {
		return fmt.Errorf("history insert: %w", err)
	}

	return tx.Commit()
}

// HistoryByBase returns all exchange rates for the given base currency on the given calendar date.
func (r *RatesRepository) HistoryByBase(ctx context.Context, base string, date time.Time) ([]entities.ExchangeRate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT base, quote, rate, as_of, fetched_at
		 FROM exchange_rates_history
		 WHERE base = $1 AND as_of::date = $2::date
		 ORDER BY quote`,
		base, date)
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
