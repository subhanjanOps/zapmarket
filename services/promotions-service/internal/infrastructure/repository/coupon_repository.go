package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domainerrors "github.com/zapmarket/zapmarket/services/promotions-service/internal/domain/errors"

	"github.com/zapmarket/zapmarket/services/promotions-service/internal/domain"
)

type CouponRepository struct{ db *sql.DB }

func NewCouponRepository(db *sql.DB) *CouponRepository { return &CouponRepository{db: db} }

func (r *CouponRepository) FindByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, code, discount_type, discount_value, min_order_paise,
		       max_uses_total, max_uses_per_user, expires_at, is_active, created_at
		FROM coupons WHERE code = $1`, code)

	var c domain.Coupon
	var expiresAt sql.NullTime
	err := row.Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &c.MinOrderPaise,
		&c.MaxUsesTotal, &c.MaxUsesPerUser, &expiresAt, &c.IsActive, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrCouponNotFound
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		c.ExpiresAt = &t
	}
	return &c, nil
}

func (r *CouponRepository) CountUsageByUser(ctx context.Context, couponID, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_usage WHERE coupon_id = $1 AND user_id = $2`, couponID, userID).Scan(&n)
	return n, err
}

func (r *CouponRepository) CountUsageTotal(ctx context.Context, couponID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM coupon_usage WHERE coupon_id = $1`, couponID).Scan(&n)
	return n, err
}

var ErrUsageLimitReached = errors.New("coupon usage limit has been reached")

// RecordUsage atomically re-checks the coupon's usage limits and inserts the
// usage row within a single transaction, locking the coupon row so
// concurrent redemptions of the same coupon are serialized. This prevents
// over-redemption past max_uses_total/max_uses_per_user under concurrency.
func (r *CouponRepository) RecordUsage(ctx context.Context, couponID, userID, orderID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var maxTotal, maxPerUser int
	if err := tx.QueryRowContext(ctx,
		`SELECT max_uses_total, max_uses_per_user FROM coupons WHERE id = $1 FOR UPDATE`, couponID,
	).Scan(&maxTotal, &maxPerUser); err != nil {
		if err == sql.ErrNoRows {
			return domainerrors.ErrCouponNotFound
		}
		return err
	}

	if maxPerUser > 0 {
		var used int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM coupon_usage WHERE coupon_id = $1 AND user_id = $2`, couponID, userID,
		).Scan(&used); err != nil {
			return err
		}
		if used >= maxPerUser {
			return ErrUsageLimitReached
		}
	}

	if maxTotal > 0 {
		var total int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM coupon_usage WHERE coupon_id = $1`, couponID,
		).Scan(&total); err != nil {
			return err
		}
		if total >= maxTotal {
			return ErrUsageLimitReached
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO coupon_usage (coupon_id, user_id, order_id, used_at) VALUES ($1,$2,$3,$4)`,
		couponID, userID, orderID, time.Now(),
	); err != nil {
		return err
	}

	return tx.Commit()
}

// GetActiveSales returns PERCENT coupons whose sale window is currently active.
func (r *CouponRepository) GetActiveSales(ctx context.Context) ([]*domain.Coupon, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, discount_type, discount_value, min_order_paise,
		       max_uses_total, max_uses_per_user, expires_at, starts_at, ends_at, is_active, created_at
		FROM coupons
		WHERE discount_type = 'PERCENT'
		  AND is_active = TRUE
		  AND starts_at IS NOT NULL
		  AND ends_at IS NOT NULL
		  AND starts_at <= NOW()
		  AND ends_at >= NOW()
		ORDER BY ends_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Coupon
	for rows.Next() {
		var c domain.Coupon
		var expiresAt, startsAt, endsAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &c.MinOrderPaise,
			&c.MaxUsesTotal, &c.MaxUsesPerUser, &expiresAt, &startsAt, &endsAt, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			t := expiresAt.Time
			c.ExpiresAt = &t
		}
		if startsAt.Valid {
			t := startsAt.Time
			c.StartsAt = &t
		}
		if endsAt.Valid {
			t := endsAt.Time
			c.EndsAt = &t
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
