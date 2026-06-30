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
		INSERT INTO users (id, email, phone, password_hash, full_name, role, is_verified, seller_status,
		                   phone_verified, registration_step, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
		user.PhoneVerified,
		user.RegistrationStep,
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
	row := r.db.QueryRowContext(ctx, `SELECT `+userSelectCols+` FROM users WHERE email = $1 AND deleted_at IS NULL`, email)
	u, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	return u, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userSelectCols+` FROM users WHERE id = $1 AND deleted_at IS NULL`, userID)
	u, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	return u, nil
}

// GetUserByPhone retrieves a user by their phone number.
func (r *UserRepository) GetUserByPhone(ctx context.Context, phone string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userSelectCols+` FROM users WHERE phone = $1 AND deleted_at IS NULL`, phone)
	u, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	return u, nil
}

// UpdateUser updates a user's information
func (r *UserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, phone = $2, password_hash = $3, full_name = $4, role = $5, is_verified = $6, seller_status = $7, updated_at = $8
		WHERE id = $9 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.IsVerified,
		user.SellerStatus,
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

// UpdateProfile updates profile fields set during registration wizard steps.
func (r *UserRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, dob *time.Time, gender, pfpURL *string, phoneVerified *bool, registrationStep *int) error {
	sets := []string{"updated_at = $1"}
	args := []any{time.Now()}
	i := 2
	if dob != nil {
		sets = append(sets, fmt.Sprintf("dob = $%d", i)); args = append(args, *dob); i++
	}
	if gender != nil {
		sets = append(sets, fmt.Sprintf("gender = $%d", i)); args = append(args, *gender); i++
	}
	if pfpURL != nil {
		sets = append(sets, fmt.Sprintf("pfp_url = $%d", i)); args = append(args, *pfpURL); i++
	}
	if phoneVerified != nil {
		sets = append(sets, fmt.Sprintf("phone_verified = $%d", i)); args = append(args, *phoneVerified); i++
	}
	if registrationStep != nil {
		sets = append(sets, fmt.Sprintf("registration_step = $%d", i)); args = append(args, *registrationStep); i++
	}
	args = append(args, userID)
	q := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL", strings.Join(sets, ", "), i)
	_, err := r.db.ExecContext(ctx, q, args...)
	return err
}

// CompleteRegistration sets terms_accepted_at and registration_step=4 atomically.
func (r *UserRepository) CompleteRegistration(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET terms_accepted_at = $1, registration_step = 4, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		now, userID,
	)
	return err
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

const userSelectCols = `id, email, phone, password_hash, full_name, role, is_verified, seller_status,
	phone_verified, registration_step, dob, gender, pfp_url, terms_accepted_at, created_at, updated_at, deleted_at,
	failed_login_attempts, locked_until, totp_secret, totp_enabled, totp_enrolled_at`

func scanUserRow(row interface {
	Scan(...any) error
}) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FullName, &u.Role, &u.IsVerified, &u.SellerStatus,
		&u.PhoneVerified, &u.RegistrationStep, &u.DOB, &u.Gender, &u.PfpURL, &u.TermsAcceptedAt,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		&u.FailedLoginAttempts, &u.LockedUntil,
		&u.TOTPSecret, &u.TOTPEnabled, &u.TOTPEnrolledAt,
	)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", fmt.Sprintf("failed to scan user: %v", err), err)
	}
	return u, nil
}

// IncrFailedLogin increments failed_login_attempts and locks the account after 5 failures.
func (r *UserRepository) IncrFailedLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET
			failed_login_attempts = failed_login_attempts + 1,
			locked_until = CASE WHEN failed_login_attempts + 1 >= 5 THEN NOW() + INTERVAL '30 minutes' ELSE locked_until END,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, userID)
	return err
}

// ResetLoginAttempts clears failed_login_attempts and locked_until.
func (r *UserRepository) ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, userID)
	return err
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

func (r *UserRepository) SetTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET totp_secret = $2, updated_at = NOW() WHERE id = $1`, userID, secret)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to set totp secret", err)
	}
	return nil
}

func (r *UserRepository) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET totp_enabled = TRUE, totp_enrolled_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to enable totp", err)
	}
	return nil
}

func (r *UserRepository) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET totp_enabled = FALSE, totp_secret = NULL, totp_enrolled_at = NULL, updated_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to disable totp", err)
	}
	return nil
}

func (r *UserRepository) SaveBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_backup_codes WHERE user_id = $1`, userID)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to clear backup codes", err)
	}
	for _, h := range codeHashes {
		_, err := r.db.ExecContext(ctx, `INSERT INTO user_backup_codes (user_id, code_hash) VALUES ($1, $2)`, userID, h)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to save backup code", err)
		}
	}
	return nil
}

func (r *UserRepository) GetUnusedBackupCode(ctx context.Context, userID uuid.UUID, codeHash string) (bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM user_backup_codes WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL LIMIT 1`, userID, codeHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, pkgerrors.NewInternal("DATABASE_ERROR", "failed to look up backup code", err)
	}
	return true, nil
}

func (r *UserRepository) MarkBackupCodeUsed(ctx context.Context, userID uuid.UUID, codeHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_backup_codes SET used_at = NOW() WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL`, userID, codeHash)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to mark backup code used", err)
	}
	return nil
}
