package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/ganot/threds-mcp/migrations"
	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection
type DB struct {
	*sql.DB
}

// New creates a new SQLite database connection
func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &DB{db}, nil
}

// RunMigrations runs the migrations directly (for testing)
// In production, migrations should be run via the migrate CLI or embed package
func (db *DB) RunMigrations() error {
	migration, err := migrations.FS.ReadFile("001_initial_schema.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	_, err = db.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
