package postgres

import (
	"context"
	"database/sql"
)

// Statement is a change that writes itself, for a write that isn't one
// loaded model, like removing rows by their keys. The unit of work runs
// it as it is.
type Statement interface {
	Execute(ctx context.Context, tx *sql.Tx) error
}
