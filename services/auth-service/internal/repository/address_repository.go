package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

type AddressRepository struct{ db *sql.DB }

func NewAddressRepository(db *sql.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Create(ctx context.Context, a *domain.Address) error {
	a.ID = uuid.New()
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO addresses (id, user_id, label, line1, line2, city, state, country, pincode, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		a.ID, a.UserID, a.Label, a.Line1, a.Line2, a.City, a.State, a.Country, a.Pincode, a.IsDefault, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *AddressRepository) GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*domain.Address, error) {
	a := &domain.Address{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, label, line1, line2, city, state, country, pincode, is_default, created_at, updated_at
		FROM addresses
		WHERE user_id = $1 AND is_default = true AND deleted_at IS NULL
		LIMIT 1`, userID,
	).Scan(&a.ID, &a.UserID, &a.Label, &a.Line1, &a.Line2, &a.City, &a.State, &a.Country, &a.Pincode, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.NewNotFound("NOT_FOUND", "address not found")
	}
	return a, err
}
