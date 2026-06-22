package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type RatesRepository struct{ db *sql.DB }

func NewRatesRepository(db *sql.DB) *RatesRepository {
	return &RatesRepository{db: db}
}

func (r *RatesRepository) LatestByBase(ctx context.Context, base string) ([]entities.ExchangeRate, time.Time, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT base, quote, rate, as_of, fetched_at
		 FROM exchange_rates WHERE base = $1 ORDER BY quote`, base)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("latest rates: %w", err)
	}
	defer rows.Close()

	var out []entities.ExchangeRate
	var latestAsOf time.Time
	for rows.Next() {
		var e entities.ExchangeRate
		if err := rows.Scan(&e.Base, &e.Quote, &e.Rate, &e.AsOf, &e.FetchedAt); err != nil {
			return nil, time.Time{}, fmt.Errorf("latest rates scan: %w", err)
		}
		out = append(out, e)
		if e.AsOf.After(latestAsOf) {
			latestAsOf = e.AsOf
		}
	}
	return out, latestAsOf, rows.Err()
}

func (r *RatesRepository) UpsertLatest(ctx context.Context, rates []entities.ExchangeRate, asOf time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("upsert rates begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC()
	for _, rate := range rates {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO exchange_rates (base, quote, rate, as_of, fetched_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (base, quote) DO UPDATE
			  SET rate = EXCLUDED.rate,
			      as_of = EXCLUDED.as_of,
			      fetched_at = EXCLUDED.fetched_at`,
			rate.Base, rate.Quote, rate.Rate, asOf, now)
		if err != nil {
			return fmt.Errorf("upsert rate %s/%s: %w", rate.Base, rate.Quote, err)
		}
	}

	// Also record history (idempotent via ON CONFLICT DO NOTHING).
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

	return tx.Commit()
}

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
