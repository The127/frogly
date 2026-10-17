package sqlite_test

import (
	"testing"

	"github.com/the127/frogly/internal/db/dbtest"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
)

func TestSQLiteMeetsTheContract(t *testing.T) {
	dbtest.Run(t, sqlitetest.Open)
}
