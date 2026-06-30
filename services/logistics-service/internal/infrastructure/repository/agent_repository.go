package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
)

type AgentRepository struct{ db *sql.DB }

func NewAgentRepository(db *sql.DB) *AgentRepository { return &AgentRepository{db: db} }

func (r *AgentRepository) Create(ctx context.Context, a *domain.DeliveryAgent) error {
	a.ID = uuid.NewString()
	a.CreatedAt = time.Now()
	if a.Status == "" {
		a.Status = "AVAILABLE"
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO delivery_agents (id, user_id, name, phone, vehicle_type, zone, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		a.ID, a.UserID, a.Name, a.Phone, a.VehicleType, a.Zone, a.Status, a.CreatedAt,
	)
	return err
}

func (r *AgentRepository) List(ctx context.Context) ([]*domain.DeliveryAgent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, phone, vehicle_type, zone, status, created_at
		 FROM delivery_agents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []*domain.DeliveryAgent
	for rows.Next() {
		a := &domain.DeliveryAgent{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Phone, &a.VehicleType, &a.Zone, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}
