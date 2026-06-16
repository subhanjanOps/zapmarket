// Package migrate runs golang-migrate up/down migrations against a service's
// Postgres database from a local migrations directory.
package migrate

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/zapmarket/zapmarket/pkg/config"
)

func newMigrator(cfg *config.Config, migrationsDir string) (*migrate.Migrate, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)
	m, err := migrate.New("file://"+migrationsDir, dsn)
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}
	return m, nil
}

// Up applies all pending up migrations from migrationsDir. A no-op (nil
// error) if the schema is already current.
func Up(cfg *config.Config, migrationsDir string) error {
	m, err := newMigrator(cfg, migrationsDir)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down rolls back all migrations from migrationsDir.
func Down(cfg *config.Config, migrationsDir string) error {
	m, err := newMigrator(cfg, migrationsDir)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

