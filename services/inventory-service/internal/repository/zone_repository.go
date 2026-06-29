package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type ZoneRepository interface {
	FindWarehouseByPincode(ctx context.Context, pincode string) (uuid.UUID, error)
	DefaultWarehouseID(ctx context.Context) (uuid.UUID, error)
}

type postgresZoneRepository struct{ db *sql.DB }

func NewZoneRepository(db *sql.DB) ZoneRepository {
	return &postgresZoneRepository{db: db}
}

func (r *postgresZoneRepository) FindWarehouseByPincode(ctx context.Context, pincode string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT warehouse_id FROM pincode_zones WHERE pincode = $1`, pincode,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return r.DefaultWarehouseID(ctx)
	}
	return id, err
}

func (r *postgresZoneRepository) DefaultWarehouseID(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM warehouses WHERE is_active = true ORDER BY created_at ASC LIMIT 1`,
	).Scan(&id)
	return id, err
}
