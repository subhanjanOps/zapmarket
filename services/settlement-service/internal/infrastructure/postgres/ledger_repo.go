package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type LedgerRepo struct{ db *sql.DB }

func NewLedgerRepo(db *sql.DB) *LedgerRepo { return &LedgerRepo{db: db} }

// ApplyLedgerEntry inserts the ledger entry and adjusts the seller balance in
// one transaction. The insert is deduplicated on (payment_id, entry_type) via
// the unique index added in migration 0003, so a Kafka redelivery of the same
// event is a safe no-op instead of double-crediting/debiting the seller.
func (r *LedgerRepo) ApplyLedgerEntry(ctx context.Context, e domain.LedgerEntry, balanceDeltaPaise int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO seller_ledger (id, seller_id, order_id, payment_id, entry_type, amount_paise, commission_paise, tds_paise, gst_on_commission_paise, net_paise, currency, note, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (payment_id, entry_type) WHERE payment_id IS NOT NULL DO NOTHING`,
		e.ID, e.SellerID, nullableStr(e.OrderID), nullableStr(e.PaymentID),
		string(e.EntryType), e.AmountPaise, e.CommissionPaise, e.TDSPaise, e.GSTOnCommissionPaise, e.NetPaise,
		e.Currency, e.Note, e.CreatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Duplicate event already applied; nothing more to do.
		return tx.Commit()
	}

	if balanceDeltaPaise >= 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO seller_balances (seller_id, pending_paise, last_updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (seller_id) DO UPDATE
			SET pending_paise = seller_balances.pending_paise + $2,
			    last_updated_at = NOW()`, e.SellerID, balanceDeltaPaise); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE seller_balances
			SET pending_paise = pending_paise + $2,
			    last_updated_at = NOW()
			WHERE seller_id = $1`, e.SellerID, balanceDeltaPaise); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *LedgerRepo) GetBalance(ctx context.Context, sellerID string) (*domain.SellerBalance, error) {
	var b domain.SellerBalance
	err := r.db.QueryRowContext(ctx, `
		SELECT seller_id, pending_paise, paid_out_paise, currency, last_updated_at
		FROM seller_balances WHERE seller_id = $1`, sellerID).
		Scan(&b.SellerID, &b.PendingPaise, &b.PaidOutPaise, &b.Currency, &b.LastUpdatedAt)
	if err == sql.ErrNoRows {
		return &domain.SellerBalance{SellerID: sellerID, Currency: "INR", LastUpdatedAt: time.Now()}, nil
	}
	return &b, err
}

// CreatePendingPayout records an in-flight payout. The unique partial index
// idx_seller_payouts_seller_pending (migration 0003) ensures this fails with
// a unique-violation if the seller already has a PENDING payout, so
// concurrent scheduler runs cannot double-initiate.
func (r *LedgerRepo) CreatePendingPayout(ctx context.Context, sellerID string, amountPaise int64, currency string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO seller_payouts (id, seller_id, amount_paise, currency, status, initiated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, 'PENDING', NOW())
		RETURNING id`, sellerID, amountPaise, currency).Scan(&id)
	return id, err
}

// CompletePayout marks a payout COMPLETED and debits the seller balance
// atomically in one transaction.
func (r *LedgerRepo) CompletePayout(ctx context.Context, payoutID, gatewayPayoutID string, sellerID string, amountPaise int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE seller_payouts SET status = 'COMPLETED', razorpay_payout_id = $2, processed_at = NOW()
		WHERE id = $1`, payoutID, gatewayPayoutID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE seller_balances
		SET pending_paise = pending_paise - $2, paid_out_paise = paid_out_paise + $2, last_updated_at = NOW()
		WHERE seller_id = $1`, sellerID, amountPaise); err != nil {
		return err
	}
	return tx.Commit()
}

// FailPayout marks a payout FAILED so it no longer blocks future attempts
// via the PENDING unique index.
func (r *LedgerRepo) FailPayout(ctx context.Context, payoutID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE seller_payouts SET status = 'FAILED', processed_at = NOW() WHERE id = $1`, payoutID)
	return err
}

func (r *LedgerRepo) GetPendingSellers(ctx context.Context, minBalancePaise int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT seller_id FROM seller_balances WHERE pending_paise >= $1`, minBalancePaise)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return id
}
