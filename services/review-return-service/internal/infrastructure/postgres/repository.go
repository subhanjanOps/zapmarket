package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/zapmarket/zapmarket/services/review-return-service/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

// ── Returns ──────────────────────────────────────────────────────────────────

func (r *Repository) CreateReturn(ctx context.Context, req *domain.ReturnRequest) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO return_requests (id, order_id, order_item_id, user_id, reason, description, status, image_urls, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'REQUESTED', $6, NOW(), NOW())
		 RETURNING id`,
		req.OrderID, req.OrderItemID, req.UserID, req.Reason, req.Description, pq.Array(req.ImageURLs),
	)
	return err
}

func (r *Repository) GetReturnByID(ctx context.Context, id string) (*domain.ReturnRequest, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, order_id, order_item_id, user_id, reason, description, status,
		        image_urls, approved_at, rejected_at, rejection_reason, reverse_shipment_id,
		        created_at, updated_at
		 FROM return_requests WHERE id = $1`, id)
	req := &domain.ReturnRequest{}
	var imageURLs pq.StringArray
	err := row.Scan(
		&req.ID, &req.OrderID, &req.OrderItemID, &req.UserID, &req.Reason, &req.Description, &req.Status,
		&imageURLs, &req.ApprovedAt, &req.RejectedAt, &req.RejectionReason, &req.ReverseShipmentID,
		&req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	req.ImageURLs = []string(imageURLs)
	return req, nil
}

func (r *Repository) ApproveReturn(ctx context.Context, id, reverseShipmentID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE return_requests SET status = 'APPROVED', approved_at = $1, reverse_shipment_id = $2, updated_at = NOW()
		 WHERE id = $3`, now, reverseShipmentID, id)
	return err
}

func (r *Repository) RejectReturn(ctx context.Context, id, reason string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE return_requests SET status = 'REJECTED', rejected_at = $1, rejection_reason = $2, updated_at = NOW()
		 WHERE id = $3`, now, reason, id)
	return err
}

// ── Ratings ──────────────────────────────────────────────────────────────────

func (r *Repository) GetProductRating(ctx context.Context, productID string) (*domain.ProductRating, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT product_id, avg_rating, review_count FROM product_ratings WHERE product_id = $1`, productID)
	pr := &domain.ProductRating{}
	if err := row.Scan(&pr.ProductID, &pr.AvgRating, &pr.ReviewCount); err != nil {
		return nil, err
	}
	return pr, nil
}

func (r *Repository) RefreshRatings(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY product_ratings`)
	return err
}
