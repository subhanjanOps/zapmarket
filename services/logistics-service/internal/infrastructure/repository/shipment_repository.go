package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
)

type ShipmentRepository struct{ db *sql.DB }

func NewShipmentRepository(db *sql.DB) *ShipmentRepository { return &ShipmentRepository{db: db} }

func (r *ShipmentRepository) Save(ctx context.Context, s *domain.Shipment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO shipments (id, order_id, carrier_shipment_id, carrier, tracking_url, status, label_url, estimated_delivery, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.OrderID, s.CarrierShipmentID, s.Carrier, s.TrackingURL, string(s.Status), s.LabelURL, s.EstimatedDelivery, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *ShipmentRepository) GetByID(ctx context.Context, id string) (*domain.Shipment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, order_id, carrier_shipment_id, carrier, tracking_url, status, label_url,
		        estimated_delivery, assigned_agent_id, attempt_count, next_attempt_at, last_attempt_at,
		        COALESCE(shipment_type, 'FORWARD'), parent_shipment_id,
		        created_at, updated_at
		 FROM shipments WHERE id = $1`, id)
	return scanShipment(row)
}

func (r *ShipmentRepository) AssignAgent(ctx context.Context, shipmentID, agentID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE shipments SET assigned_agent_id = $1, status = 'OUT_FOR_DELIVERY', updated_at = NOW() WHERE id = $2`,
		agentID, shipmentID,
	)
	return err
}

// RecordAttempt updates delivery attempt state. Returns (undelivered bool, err).
// If success, status = DELIVERED.
// If failure and attempt_count < 3, status = REATTEMPT_SCHEDULED with next_attempt_at = now+24h.
// If failure and attempt_count >= 3, status = UNDELIVERED.
func (r *ShipmentRepository) RecordAttempt(ctx context.Context, shipmentID string, success bool) (undelivered bool, err error) {
	// Fetch current attempt count
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT attempt_count FROM shipments WHERE id = $1`, shipmentID).Scan(&count); err != nil {
		return false, err
	}
	newCount := count + 1
	now := time.Now()
	var newStatus string
	var nextAttempt *time.Time
	switch {
	case success:
		newStatus = "DELIVERED"
	case newCount < 3:
		newStatus = "REATTEMPT_SCHEDULED"
		t := now.Add(24 * time.Hour)
		nextAttempt = &t
	default:
		newStatus = "UNDELIVERED"
		undelivered = true
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE shipments
		 SET attempt_count = $1, last_attempt_at = $2, next_attempt_at = $3, status = $4, updated_at = NOW()
		 WHERE id = $5`,
		newCount, now, nextAttempt, newStatus, shipmentID,
	)
	return undelivered, err
}

// WriteUndeliveredEvent writes a shipment.undelivered outbox event.
func (r *ShipmentRepository) WriteUndeliveredEvent(ctx context.Context, shipmentID, orderID string) error {
	payload, _ := json.Marshal(map[string]string{"shipment_id": shipmentID, "order_id": orderID})
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO outbox (id, topic, payload, created_at) VALUES ($1, 'shipment.undelivered', $2, $3)`,
		uuid.NewString(), payload, time.Now(),
	)
	return err
}

// CreateReverse inserts a reverse shipment linked to a parent.
func (r *ShipmentRepository) CreateReverse(ctx context.Context, parentID, orderID, carrierShipmentID, carrier, trackingURL string) (*domain.Shipment, error) {
	s := &domain.Shipment{
		ID:                uuid.NewString(),
		OrderID:           orderID,
		CarrierShipmentID: carrierShipmentID,
		Carrier:           carrier,
		TrackingURL:       trackingURL,
		Status:            domain.ShipmentStatusCreated,
		ShipmentType:      "REVERSE",
		ParentShipmentID:  &parentID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO shipments (id, order_id, carrier_shipment_id, carrier, tracking_url, status, shipment_type, parent_shipment_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,'REVERSE',$7,$8,$9)`,
		s.ID, s.OrderID, s.CarrierShipmentID, s.Carrier, s.TrackingURL, string(s.Status),
		parentID, s.CreatedAt, s.UpdatedAt,
	)
	return s, err
}

func (r *ShipmentRepository) MarkDelivered(ctx context.Context, shipmentID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE shipments SET status = 'DELIVERED', updated_at = NOW() WHERE id = $1`, shipmentID)
	return err
}

// WriteDeliveredEvent writes a shipment.delivered outbox event atomically (caller should use a tx, but we keep it simple).
func (r *ShipmentRepository) WriteDeliveredEvent(ctx context.Context, tx *sql.Tx, shipmentID, orderID string) error {
	payload, _ := json.Marshal(map[string]string{"shipment_id": shipmentID, "order_id": orderID})
	_, err := tx.ExecContext(ctx,
		`INSERT INTO outbox (id, topic, payload, created_at) VALUES ($1, 'shipment.delivered', $2, $3)`,
		uuid.NewString(), payload, time.Now(),
	)
	return err
}

func (r *ShipmentRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// WriteCODReconciliation inserts a cod_reconciliations row inside an existing transaction.
func (r *ShipmentRepository) WriteCODReconciliationTx(ctx context.Context, tx *sql.Tx, shipmentID, agentID string, amountPaise int64) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO cod_reconciliations (id, shipment_id, agent_id, amount_paise, collected_at, status)
		 VALUES (gen_random_uuid(), $1, $2, $3, NOW(), 'COLLECTED')`,
		shipmentID, agentID, amountPaise,
	)
	return err
}

func scanShipment(row *sql.Row) (*domain.Shipment, error) {
	s := &domain.Shipment{}
	var status string
	err := row.Scan(
		&s.ID, &s.OrderID, &s.CarrierShipmentID, &s.Carrier, &s.TrackingURL,
		&status, &s.LabelURL, &s.EstimatedDelivery,
		&s.AssignedAgentID, &s.AttemptCount, &s.NextAttemptAt, &s.LastAttemptAt,
		&s.ShipmentType, &s.ParentShipmentID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Status = domain.ShipmentStatus(status)
	return s, nil
}
