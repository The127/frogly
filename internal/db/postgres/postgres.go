// Package postgres is frogly's database in Postgres.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the pgx driver for database/sql
	migrate "github.com/rubenv/sql-migrate"

	"github.com/the127/frogly/internal/db"
)

//go:embed migrations/*.sql
var migrations embed.FS

// leaves room for other clients of the same Postgres, like a migration or
// a person with psql
const maxConnections = 20

// Database is a Postgres database.
type Database struct {
	connection *sql.DB
}

// Open connects to the database at url, like
// postgres://user:password@host:5432/database.
func Open(url string) (*Database, error) {
	connection, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("opening the Postgres database: %w", err)
	}

	// without limits, a burst of requests opens a connection each and could
	// pass Postgres's max_connections, 100 by default. Idle ones stay
	// around, so a busy frogly doesn't connect anew on every request.
	connection.SetMaxOpenConns(maxConnections)
	connection.SetMaxIdleConns(maxConnections)
	connection.SetConnMaxIdleTime(5 * time.Minute)

	err = connection.Ping()
	if err != nil {
		_ = connection.Close() // the connection failed already, that error says more

		// the URL may hold a password, so the error doesn't repeat it
		return nil, fmt.Errorf("connecting to the Postgres database: %w", err)
	}

	return &Database{connection: connection}, nil
}

// Migrate brings the schema up to date.
func (database *Database) Migrate(ctx context.Context) error {
	source := migrate.EmbedFileSystemMigrationSource{FileSystem: migrations, Root: "migrations"}

	_, err := migrate.ExecContext(ctx, database.connection, "postgres", source, migrate.Up)
	if err != nil {
		return fmt.Errorf("migrating: %w", err)
	}

	return nil
}

// NewContext starts a unit of work.
func (database *Database) NewContext() db.Context {
	return newContext(database.connection)
}

// Close closes the connections.
func (database *Database) Close() error {
	return database.connection.Close()
}
