package projects_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
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

func TestACreatedProjectIsListedForItsCreator(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")

	// act
	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)

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

	_, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
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
	_, err := projects.Create(t.Context(), database.NewContext(), karo, "  ", repositories.VisibilityPrivate, repositories.JoiningClosed)

	// assert
	invalid, ok := errors.AsType[*repositories.InvalidError](err)
	require.True(t, ok, "want an InvalidError, got %v", err)
	assert.Equal(t, "a project needs a name", invalid.Reason())

	listed, err := projects.ListFor(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestAProjectNeedsASavedCreator(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	stranger, err := repositories.NewUser("Stranger", "")
	require.NoError(t, err)

	// act
	_, err = projects.Create(t.Context(), database.NewContext(), stranger, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)

	// assert
	require.Error(t, err)

	// a missing creator is frogly's mistake, not something the client could fix
	_, invalid := errors.AsType[*repositories.InvalidError](err)
	assert.False(t, invalid, "want a database error, got %v", err)
}

func TestAMemberGetsTheProject(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")

	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	project, err := projects.Get(t.Context(), database.NewContext(), karo, created.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, created.ID(), project.ID())
	assert.Equal(t, "Garden", project.Name())
}

func TestAProjectYouCantSeeIsNotFound(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")

	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID uuid.UUID
	}{
		{name: "not a member", projectID: created.ID()},
		{name: "no such project", projectID: uuid.Must(uuid.NewV7())},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := projects.Get(t.Context(), database.NewContext(), ana, test.projectID)

			// assert
			require.ErrorIs(t, err, projects.ErrNotFound)
		})
	}
}

func TestACreatedProjectHasTheChosenVisibility(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")

	// act
	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPublic, repositories.JoiningClosed)

	// assert
	require.NoError(t, err)

	project, err := projects.Get(t.Context(), database.NewContext(), karo, created.ID())
	require.NoError(t, err)
	assert.Equal(t, repositories.VisibilityPublic, project.Visibility())
}

func TestAVisibilityFroglyDoesntKnowCreatesNothing(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")

	// act
	_, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", "secret", repositories.JoiningClosed)

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))

	listed, err := projects.ListFor(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestAMemberChangesTheVisibility(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	changed, err := projects.ChangeVisibility(t.Context(), database.NewContext(), karo, created.ID(), repositories.VisibilityPublic)

	// assert
	require.NoError(t, err)
	assert.Equal(t, repositories.VisibilityPublic, changed.Visibility())

	project, err := projects.Get(t.Context(), database.NewContext(), karo, created.ID())
	require.NoError(t, err)
	assert.Equal(t, repositories.VisibilityPublic, project.Visibility())
}

func TestOnlyAMemberChangesTheVisibility(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPublic, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.ChangeVisibility(t.Context(), database.NewContext(), ana, created.ID(), repositories.VisibilityPrivate)

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)

	project, err := projects.Get(t.Context(), database.NewContext(), karo, created.ID())
	require.NoError(t, err)
	assert.Equal(t, repositories.VisibilityPublic, project.Visibility())
}

func TestAVisibilityFroglyDoesntKnowChangesNothing(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	created, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.ChangeVisibility(t.Context(), database.NewContext(), karo, created.ID(), "secret")

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))

	project, err := projects.Get(t.Context(), database.NewContext(), karo, created.ID())
	require.NoError(t, err)
	assert.Equal(t, repositories.VisibilityPrivate, project.Visibility())
}

func TestOthersPublicProjectsAreListed(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	fair, err := projects.Create(t.Context(), database.NewContext(), ana, "Summer fair", repositories.VisibilityPublic, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	listed, err := projects.ListPublicFor(t.Context(), database.NewContext(), karo)

	// assert
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, fair.ID(), listed[0].ID())
}

func TestAMemberFindsPeopleToAdd(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana Costa")
	savedUser(t, database, "Tomás")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	found, err := projects.FindPeople(t.Context(), database.NewContext(), karo, garden.ID(), "  costa ")

	// assert
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, ana.ID(), found[0].ID())
}

