// Package migrator runs the project's embedded SQL migrations
// (elproof/migrations) against the configured database from inside the
// compiled binary — the counterpart of `npm run migrate:up`/`migrate:down`
// (which drive the same *.sql files via the external golang-migrate CLI for
// local dev) for environments where only the deployed image exists, no CLI
// or checked-out source tree. See docs/DEPLOYMENT.md.
package migrator

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"elproof/migrations"
)

// newMigrate opens its own dedicated connection from databaseURL (the
// project's "mysql://..." DATABASE_URL) rather than reusing the app's shared
// *sql.DB pool. Several of this project's migration files contain multiple
// statements per file — the mysql driver only allows that over a connection
// opened with multiStatements=true, which golang-migrate's own Open(url)
// sets automatically; wrapping an already-open *sql.DB via WithInstance does
// not, and fails with a syntax error on the second statement.
func newMigrate(databaseURL string) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrator: load embedded source: %w", err)
	}
	return migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
}

// Up applies all pending migrations. A no-op (nil error) if already current.
func Up(databaseURL string) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrator: up: %w", err)
	}
	return nil
}

// Down rolls back exactly one migration step, mirroring `npm run migrate:down`.
func Down(databaseURL string) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrator: down: %w", err)
	}
	return nil
}

// Force sets schema_migrations to the given version with dirty cleared,
// without running any migration SQL — mirroring the upstream golang-migrate
// CLI's `migrate force <version>`. This is the standard recovery tool for a
// migration left "dirty" by a prior failed run (e.g. an interrupted
// container, a driver bug since fixed): once the actual schema state at
// that version has been verified by hand, Force just corrects the tracking
// table so Up/Down can proceed again.
func Force(databaseURL string, version int) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	if err := m.Force(version); err != nil {
		return fmt.Errorf("migrator: force: %w", err)
	}
	return nil
}
