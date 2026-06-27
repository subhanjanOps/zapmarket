package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

type OTPRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

func (r *OTPRepository) CreateOTP(ctx context.Context, otp *domain.OTPVerification) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO otp_verifications (id, user_id, code_hash, purpose, recipient, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		otp.ID, otp.UserID, otp.CodeHash, string(otp.Purpose), otp.Recipient, otp.ExpiresAt, otp.CreatedAt,
	)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to create OTP: %v", err), err)
	}
	return nil
}

func (r *OTPRepository) GetLatestUnusedOTP(ctx context.Context, userID uuid.UUID, purpose domain.OTPPurpose) (*domain.OTPVerification, error) {
	otp := &domain.OTPVerification{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, code_hash, purpose, recipient, expires_at, used_at, created_at
		 FROM otp_verifications
		 WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID, string(purpose),
	).Scan(&otp.ID, &otp.UserID, &otp.CodeHash, &otp.Purpose, &otp.Recipient, &otp.ExpiresAt, &otp.UsedAt, &otp.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("OTP_NOT_FOUND", "no active OTP found")
		}
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get OTP: %v", err), err)
	}
	return otp, nil
}

func (r *OTPRepository) MarkOTPUsed(ctx context.Context, otpID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE otp_verifications SET used_at = $1 WHERE id = $2 AND used_at IS NULL`,
		time.Now(), otpID,
	)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to mark OTP used: %v", err), err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return pkgerrors.NewNotFound("OTP_NOT_FOUND", "OTP not found or already used")
	}
	return nil
}
