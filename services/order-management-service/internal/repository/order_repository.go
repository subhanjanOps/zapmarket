package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db}
}

func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Order, error) {
	return scanOrder(r.db.QueryRowContext(ctx,
		orderSelectQuery+" WHERE idempotency_key = $1 AND deleted_at IS NULL", key))
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return scanOrder(r.db.QueryRowContext(ctx,
		orderSelectQuery+" WHERE id = $1 AND deleted_at IS NULL", id))
}

func (r *OrderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error) {
	limit, offset := pageArgs(p)
	var total int64
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM orders WHERE user_id = $1 AND deleted_at IS NULL", userID,
	).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to count orders", err)
	}
	rows, err := r.db.QueryContext(ctx,
		orderSelectQuery+" WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		userID, limit, offset)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to list orders", err)
	}
	defer rows.Close()
	var orders []*domain.Order
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func (r *OrderRepository) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, sku_id, seller_id, quantity, unit_price, reservation_id, created_at
		FROM order_items
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get order items", err)
	}
	defer rows.Close()

	var items []*domain.OrderItem
	for rows.Next() {
		item := &domain.OrderItem{}
		err := rows.Scan(&item.ID, &item.OrderID, &item.SKUID, &item.SellerID, &item.Quantity, &item.UnitPrice, &item.ReservationID, &item.CreatedAt)
		if err != nil {
			return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan order item", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *OrderRepository) GetBySellerID(ctx context.Context, sellerID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error) {
	limit, offset := pageArgs(p)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE id IN (SELECT DISTINCT order_id FROM order_items WHERE seller_id = $1)
		AND deleted_at IS NULL`, sellerID,
	).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to count seller orders", err)
	}
	rows, err := r.db.QueryContext(ctx,
		orderSelectQuery+` WHERE id IN (
			SELECT DISTINCT order_id FROM order_items WHERE seller_id = $1
		) AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		sellerID, limit, offset)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to list seller orders", err)
	}
	defer rows.Close()
	var orders []*domain.Order
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func pageArgs(p contracts.OrderPageParams) (limit, offset int) {
	limit = p.Limit
	if limit <= 0 {
		limit = 20
	}
	offset = p.Offset
	if offset < 0 {
		offset = 0
	}
	return
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		id := uuid.New()
		err := tx.QueryRowContext(ctx, `
			INSERT INTO orders (id, user_id, idempotency_key, status, total_amount, currency)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING created_at, updated_at
		`, id, order.UserID, order.IdempotencyKey, order.Status, order.TotalAmount, order.Currency).
			Scan(&order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to create order", err)
		}
		order.ID = id

		for _, item := range items {
			itemID := uuid.New()
			err := tx.QueryRowContext(ctx, `
				INSERT INTO order_items (id, order_id, sku_id, seller_id, quantity, unit_price)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING created_at
			`, itemID, id, item.SKUID, item.SellerID, item.Quantity, item.UnitPrice).Scan(&item.CreatedAt)
			if err != nil {
				return pkgerrors.NewInternal("DATABASE_ERROR", "failed to create order item", err)
			}
			item.ID = itemID
			item.OrderID = id
		}
		return nil
	})
}

func (r *OrderRepository) MarkReserved(ctx context.Context, orderID uuid.UUID, items []*domain.OrderItem) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE orders SET status = 'RESERVED', updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, orderID)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to mark order reserved", err)
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
		}

		for _, item := range items {
			if item.ReservationID == nil {
				continue
			}
			_, err := tx.ExecContext(ctx, `
				UPDATE order_items SET reservation_id = $2 WHERE id = $1
			`, item.ID, item.ReservationID)
			if err != nil {
				return pkgerrors.NewInternal("DATABASE_ERROR", "failed to update order item reservation", err)
			}
		}
		return nil
	})
}

func (r *OrderRepository) MarkConfirmed(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID, outboxPayload []byte) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE orders SET status = 'CONFIRMED', payment_id = $2, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, orderID, paymentID)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to confirm order", err)
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
		}
		return insertOutboxEvent(ctx, tx, orderID, "order", "order.confirmed", outboxPayload)
	})
}

func (r *OrderRepository) MarkCancelled(ctx context.Context, orderID uuid.UUID, outboxPayload []byte) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE orders SET status = 'CANCELLED', updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, orderID)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to cancel order", err)
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
		}
		return insertOutboxEvent(ctx, tx, orderID, "order", "order.cancelled", outboxPayload)
	})
}

func insertOutboxEvent(ctx context.Context, tx *sql.Tx, aggregateID uuid.UUID, aggregateType, eventType string, payload []byte) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload)
		VALUES ($1, $2, $3, $4)
	`, aggregateID, aggregateType, eventType, payload)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to insert outbox event", err)
	}
	return nil
}

const orderSelectQuery = `
	SELECT id, user_id, idempotency_key, status, total_amount, currency, payment_id, created_at, updated_at
	FROM orders
`

func scanOrder(row *sql.Row) (*domain.Order, error) {
	o := &domain.Order{}
	err := row.Scan(
		&o.ID, &o.UserID, &o.IdempotencyKey, &o.Status,
		&o.TotalAmount, &o.Currency, &o.PaymentID,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
	}
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get order", err)
	}
	return o, nil
}

// ListAll returns all orders with optional filters — admin use only.
func (r *OrderRepository) ListAll(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error) {
	args := []any{}
	conditions := []string{"deleted_at IS NULL"}
	i := 1

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", i))
		args = append(args, *params.UserID)
		i++
	}
	if params.From != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, *params.From)
		i++
	}
	if params.To != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", i))
		args = append(args, *params.To)
		i++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders "+where, args...).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to count orders", err)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit, params.Offset)
	query := fmt.Sprintf(
		"SELECT id, user_id, idempotency_key, status, total_amount, currency, payment_id, created_at, updated_at FROM orders %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, i, i+1,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to list orders", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func scanOrderRow(rows *sql.Rows) (*domain.Order, error) {
	o := &domain.Order{}
	err := rows.Scan(
		&o.ID, &o.UserID, &o.IdempotencyKey, &o.Status,
		&o.TotalAmount, &o.Currency, &o.PaymentID,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan order", err)
	}
	return o, nil
}
