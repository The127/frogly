//go:build integration

package postgres_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // the driver the test talks to Postgres with
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/dbtest"
	"github.com/the127/frogly/internal/db/postgres"
)

// the Postgres `just dev` and `just integration` start, unless set otherwise
const defaultURL = "postgres://frogly:frogly@localhost:5432/frogly?sslmode=disable" //nolint:gosec // the public password of the local dev database

func TestPostgresMeetsTheContract(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) db.Database {
		t.Helper()

		database, err := postgres.Open(freshDatabase(t))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, database.Close())
		})

		require.NoError(t, database.Migrate(t.Context()))

		return database
	})
}

// freshDatabase creates an empty database for one test and drops it again
// after the test, so tests never see each other's rows.
func freshDatabase(t *testing.T) string {
	t.Helper()

	server := os.Getenv("TEST_POSTGRES_URL")
	if server == "" {
		server = defaultURL
	}

	admin, err := sql.Open("pgx", server)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, admin.Close())
	})

	name := "dbtest_" + strings.ToLower(rand.Text())

	_, err = admin.ExecContext(t.Context(), "create database "+name)
	require.NoError(t, err, "is Postgres running? `just integration` starts it")

	t.Cleanup(func() {
		// the test's context is done by the time cleanups run
		_, err := admin.ExecContext(context.Background(), "drop database "+name+" with (force)")
		require.NoError(t, err)
	})

	fresh, err := url.Parse(server)
	require.NoError(t, err)

	fresh.Path = "/" + name

	return fresh.String()
}
