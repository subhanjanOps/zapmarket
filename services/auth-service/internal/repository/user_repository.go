package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/auth-service/internal/domain/contracts"
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
		INSERT INTO users (id, email, phone, password_hash, full_name, role, is_verified, seller_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.IsVerified,
		user.SellerStatus,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return pkgerrors.NewConflict("USER_ALREADY_EXISTS", "user with this email already exists")
		}
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to create user: %v", err), err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, phone, password_hash, full_name, role, is_verified, seller_status, created_at, updated_at, deleted_at
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
		&user.SellerStatus,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get user: %v", err), err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, phone, password_hash, full_name, role, is_verified, seller_status, created_at, updated_at, deleted_at
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
		&user.SellerStatus,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get user: %v", err), err)
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
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to update user: %v", err), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get rows affected: %v", err), err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
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
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to verify user: %v", err), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get rows affected: %v", err), err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
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
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to delete user: %v", err), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to get rows affected: %v", err), err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
	}

	return nil
}

// ── Admin methods ─────────────────────────────────────────────────────────────

const userSelectCols = `id, email, phone, password_hash, full_name, role, is_verified, seller_status, created_at, updated_at, deleted_at`

func scanUserRow(row interface {
	Scan(...any) error
}) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FullName, &u.Role, &u.IsVerified, &u.SellerStatus, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to scan user: %v", err), err)
	}
	return u, nil
}

// ListUsers returns a paginated list of users filtered by role and/or search term.
func (r *UserRepository) ListUsers(ctx context.Context, params contracts.UserListParams) ([]*domain.User, int64, error) {
	args := []any{}
	conditions := []string{"deleted_at IS NULL"}
	i := 1

	if params.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", i))
		args = append(args, params.Role)
		i++
	}
	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(email ILIKE $%d OR full_name ILIKE $%d)", i, i+1))
		like := "%" + params.Search + "%"
		args = append(args, like, like)
		i += 2
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to count users", err)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit, params.Offset)
	query := fmt.Sprintf("SELECT %s FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		userSelectCols, where, i, i+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to list users", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// UpdateSellerStatus sets seller_status for a seller user.
func (r *UserRepository) UpdateSellerStatus(ctx context.Context, userID uuid.UUID, status string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE users SET seller_status = $1, updated_at = $2 WHERE id = $3 AND role = 'seller' AND deleted_at IS NULL",
		status, time.Now(), userID)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to update seller status: %v", err), err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return pkgerrors.NewNotFound("USER_NOT_FOUND", "seller not found")
	}
	return nil
}

// ListSellers returns sellers, optionally filtered by seller_status.
func (r *UserRepository) ListSellers(ctx context.Context, status string, limit, offset int) ([]*domain.User, int64, error) {
	args := []any{}
	conditions := []string{"role = 'seller'", "deleted_at IS NULL"}
	i := 1

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("seller_status = $%d", i))
		args = append(args, status)
		i++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to count sellers", err)
	}

	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit, offset)
	query := fmt.Sprintf("SELECT %s FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		userSelectCols, where, i, i+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to list sellers", err)
	}
	defer rows.Close()

	var sellers []*domain.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		sellers = append(sellers, u)
	}
	return sellers, total, rows.Err()
}
