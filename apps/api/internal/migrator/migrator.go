// Package migrator runs the project's embedded SQL migrations
// (elproof/migrations) against the configured database from inside the
// compiled binary — the counterpart of `npm run migrate:up`/`migrate:down`
// (which drive the same *.sql files via the external golang-migrate CLI for
// local dev) for environments where only the deployed image exists, no CLI
// or checked-out source tree. See docs/DEPLOYMENT.md.
package migrator

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"elproof/migrations"
)

func newMigrate(db *sql.DB) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrator: load embedded source: %w", err)
	}
	dbDriver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("migrator: init mysql driver: %w", err)
	}
	return migrate.NewWithInstance("iofs", sourceDriver, "mysql", dbDriver)
}

// Up applies all pending migrations. A no-op (nil error) if already current.
func Up(db *sql.DB) error {
	m, err := newMigrate(db)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrator: up: %w", err)
	}
	return nil
}

// Down rolls back exactly one migration step, mirroring `npm run migrate:down`.
func Down(db *sql.DB) error {
	m, err := newMigrate(db)
	if err != nil {
		return err
	}
	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrator: down: %w", err)
	}
	return nil
}
