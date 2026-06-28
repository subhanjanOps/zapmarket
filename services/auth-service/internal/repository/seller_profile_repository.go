package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

type SellerProfileRepository struct{ db *sql.DB }

func NewSellerProfileRepository(db *sql.DB) *SellerProfileRepository {
	return &SellerProfileRepository{db: db}
}

func (r *SellerProfileRepository) Create(ctx context.Context, p *domain.SellerProfile) error {
	p.ID = uuid.New()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO seller_profiles
			(id, user_id, store_name, tagline, category, gstin, pan, business_phone, city, pincode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		p.ID, p.UserID, p.StoreName, p.Tagline, p.Category,
		p.GSTIN, p.PAN, p.BusinessPhone, p.City, p.Pincode,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *SellerProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.SellerProfile, error) {
	p := &domain.SellerProfile{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, store_name, tagline, category, gstin, pan, business_phone, city, pincode, created_at, updated_at
		FROM seller_profiles
		WHERE user_id = $1`, userID,
	).Scan(
		&p.ID, &p.UserID, &p.StoreName, &p.Tagline, &p.Category,
		&p.GSTIN, &p.PAN, &p.BusinessPhone, &p.City, &p.Pincode,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.NewNotFound("NOT_FOUND", "seller profile not found")
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