func TestASearchGivesAtMostTenPeople(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	for range 11 {
		savedUser(t, database, "Ana")
	}

	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	found, err := projects.FindPeople(t.Context(), database.NewContext(), karo, garden.ID(), "ana")

	// assert
	require.NoError(t, err)
	assert.Len(t, found, 10)
}

func TestAnEmptySearchFindsNobody(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.FindPeople(t.Context(), database.NewContext(), karo, garden.ID(), "   ")

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))
}

func TestASearchLongerThanAnyEmailIsRefused(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.FindPeople(t.Context(), database.NewContext(), karo, garden.ID(), strings.Repeat("a", 100_000))

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))
}

func TestSomeoneOutsideAProjectFindsNobodyForIt(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPublic, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.FindPeople(t.Context(), database.NewContext(), ana, garden.ID(), "karo")

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)
}

func TestAMemberAddsSomeone(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	added, err := projects.AddMember(t.Context(), database.NewContext(), karo, garden.ID(), ana.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, ana.ID(), added.ID())

	project, err := projects.Get(t.Context(), database.NewContext(), ana, garden.ID())
	require.NoError(t, err)
	assert.Equal(t, garden.ID(), project.ID())
}

func TestAddingAMemberAgainChangesNothing(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	_, err = projects.AddMember(t.Context(), database.NewContext(), karo, garden.ID(), ana.ID())
	require.NoError(t, err)

	for _, who := range []*repositories.User{ana, karo} {
		// act
		_, err = projects.AddMember(t.Context(), database.NewContext(), karo, garden.ID(), who.ID())

		// assert
		require.NoError(t, err, "adding %s", who.Name())

		listed, err := projects.ListFor(t.Context(), database.NewContext(), who)
		require.NoError(t, err)
		assert.Len(t, listed, 1, "%s is a member once", who.Name())
	}
}

func TestSomeoneOutsideAProjectAddsNobody(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPublic, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.AddMember(t.Context(), database.NewContext(), ana, garden.ID(), ana.ID())

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)

	_, err = projects.Get(t.Context(), database.NewContext(), ana, garden.ID())
	require.ErrorIs(t, err, projects.ErrNotFound)
}

func TestSomeoneOutsideAProjectLearnsNothingFromWhatTheySend(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPublic, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, searched := projects.FindPeople(t.Context(), database.NewContext(), ana, garden.ID(), "   ")
	_, added := projects.AddMember(t.Context(), database.NewContext(), ana, garden.ID(), uuid.Must(uuid.NewV7()))

	// assert
	require.ErrorIs(t, searched, projects.ErrNotFound, "an empty search")
	require.ErrorIs(t, added, projects.ErrNotFound, "an unknown ID")
}

func TestNobodyWithTheIDIsAdded(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	garden, err := projects.Create(t.Context(), database.NewContext(), karo, "Garden", repositories.VisibilityPrivate, repositories.JoiningClosed)
	require.NoError(t, err)

	// act
	_, err = projects.AddMember(t.Context(), database.NewContext(), karo, garden.ID(), uuid.Must(uuid.NewV7()))

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))
}

func createdProject(t *testing.T, database db.Database, creator *repositories.User, visibility repositories.Visibility, joining repositories.Joining) *repositories.Project {
	t.Helper()

	project, err := projects.Create(t.Context(), database.NewContext(), creator, "Summer fair", visibility, joining)
	require.NoError(t, err)

	return project
}

func TestSomeoneJoinsAnOpenProject(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	fair := createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningOpen)

	// act
	joined, err := projects.Join(t.Context(), database.NewContext(), ana, fair.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, fair.ID(), joined.ID())

	listed, err := projects.ListFor(t.Context(), database.NewContext(), ana)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, fair.ID(), listed[0].ID())
}

