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

// PasswordResetRepository handles password reset token database operations
type PasswordResetRepository struct {
	db *sql.DB
}

// NewPasswordResetRepository creates a new password reset repository
func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// CreatePasswordResetToken inserts a new reset token row
func (r *PasswordResetRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.PasswordResetToken, error) {
	id := uuid.New()
	now := time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, userID, tokenHash, expiresAt, now,
	)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to create password reset token: %v", err), err)
	}

	return &domain.PasswordResetToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// GetPasswordResetTokenByHash retrieves a token by its SHA-256 hash
func (r *PasswordResetRepository) GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	t := &domain.PasswordResetToken{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_reset_tokens
		 WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("TOKEN_NOT_FOUND", "password reset token not found")
		}
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get password reset token: %v", err), err)
	}

	return t, nil
}

// MarkPasswordResetTokenUsed sets used_at to now for the given token ID
func (r *PasswordResetRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = $1 WHERE id = $2 AND used_at IS NULL`,
		time.Now(), tokenID,
	)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to mark reset token used: %v", err), err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to get rows affected", err)
	}
	if n == 0 {
		return pkgerrors.NewNotFound("TOKEN_NOT_FOUND", "password reset token not found or already used")
	}
	return nil
}
