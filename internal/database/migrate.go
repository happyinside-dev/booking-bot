package database

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver, used only for migrations
	"github.com/pressly/goose/v3"
)

// migrationsFS embeds the .sql migration files directly into the binary, so
// the application never depends on a migrations/ directory being present on
// disk (important for the Docker image, where nothing outside the compiled
// binary is guaranteed to exist).
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending goose migrations. It opens its own
// short-lived database/sql connection (goose requires database/sql, not
// pgx's native pool) and closes it before returning.
//
// Trade-off, noted deliberately: this runs on every process start, which is
// fine for a single-instance pet-project deployment. In a multi-replica
// production deployment, migrations would normally run once from a separate
// job/init step instead of from every replica concurrently.
func RunMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("database: open migration connection: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("database: set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("database: run migrations: %w", err)
	}

	return nil
}
