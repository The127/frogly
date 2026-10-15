package sqlite_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/the127/frogly/internal/db/dbtest"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
)

func TestSQLiteMeetsTheContract(t *testing.T) {
	suite.Run(t, &dbtest.Suite{Open: sqlitetest.Open})
}
