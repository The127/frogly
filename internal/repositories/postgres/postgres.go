// Package postgres holds the repositories for Postgres. Reads go to the
// database right away. Writes are recorded in the unit of work's tracker,
// and its SaveChanges runs the Execute methods inside one transaction.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/the127/frogly/internal/repositories"
)

// Reader is what the repositories read through, the database outside of
// any transaction.
type Reader interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type scanner interface {
	Scan(dest ...any) error
}

// Postgres answers a stale update with the version trigger's serialization
// failure, and a taken unique value with a unique violation. Both mean
// someone else got there first.
func written(err error, what string) error {
	if err == nil {
		return nil
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case pgerrcode.SerializationFailure, pgerrcode.UniqueViolation:
			return fmt.Errorf("%w: %s: %w", repositories.ErrConcurrentModification, what, err)
		}
	}

	return fmt.Errorf("%s: %w", what, err)
}

func requireRow(result sql.Result, what string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("counting the rows written for %s: %w", what, err)
	}

	if rows == 0 {
		return fmt.Errorf("%s: %w", what, repositories.ErrNotFound)
	}

	return nil
}
