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

func (r *LedgerRepo) InsertEntry(ctx context.Context, e domain.LedgerEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO seller_ledger (id, seller_id, order_id, payment_id, entry_type, amount_paise, commission_paise, tds_paise, gst_on_commission_paise, net_paise, currency, note, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		e.ID, e.SellerID, nullableStr(e.OrderID), nullableStr(e.PaymentID),
		string(e.EntryType), e.AmountPaise, e.CommissionPaise, e.TDSPaise, e.GSTOnCommissionPaise, e.NetPaise,
		e.Currency, e.Note, e.CreatedAt)
	return err
}

func (r *LedgerRepo) CreditBalance(ctx context.Context, sellerID string, netPaise int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO seller_balances (seller_id, pending_paise, last_updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (seller_id) DO UPDATE
		SET pending_paise = seller_balances.pending_paise + $2,
		    last_updated_at = NOW()`, sellerID, netPaise)
	return err
}

func (r *LedgerRepo) DebitBalance(ctx context.Context, sellerID string, amountPaise int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE seller_balances
		SET pending_paise  = pending_paise - $2,
		    paid_out_paise = paid_out_paise + $2,
		    last_updated_at = NOW()
		WHERE seller_id = $1`, sellerID, amountPaise)
	return err
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
