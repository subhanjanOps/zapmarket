package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// RefreshTokenRepository handles refresh token database operations
type RefreshTokenRepository struct {
	db *sql.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// hashToken generates a SHA256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// CreateRefreshToken creates a new refresh token in the database
func (r *RefreshTokenRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*domain.RefreshToken, error) {
	tokenID := uuid.New()
	tokenHash := hashToken(token)

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		tokenID,
		userID,
		tokenHash,
		expiresAt,
		now,
		now,
	)

	if err != nil {
		return nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to create refresh token: %v", err))
	}

	return &domain.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at, updated_at, deleted_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND deleted_at IS NULL
	`

	token := &domain.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
		&token.UpdatedAt,
		&token.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewDomainError(domain.ErrInvalidToken, "refresh token not found")
		}
		return nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get refresh token: %v", err))
	}

	// Check if token has been revoked
	if token.RevokedAt != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidToken, "refresh token has been revoked")
	}

	// Check if token has expired
	if time.Now().After(token.ExpiresAt) {
		return nil, domain.NewDomainError(domain.ErrExpiredToken, "refresh token has expired")
	}

	return token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *RefreshTokenRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), tokenID)
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to revoke refresh token: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rowsAffected == 0 {
		return domain.NewDomainError(domain.ErrInvalidToken, "refresh token not found")
	}

	return nil
}

// InvalidateUserTokens revokes all refresh tokens for a user
func (r *RefreshTokenRepository) InvalidateUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1, updated_at = $2
		WHERE user_id = $3 AND revoked_at IS NULL AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), userID)
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to invalidate user tokens: %v", err))
	}

	return nil
}

// GetRefreshTokenByTokenString hashes the token and retrieves it
func (r *RefreshTokenRepository) GetRefreshTokenByTokenString(ctx context.Context, tokenString string) (*domain.RefreshToken, error) {
	tokenHash := hashToken(tokenString)
	return r.GetRefreshTokenByHash(ctx, tokenHash)
}
