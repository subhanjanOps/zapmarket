package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
)

// OAuthRepository handles OAuth account database operations
type OAuthRepository struct {
	db *sql.DB
}

// NewOAuthRepository creates a new OAuth repository
func NewOAuthRepository(db *sql.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// CreateOAuthAccount creates a new OAuth account in the database
func (r *OAuthRepository) CreateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error {
	query := `
		INSERT INTO oauth_accounts (id, user_id, provider, provider_uid, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		account.ID,
		account.UserID,
		string(account.Provider),
		account.ProviderUID,
		account.CreatedAt,
		account.UpdatedAt,
	)

	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to create oauth account: %v", err))
	}

	return nil
}

// GetOAuthAccountByProviderUID retrieves an OAuth account by provider and provider UID
// Returns both the OAuth account and the associated user
func (r *OAuthRepository) GetOAuthAccountByProviderUID(ctx context.Context, provider domain.OAuthProvider, providerUID string) (*domain.OAuthAccount, *domain.User, error) {
	query := `
		SELECT oa.id, oa.user_id, oa.provider, oa.provider_uid, oa.created_at, oa.updated_at, oa.deleted_at,
		       u.id, u.email, u.phone, u.password_hash, u.full_name, u.role, u.is_verified, u.created_at, u.updated_at, u.deleted_at
		FROM oauth_accounts oa
		JOIN users u ON oa.user_id = u.id
		WHERE oa.provider = $1 AND oa.provider_uid = $2 AND oa.deleted_at IS NULL AND u.deleted_at IS NULL
	`

	oauthAccount := &domain.OAuthAccount{}
	user := &domain.User{}

	err := r.db.QueryRowContext(ctx, query, string(provider), providerUID).Scan(
		&oauthAccount.ID,
		&oauthAccount.UserID,
		&oauthAccount.Provider,
		&oauthAccount.ProviderUID,
		&oauthAccount.CreatedAt,
		&oauthAccount.UpdatedAt,
		&oauthAccount.DeletedAt,
		&user.ID,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, domain.NewDomainError(domain.ErrUserNotFound, "oauth account not found")
		}
		return nil, nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get oauth account: %v", err))
	}

	return oauthAccount, user, nil
}

// UpdateOAuthAccount updates an OAuth account
func (r *OAuthRepository) UpdateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error {
	query := `
		UPDATE oauth_accounts
		SET provider = $1, provider_uid = $2, updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		string(account.Provider),
		account.ProviderUID,
		time.Now(),
		account.ID,
	)

	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to update oauth account: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rowsAffected == 0 {
		return domain.NewDomainError(domain.ErrUserNotFound, "oauth account not found")
	}

	return nil
}

// DeleteOAuthAccount soft-deletes an OAuth account
func (r *OAuthRepository) DeleteOAuthAccount(ctx context.Context, accountID uuid.UUID) error {
	query := `
		UPDATE oauth_accounts
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), accountID)
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to delete oauth account: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rowsAffected == 0 {
		return domain.NewDomainError(domain.ErrUserNotFound, "oauth account not found")
	}

	return nil
}
