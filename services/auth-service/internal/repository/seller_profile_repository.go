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
	if p.BusinessType == "" {
		p.BusinessType = "individual"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO seller_profiles
			(id, user_id, store_name, tagline, category, gstin, pan, business_phone, city, pincode,
			 business_type, tax_id, biz_line1, biz_city, biz_state, biz_country, biz_pincode,
			 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		p.ID, p.UserID, p.StoreName, p.Tagline, p.Category, p.GSTIN, p.PAN, p.BusinessPhone, p.City, p.Pincode,
		p.BusinessType, p.TaxID, p.BizLine1, p.BizCity, p.BizState, p.BizCountry, p.BizPincode,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *SellerProfileRepository) Update(ctx context.Context, p *domain.SellerProfile) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE seller_profiles SET
			store_name=$1, tagline=$2, category=$3, gstin=$4, pan=$5,
			business_phone=$6, city=$7, pincode=$8,
			business_type=$9, tax_id=$10,
			biz_line1=$11, biz_city=$12, biz_state=$13, biz_country=$14, biz_pincode=$15,
			updated_at=$16
		WHERE user_id=$17`,
		p.StoreName, p.Tagline, p.Category, p.GSTIN, p.PAN,
		p.BusinessPhone, p.City, p.Pincode,
		p.BusinessType, p.TaxID,
		p.BizLine1, p.BizCity, p.BizState, p.BizCountry, p.BizPincode,
		p.UpdatedAt, p.UserID,
	)
	return err
}

func (r *SellerProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.SellerProfile, error) {
	p := &domain.SellerProfile{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, store_name, tagline, category, gstin, pan, business_phone, city, pincode,
		       business_type, tax_id, biz_line1, biz_city, biz_state, biz_country, biz_pincode,
		       created_at, updated_at
		FROM seller_profiles
		WHERE user_id = $1`, userID,
	).Scan(
		&p.ID, &p.UserID, &p.StoreName, &p.Tagline, &p.Category,
		&p.GSTIN, &p.PAN, &p.BusinessPhone, &p.City, &p.Pincode,
		&p.BusinessType, &p.TaxID, &p.BizLine1, &p.BizCity, &p.BizState, &p.BizCountry, &p.BizPincode,
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
