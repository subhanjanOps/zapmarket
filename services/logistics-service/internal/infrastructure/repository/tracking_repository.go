package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
)

type TrackingRepository struct{ db *sql.DB }

func NewTrackingRepository(db *sql.DB) *TrackingRepository { return &TrackingRepository{db: db} }

func (r *TrackingRepository) Insert(ctx context.Context, shipmentID, status, description, location string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tracking_events (id, shipment_id, status, description, location, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.NewString(), shipmentID, status, description, location, time.Now(),
	)
	return err
}

func (r *TrackingRepository) List(ctx context.Context, shipmentID string) ([]*domain.TrackingEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT status, description, location, occurred_at
		 FROM tracking_events WHERE shipment_id = $1 ORDER BY occurred_at DESC`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*domain.TrackingEvent
	for rows.Next() {
		e := &domain.TrackingEvent{}
		if err := rows.Scan(&e.Status, &e.Description, &e.Location, &e.OccurredAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