func TestJoiningAgainChangesNothing(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	fair := createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningOpen)
	_, err := projects.Join(t.Context(), database.NewContext(), ana, fair.ID())
	require.NoError(t, err)

	for _, who := range []*repositories.User{ana, karo} {
		// act
		_, err := projects.Join(t.Context(), database.NewContext(), who, fair.ID())

		// assert
		require.NoError(t, err, who.Name())

		listed, err := projects.ListFor(t.Context(), database.NewContext(), who)
		require.NoError(t, err)
		assert.Len(t, listed, 1, "%s is a member once", who.Name())
	}
}

func TestAProjectThatIsntOpenCantBeJoined(t *testing.T) {
	tests := []struct {
		name       string
		visibility repositories.Visibility
		joining    repositories.Joining
	}{
		{name: "public and closed", visibility: repositories.VisibilityPublic, joining: repositories.JoiningClosed},
		{name: "private", visibility: repositories.VisibilityPrivate, joining: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			database := sqlitetest.Open(t)
			karo := savedUser(t, database, "Karo")
			ana := savedUser(t, database, "Ana")
			project := createdProject(t, database, karo, test.visibility, test.joining)

			// act
			_, err := projects.Join(t.Context(), database.NewContext(), ana, project.ID())

			// assert
			require.ErrorIs(t, err, projects.ErrNotFound)

			listed, err := projects.ListFor(t.Context(), database.NewContext(), ana)
			require.NoError(t, err)
			assert.Empty(t, listed)
		})
	}
}

func TestAProjectThatDoesntExistCantBeJoined(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	ana := savedUser(t, database, "Ana")

	// act
	_, err := projects.Join(t.Context(), database.NewContext(), ana, uuid.Must(uuid.NewV7()))

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)
}

func TestAMemberOpensAProject(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	board := createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningClosed)

	// act
	changed, err := projects.ChangeJoining(t.Context(), database.NewContext(), karo, board.ID(), repositories.JoiningOpen)

	// assert
	require.NoError(t, err)
	assert.Equal(t, repositories.JoiningOpen, changed.Joining())

	project, err := projects.Get(t.Context(), database.NewContext(), karo, board.ID())
	require.NoError(t, err)
	assert.Equal(t, repositories.JoiningOpen, project.Joining())
}

func TestOnlyAMemberChangesHowPeopleJoin(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	board := createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningClosed)

	// act
	_, err := projects.ChangeJoining(t.Context(), database.NewContext(), ana, board.ID(), repositories.JoiningOpen)

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)

	project, err := projects.Get(t.Context(), database.NewContext(), karo, board.ID())
	require.NoError(t, err)
	assert.Equal(t, repositories.JoiningClosed, project.Joining())
}

func TestAPrivateProjectCantBeOpened(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	secret := createdProject(t, database, karo, repositories.VisibilityPrivate, "")

	// act
	_, err := projects.ChangeJoining(t.Context(), database.NewContext(), karo, secret.ID(), repositories.JoiningOpen)

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))
}

// racedUnit saves the same membership through another unit of work right
// before its own save, like a second tab of the same person.
type racedUnit struct {
	db.Context

	database  db.Database
	projectID uuid.UUID
	userID    uuid.UUID
}

func (unit racedUnit) SaveChanges(ctx context.Context) error {
	membership, err := repositories.NewMembership(unit.projectID, unit.userID)
	if err != nil {
		return err
	}

	other := unit.database.NewContext()
	other.Memberships().Insert(membership)

	err = other.SaveChanges(ctx)
	if err != nil {
		return err
	}

	return unit.Context.SaveChanges(ctx)
}

func TestAJoinThatLostARaceToTheSamePersonStillJoins(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo := savedUser(t, database, "Karo")
	ana := savedUser(t, database, "Ana")
	fair := createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningOpen)
	unit := racedUnit{Context: database.NewContext(), database: database, projectID: fair.ID(), userID: ana.ID()}

	// act
	joined, err := projects.Join(t.Context(), unit, ana, fair.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, fair.ID(), joined.ID())
}
