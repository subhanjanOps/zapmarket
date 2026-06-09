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

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, phone, password_hash, full_name, role, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.IsVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			return domain.NewDomainError(domain.ErrUserExists, "user with this email already exists")
		}
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to create user: %v", err))
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, phone, password_hash, full_name, role, is_verified, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
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
			return nil, domain.NewDomainError(domain.ErrUserNotFound, "user not found")
		}
		return nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get user: %v", err))
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, phone, password_hash, full_name, role, is_verified, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
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
			return nil, domain.NewDomainError(domain.ErrUserNotFound, "user not found")
		}
		return nil, domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get user: %v", err))
	}

	return user, nil
}

// UpdateUser updates a user's information
func (r *UserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, phone = $2, password_hash = $3, full_name = $4, role = $5, is_verified = $6, updated_at = $7
		WHERE id = $8 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.IsVerified,
		time.Now(),
		user.ID,
	)

	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to update user: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rowsAffected == 0 {
		return domain.NewDomainError(domain.ErrUserNotFound, "user not found")
	}

	return nil
}

// VerifyUser marks a user as verified
func (r *UserRepository) VerifyUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET is_verified = true, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to verify user: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rowsAffected == 0 {
		return domain.NewDomainError(domain.ErrUserNotFound, "user not found")
	}

	return nil
}

// DeleteUser soft-deletes a user
func (r *UserRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), userID)
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to delete user: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.NewDomainError(domain.ErrDatabaseError, fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rowsAffected == 0 {
		return domain.NewDomainError(domain.ErrUserNotFound, "user not found")
	}

	return nil
}
