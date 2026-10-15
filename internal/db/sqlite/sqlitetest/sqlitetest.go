// Package sqlitetest gives tests a database of their own.
package sqlitetest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite"
)

// Open creates a migrated SQLite database in the test's temporary
// directory, and closes it after the test.
func Open(t *testing.T) db.Database {
	t.Helper()

	database, err := sqlite.Open(filepath.Join(t.TempDir(), "frogly.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	require.NoError(t, database.Migrate(t.Context()))

	return database
}
