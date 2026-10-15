// Package sqlite is frogly's database in a single SQLite file.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"

	migrate "github.com/rubenv/sql-migrate"
	_ "modernc.org/sqlite" // registers the sqlite driver, pure Go, so frogly builds without cgo

	"github.com/the127/frogly/internal/db"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Database is a SQLite file.
type Database struct {
	connection *sql.DB
}

// Open opens the database at path, and creates the file if it doesn't
// exist.
func Open(path string) (*Database, error) {
	options := url.Values{}
	options.Add("_pragma", "foreign_keys(1)")
	options.Add("_pragma", "journal_mode(WAL)")
	options.Add("_pragma", "busy_timeout(5000)")
	// a transaction takes the write lock at its start, so two that read
	// first and write later can't deadlock each other
	options.Set("_txlock", "immediate")

	connection, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: path, RawQuery: options.Encode(), OmitHost: true}).String())
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	// SQLite writes one transaction at a time anyway, and a single
	// connection keeps it from answering a second writer with "database is
	// locked"
	connection.SetMaxOpenConns(1)

	err = connection.Ping()
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	return &Database{connection: connection}, nil
}

// Migrate brings the schema up to date.
func (database *Database) Migrate(ctx context.Context) error {
	source := migrate.EmbedFileSystemMigrationSource{FileSystem: migrations, Root: "migrations"}

	_, err := migrate.ExecContext(ctx, database.connection, "sqlite3", source, migrate.Up)
	if err != nil {
		return fmt.Errorf("migrating: %w", err)
	}

	return nil
}

// NewContext starts a unit of work.
func (database *Database) NewContext() db.Context {
	return newContext(database.connection)
}

// Close closes the database.
func (database *Database) Close() error {
	return database.connection.Close()
}
