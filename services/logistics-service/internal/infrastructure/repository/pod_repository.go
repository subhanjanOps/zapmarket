package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
)

type PODRepository struct{ db *sql.DB }

func NewPODRepository(db *sql.DB) *PODRepository { return &PODRepository{db: db} }

func (r *PODRepository) Create(ctx context.Context, p *domain.ProofOfDelivery) error {
	p.ID = uuid.NewString()
	p.DeliveredAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO proof_of_delivery (id, shipment_id, method, otp_verified, photo_url, delivered_at, delivered_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.ID, p.ShipmentID, p.Method, p.OTPVerified, p.PhotoURL, p.DeliveredAt, p.DeliveredBy,
	)
	return err
}

func (r *PODRepository) CreateTx(ctx context.Context, tx *sql.Tx, p *domain.ProofOfDelivery) error {
	p.ID = uuid.NewString()
	p.DeliveredAt = time.Now()
	_, err := tx.ExecContext(ctx,
		`INSERT INTO proof_of_delivery (id, shipment_id, method, otp_verified, photo_url, delivered_at, delivered_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.ID, p.ShipmentID, p.Method, p.OTPVerified, p.PhotoURL, p.DeliveredAt, p.DeliveredBy,
	)
	return err
}
