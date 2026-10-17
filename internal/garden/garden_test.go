package garden_test

import (
	"context"
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
	assert.Equal(t, "search by any word", planted.Idea.Text())
	assert.Equal(t, karo.ID(), planted.Idea.PlanterID())
	assert.Equal(t, "Karo", planted.PlantedBy)
	assert.Equal(t, 1, planted.People)

	listed, err := garden.List(t.Context(), database.NewContext(), karo, project.ID())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, planted.Idea.ID(), listed[0].Idea.ID())
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

// sharedIdea is an idea Karo planted in a project Ana is a member of too.
func sharedIdea(t *testing.T, database db.Database) (karo, ana *repositories.User, project *repositories.Project, idea *repositories.Idea) {
	t.Helper()

	karo, ana = savedUser(t, database, "Karo"), savedUser(t, database, "Ana")
	project = savedProject(t, database, karo)

	_, err := projects.AddMember(t.Context(), database.NewContext(), karo, project.ID(), ana.ID())
	require.NoError(t, err)

	planted, err := garden.Plant(t.Context(), database.NewContext(), karo, project.ID(), "search by any word")
	require.NoError(t, err)

	idea = planted.Idea

	return karo, ana, project, idea
}

func TestAMeTooCountsOneMorePerson(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, ana, project, idea := sharedIdea(t, database)

	// act
	said, err := garden.MeToo(t.Context(), database.NewContext(), ana, project.ID(), idea.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, idea.ID(), said.Idea.ID())
	assert.Equal(t, "Karo", said.PlantedBy)
	assert.Equal(t, 2, said.People)
}

func TestAMeTooAgainOrByThePlanterChangesNothing(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, project, idea := sharedIdea(t, database)
	_, err := garden.MeToo(t.Context(), database.NewContext(), ana, project.ID(), idea.ID())
	require.NoError(t, err)

	for _, who := range []*repositories.User{ana, karo} {
		// act
		said, err := garden.MeToo(t.Context(), database.NewContext(), who, project.ID(), idea.ID())

		// assert
		require.NoError(t, err, who.Name())
		assert.Equal(t, 2, said.People, who.Name())
	}

	planterSaid, err := database.NewContext().MeToos().Said(t.Context(), idea.ID(), karo.ID())
	require.NoError(t, err)
	assert.False(t, planterSaid, "nothing is stored for the planter")
}

func TestSomeoneOutsideTheProjectCantSayMeToo(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, _, project, idea := sharedIdea(t, database)
	tomas := savedUser(t, database, "Tomás")

	// act
	_, err := garden.MeToo(t.Context(), database.NewContext(), tomas, project.ID(), idea.ID())

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)
}

func TestAnIdeaThatIsntInTheProjectIsNotFound(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, project, _ := sharedIdea(t, database)
	elsewhere := savedProject(t, database, karo)
	other, err := garden.Plant(t.Context(), database.NewContext(), karo, elsewhere.ID(), "an idea elsewhere")
	require.NoError(t, err)

	for _, ideaID := range []uuid.UUID{other.Idea.ID(), uuid.Must(uuid.NewV7())} {
		// act
		_, err := garden.MeToo(t.Context(), database.NewContext(), ana, project.ID(), ideaID)

		// assert
		require.ErrorIs(t, err, garden.ErrIdeaNotFound)
	}
}

// racedUnit saves the same me too through another unit of work right
// before its own save, like a second tab of the same person.
type racedUnit struct {
	db.Context

	database db.Database
	ideaID   uuid.UUID
	userID   uuid.UUID
}

func (unit racedUnit) SaveChanges(ctx context.Context) error {
	meToo, err := repositories.NewMeToo(unit.ideaID, unit.userID)
	if err != nil {
		return err
	}

	other := unit.database.NewContext()
	other.MeToos().Insert(meToo)

	err = other.SaveChanges(ctx)
	if err != nil {
		return err
	}

	return unit.Context.SaveChanges(ctx)
}

func TestAMeTooThatLostARaceToTheSamePersonCountsOnce(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, ana, project, idea := sharedIdea(t, database)
	unit := racedUnit{Context: database.NewContext(), database: database, ideaID: idea.ID(), userID: ana.ID()}

	// act
	said, err := garden.MeToo(t.Context(), unit, ana, project.ID(), idea.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, 2, said.People)
}
