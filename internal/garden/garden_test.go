package garden_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
	"github.com/the127/frogly/internal/garden"
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

func savedProject(t *testing.T, database db.Database, creator *repositories.User) *repositories.Project {
	t.Helper()

	project, err := projects.Create(t.Context(), database.NewContext(), creator, "Garden", repositories.VisibilityPrivate)
	require.NoError(t, err)

	return project
}

func TestAPlantedIdeaIsInTheGardenWithItsPlanter(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	project := savedProject(t, database, karo)

	// act
	planted, err := garden.Plant(t.Context(), database.NewContext(), karo, project.ID(), "  search by any word ")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "search by any word", planted.Text())
	assert.Equal(t, karo.ID(), planted.PlanterID())

	listed, err := garden.List(t.Context(), database.NewContext(), karo, project.ID())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, planted.ID(), listed[0].Idea.ID())
	assert.Equal(t, "Karo", listed[0].PlantedBy)
}

func TestTheGardenNamesEveryPlanter(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	project := savedProject(t, database, karo)

	membership, err := repositories.NewMembership(project.ID(), ana.ID())
	require.NoError(t, err)

	unit := database.NewContext()
	unit.Memberships().Insert(membership)
	require.NoError(t, unit.SaveChanges(t.Context()))

	for _, planting := range []struct {
		planter *repositories.User
		text    string
	}{{karo, "first"}, {ana, "second"}, {karo, "third"}} {
		_, err = garden.Plant(t.Context(), database.NewContext(), planting.planter, project.ID(), planting.text)
		require.NoError(t, err)
	}

	// act
	listed, err := garden.List(t.Context(), database.NewContext(), karo, project.ID())

	// assert
	require.NoError(t, err)
	require.Len(t, listed, 3)
	// newest first
	assert.Equal(t, "Karo", listed[0].PlantedBy)
	assert.Equal(t, "Ana", listed[1].PlantedBy)
	assert.Equal(t, "Karo", listed[2].PlantedBy)
}

func TestAGardenIsOnlyForMembers(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	project := savedProject(t, database, karo)

	tests := []struct {
		name      string
		projectID uuid.UUID
	}{
		{name: "not a member", projectID: project.ID()},
		{name: "no such project", projectID: uuid.Must(uuid.NewV7())},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, plantErr := garden.Plant(t.Context(), database.NewContext(), ana, test.projectID, "let me in")
			_, listErr := garden.List(t.Context(), database.NewContext(), ana, test.projectID)

			// assert
			require.ErrorIs(t, plantErr, projects.ErrNotFound)
			require.ErrorIs(t, listErr, projects.ErrNotFound)
		})
	}
}

func TestPlantingRefusesATextItCantUse(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	project := savedProject(t, database, karo)

	// act
	_, err := garden.Plant(t.Context(), database.NewContext(), karo, project.ID(), "   ")

	// assert
	invalid, ok := errors.AsType[*repositories.InvalidError](err)
	require.True(t, ok, "want an InvalidError, got %v", err)
	assert.Equal(t, "an idea needs a text", invalid.Reason())

	listed, err := garden.List(t.Context(), database.NewContext(), karo, project.ID())
	require.NoError(t, err)
	assert.Empty(t, listed)
}
