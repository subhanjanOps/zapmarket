package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PreferencesRepository is the concrete implementation backed by PostgreSQL.
type PreferencesRepository struct {
	db *sql.DB
}

// NewPreferencesRepository creates a new PreferencesRepository.
func NewPreferencesRepository(db *sql.DB) *PreferencesRepository {
	return &PreferencesRepository{db: db}
}

// Get retrieves a preference value for a user and key.
// Returns ("", false, nil) when no row exists.
func (r *PreferencesRepository) Get(ctx context.Context, userID uuid.UUID, key string) (string, bool, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM user_preferences WHERE user_id = $1 AND key = $2`,
		userID, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get preference: %w", err)
	}
	return value, true, nil
}

// Set upserts a preference value for a user and key.
func (r *PreferencesRepository) Set(ctx context.Context, userID uuid.UUID, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_preferences (user_id, key, value, updated_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, key) DO UPDATE
		   SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		userID, key, value, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("set preference: %w", err)
	}
	return nil
}
