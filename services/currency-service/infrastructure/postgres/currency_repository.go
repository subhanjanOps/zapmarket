package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domainerrors "github.com/zapmarket/zapmarket/services/currency-service/domain/errors"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type CurrencyRepository struct{ db *sql.DB }

func NewCurrencyRepository(db *sql.DB) *CurrencyRepository {
	return &CurrencyRepository{db: db}
}

func (r *CurrencyRepository) ListEnabled(ctx context.Context) ([]entities.Currency, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT code, name, flag, decimals, enabled, created_at, updated_at
		 FROM currencies WHERE enabled = true ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list currencies: %w", err)
	}
	defer rows.Close()

	var out []entities.Currency
	for rows.Next() {
		var c entities.Currency
		if err := rows.Scan(&c.Code, &c.Name, &c.Flag, &c.Decimals, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list currencies scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CurrencyRepository) Get(ctx context.Context, code string) (entities.Currency, error) {
	var c entities.Currency
	err := r.db.QueryRowContext(ctx,
		`SELECT code, name, flag, decimals, enabled, created_at, updated_at
		 FROM currencies WHERE code = $1`, code).
		Scan(&c.Code, &c.Name, &c.Flag, &c.Decimals, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return entities.Currency{}, &domainerrors.ErrCurrencyNotFound{Code: code}
	}
	if err != nil {
		return entities.Currency{}, fmt.Errorf("get currency: %w", err)
	}
	return c, nil
}

func (r *CurrencyRepository) Toggle(ctx context.Context, code string, enabled bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE currencies SET enabled = $1, updated_at = $2 WHERE code = $3`,
		enabled, time.Now().UTC(), code)
	if err != nil {
		return fmt.Errorf("toggle currency: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("toggle currency: rows affected: %w", err)
	}
	if n == 0 {
		return &domainerrors.ErrCurrencyNotFound{Code: code}
	}
	return nil
}
