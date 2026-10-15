package projects_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

func savedUser(t *testing.T, database db.Database, name string) *repositories.User {
	t.Helper()

	user, err := repositories.NewUser(name, "")
	require.NoError(t, err)

	unit := database.NewContext()
	unit.Users().Insert(user)
	require.NoError(t, unit.SaveChanges(t.Context()))

	return user
}

func TestACreatedProjectIsListedForItsOwner(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")

	// act
	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "Garden", created.Name())

	listed, err := projects.ListFor(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID(), listed[0].ID())
}

func TestAProjectIsNotListedForOthers(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")

	_, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden")
	require.NoError(t, err)

	// act
	listed, err := projects.ListFor(t.Context(), database.NewContext(), ana)

	// assert
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestANameFroglyCantUseCreatesNothing(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")

	// act
	_, err := projects.Create(t.Context(), database.NewContext(), karo, "  ")

	// assert
	invalid, ok := errors.AsType[*repositories.InvalidError](err)
	require.True(t, ok, "want an InvalidError, got %v", err)
	assert.Equal(t, "a project needs a name", invalid.Reason())

	listed, err := projects.ListFor(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestAProjectNeedsASavedOwner(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	stranger, err := repositories.NewUser("Stranger", "")
	require.NoError(t, err)

	// act
	_, err = projects.Create(t.Context(), database.NewContext(), stranger, "Garden")

	// assert
	require.Error(t, err)

	// a missing owner is frogly's mistake, not something the client could fix
	_, invalid := errors.AsType[*repositories.InvalidError](err)
	assert.False(t, invalid, "want a database error, got %v", err)
}
