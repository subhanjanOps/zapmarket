package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db}
}

func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Payment, error) {
	return scanPayment(r.db.QueryRowContext(ctx, paymentSelectQuery+" WHERE idempotency_key = $1 AND deleted_at IS NULL", key))
}

func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	return scanPayment(r.db.QueryRowContext(ctx, paymentSelectQuery+" WHERE id = $1 AND deleted_at IS NULL", id))
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	id := uuid.New()

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO payments (id, order_id, user_id, idempotency_key, status, amount, currency, gateway)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`, id, p.OrderID, p.UserID, p.IdempotencyKey, p.Status, p.Amount, p.Currency, p.Gateway).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to create payment", err)
	}

	p.ID = id
	return nil
}

func (r *PaymentRepository) MarkCaptured(ctx context.Context, paymentID uuid.UUID, gatewayTxnID string, entries []*domain.LedgerEntry) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var orderID, userID uuid.UUID
		err := tx.QueryRowContext(ctx, `
			UPDATE payments
			SET status = 'CAPTURED',
				gateway_txn_id = $2,
				updated_at = NOW()
			WHERE id = $1
				AND deleted_at IS NULL
			RETURNING order_id, user_id
		`, paymentID, gatewayTxnID).Scan(&orderID, &userID)
		if errors.Is(err, sql.ErrNoRows) {
			return pkgerrors.NewNotFound("PAYMENT_NOT_FOUND", "payment not found")
		}
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to mark payment captured", err)
		}

		for _, e := range entries {
			if err := insertLedgerEntry(ctx, tx, paymentID, e); err != nil {
				return err
			}
		}

		payload, err := json.Marshal(map[string]string{
			"payment_id": paymentID.String(), "order_id": orderID.String(),
			"user_id": userID.String(), "gateway_txn_id": gatewayTxnID,
			"status": "CAPTURED",
		})
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}
		return insertOutboxRow(ctx, tx, paymentID, "payment", "payment.captured", payload)
	})
}

func (r *PaymentRepository) MarkFailed(ctx context.Context, paymentID uuid.UUID, reason string) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var orderID, userID uuid.UUID
		err := tx.QueryRowContext(ctx, `
			UPDATE payments
			SET status = 'FAILED',
				failure_reason = $2,
				updated_at = NOW()
			WHERE id = $1
				AND deleted_at IS NULL
			RETURNING order_id, user_id
		`, paymentID, reason).Scan(&orderID, &userID)
		if errors.Is(err, sql.ErrNoRows) {
			return pkgerrors.NewNotFound("PAYMENT_NOT_FOUND", "payment not found")
		}
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to mark payment failed", err)
		}

		payload, err := json.Marshal(map[string]string{
			"payment_id": paymentID.String(), "order_id": orderID.String(),
			"user_id": userID.String(), "reason": reason, "status": "FAILED",
		})
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}
		return insertOutboxRow(ctx, tx, paymentID, "payment", "payment.failed", payload)
	})
}

func (r *PaymentRepository) CreateRefund(ctx context.Context, refund *domain.Refund, userID uuid.UUID, newPaymentStatus domain.PaymentStatus, entries []*domain.LedgerEntry) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		id := uuid.New()

		err := tx.QueryRowContext(ctx, `
			INSERT INTO refunds (id, payment_id, order_id, amount, currency, reason, status, gateway_refund_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING created_at, updated_at
		`, id, refund.PaymentID, refund.OrderID, refund.Amount, refund.Currency, refund.Reason, refund.Status, refund.GatewayRefundID).
			Scan(&refund.CreatedAt, &refund.UpdatedAt)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to create refund", err)
		}
		refund.ID = id

		result, err := tx.ExecContext(ctx, `
			UPDATE payments SET status = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL
		`, refund.PaymentID, newPaymentStatus)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to update payment status for refund", err)
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return pkgerrors.NewNotFound("PAYMENT_NOT_FOUND", "payment not found")
		}

		for _, e := range entries {
			if err := insertLedgerEntry(ctx, tx, refund.PaymentID, e); err != nil {
				return err
			}
		}

		payload, err := json.Marshal(map[string]string{
			"payment_id": refund.PaymentID.String(),
			"refund_id":  id.String(),
			"order_id":   refund.OrderID.String(),
			"user_id":    userID.String(),
			"amount":     fmt.Sprintf("%d", refund.Amount),
			"currency":   refund.Currency,
			"status":     "REFUNDED",
		})
		if err != nil {
			return fmt.Errorf("marshal refund outbox payload: %w", err)
		}
		return insertOutboxRow(ctx, tx, id, "payment", "payment.refunded", payload)
	})
}

const paymentSelectQuery = `
	SELECT id, order_id, user_id, idempotency_key, status, amount, currency, gateway, gateway_txn_id, failure_reason, created_at, updated_at
	FROM payments
`

func scanPayment(row *sql.Row) (*domain.Payment, error) {
	p := &domain.Payment{}

	err := row.Scan(
		&p.ID, &p.OrderID, &p.UserID, &p.IdempotencyKey, &p.Status, &p.Amount, &p.Currency, &p.Gateway,
		&p.GatewayTxnID, &p.FailureReason, &p.CreatedAt, &p.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, pkgerrors.NewNotFound("PAYMENT_NOT_FOUND", "payment not found")
	}
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get payment", err)
	}

	return p, nil
}

func insertOutboxRow(ctx context.Context, tx *sql.Tx, aggregateID uuid.UUID, aggregateType, eventType string, payload []byte) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload)
		VALUES ($1, $2, $3, $4)
	`, aggregateID, aggregateType, eventType, payload)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to write outbox row", err)
	}
	return nil
}

func insertLedgerEntry(ctx context.Context, tx *sql.Tx, paymentID uuid.UUID, e *domain.LedgerEntry) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (payment_id, entry_type, account, amount, currency, description)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, paymentID, e.EntryType, e.Account, e.Amount, e.Currency, sql.NullString{String: e.Description, Valid: e.Description != ""})
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to write ledger entry", err)
	}
	return nil
}
