// Package sqlite holds the repositories for SQLite. Reads go to the
// database right away. Writes are recorded in the unit of work's tracker,
// and its SaveChanges runs the Execute methods inside one transaction.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqlitedriver "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

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

// SQLite answers a stale update with the version trigger's error, and a
// taken unique value with a constraint error. Both mean someone else got
// there first.
func written(err error, what string) error {
	if err == nil {
		return nil
	}

	if sqliteErr, ok := errors.AsType[*sqlitedriver.Error](err); ok {
		switch sqliteErr.Code() {
		case sqlite3.SQLITE_CONSTRAINT_TRIGGER, sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return fmt.Errorf("%w: %s: %w", repositories.ErrConcurrentModification, what, err)
		}
	}

	return fmt.Errorf("%s: %w", what, err)
}
