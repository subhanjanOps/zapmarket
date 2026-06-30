package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type BankAccountRepo struct{ db *sql.DB }

func NewBankAccountRepo(db *sql.DB) *BankAccountRepo { return &BankAccountRepo{db: db} }

func (r *BankAccountRepo) Create(ctx context.Context, a *domain.SellerBankAccount) error {
	a.ID = uuid.NewString()
	a.CreatedAt = time.Now()
	var upiID interface{}
	if a.UPIID != "" {
		upiID = a.UPIID
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO seller_bank_accounts (id, seller_id, account_holder_name, account_number, ifsc_code, bank_name, upi_id, is_verified, is_primary, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		a.ID, a.SellerID, a.AccountHolderName, a.AccountNumber, a.IFSCCode, a.BankName, upiID, a.IsVerified, a.IsPrimary, a.CreatedAt,
	)
	return err
}

func (r *BankAccountRepo) ListBySeller(ctx context.Context, sellerID string) ([]*domain.SellerBankAccount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, seller_id, account_holder_name, account_number, ifsc_code, bank_name, COALESCE(upi_id,''), is_verified, is_primary, created_at
		 FROM seller_bank_accounts WHERE seller_id = $1 ORDER BY created_at DESC`, sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []*domain.SellerBankAccount
	for rows.Next() {
		a := &domain.SellerBankAccount{}
		if err := rows.Scan(&a.ID, &a.SellerID, &a.AccountHolderName, &a.AccountNumber, &a.IFSCCode, &a.BankName, &a.UPIID, &a.IsVerified, &a.IsPrimary, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}
