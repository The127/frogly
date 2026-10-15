package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/dbtest"
	"github.com/the127/frogly/internal/db/sqlite"
)

func TestSQLiteMeetsTheContract(t *testing.T) {
	suite.Run(t, &dbtest.Suite{
		Open: func(t *testing.T) db.Database {
			t.Helper()

			database, err := sqlite.Open(filepath.Join(t.TempDir(), "frogly.db"))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, database.Close())
			})

			require.NoError(t, database.Migrate(t.Context()))

			return database
		},
	})
}
