package projects_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

// askedProject is a project that takes requests, which Ana asked to join.
func askedProject(t *testing.T, database db.Database) (karo, ana *repositories.User, fair *repositories.Project) {
	t.Helper()

	karo, ana = savedUser(t, database, "Karo"), savedUser(t, database, "Ana")
	fair = createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningRequest)
	asked, err := projects.AskToJoin(t.Context(), database.NewContext(), ana, fair.ID())
	require.NoError(t, err)
	require.True(t, asked)

	return karo, ana, fair
}

func TestSomeoneAsksToJoinAProjectThatTakesRequests(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)

	// act
	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	listed, listErr := projects.ListPublicFor(t.Context(), database.NewContext(), ana)

	// assert
	require.NoError(t, err)
	require.Len(t, waiting, 1)
	assert.Equal(t, ana.ID(), waiting[0].Asker.ID())

	require.NoError(t, listErr)
	require.Len(t, listed, 1)
	assert.True(t, listed[0].Asked)
}

func TestAskingAgainOrAsAMemberMakesNoNewRequest(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)

	for _, test := range []struct {
		name  string
		who   *repositories.User
		asked bool
	}{
		{name: "again", who: ana, asked: true},
		{name: "as a member", who: karo, asked: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			// act
			asked, err := projects.AskToJoin(t.Context(), database.NewContext(), test.who, fair.ID())

			// assert
			require.NoError(t, err)
			assert.Equal(t, test.asked, asked, "waits for an answer")

			waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
			require.NoError(t, err)
			assert.Len(t, waiting, 1, "still only Ana's request")
		})
	}
}

func TestAskingAsSomeoneGetsInMakesNoRequest(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana := savedUser(t, database, "Karo"), savedUser(t, database, "Ana")
	fair := createdProject(t, database, karo, repositories.VisibilityPublic, repositories.JoiningRequest)
	unit := racedUnit{Context: database.NewContext(), database: database, projectID: fair.ID(), userID: ana.ID()}

	// act
	asked, err := projects.AskToJoin(t.Context(), unit, ana, fair.ID())

	// assert
	require.NoError(t, err)
	assert.False(t, asked, "a member has nothing to wait for")

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}

func TestAProjectThatDoesntTakeRequestsCantBeAsked(t *testing.T) {
	for _, joining := range []repositories.Joining{repositories.JoiningOpen, repositories.JoiningClosed} {
		t.Run(string(joining), func(t *testing.T) {
			// arrange
			database := sqlitetest.Open(t)
			karo := savedUser(t, database, "Karo")
			ana := savedUser(t, database, "Ana")
			project := createdProject(t, database, karo, repositories.VisibilityPublic, joining)

			// act
			_, err := projects.AskToJoin(t.Context(), database.NewContext(), ana, project.ID())

			// assert
			require.ErrorIs(t, err, projects.ErrNotFound)
		})
	}
}

func TestAYesMakesTheAskerAMember(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)

	// act
	accepted, err := projects.AcceptRequest(t.Context(), database.NewContext(), karo, fair.ID(), ana.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, ana.ID(), accepted.ID())

	_, err = projects.Get(t.Context(), database.NewContext(), ana, fair.ID())
	require.NoError(t, err)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}

func TestANoRemovesTheRequestAndKeepsTheAskerOut(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)

	// act
	err := projects.DeclineRequest(t.Context(), database.NewContext(), karo, fair.ID(), ana.ID())

	// assert
	require.NoError(t, err)

	_, err = projects.Get(t.Context(), database.NewContext(), ana, fair.ID())
	require.ErrorIs(t, err, projects.ErrNotFound)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}

func TestARequestNobodyMadeIsNotFound(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, _, fair := askedProject(t, database)
	tomas := savedUser(t, database, "Tomás")

	// act
	_, accepted := projects.AcceptRequest(t.Context(), database.NewContext(), karo, fair.ID(), tomas.ID())
	declined := projects.DeclineRequest(t.Context(), database.NewContext(), karo, fair.ID(), tomas.ID())

	// assert
	require.ErrorIs(t, accepted, projects.ErrNoRequest)
	require.ErrorIs(t, declined, projects.ErrNoRequest)
}

func TestSomeoneOutsideCantSeeOrAnswerRequests(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, ana, fair := askedProject(t, database)
	tomas := savedUser(t, database, "Tomás")

	// act
	_, listed := projects.Requests(t.Context(), database.NewContext(), tomas, fair.ID())
	_, accepted := projects.AcceptRequest(t.Context(), database.NewContext(), tomas, fair.ID(), ana.ID())
	declined := projects.DeclineRequest(t.Context(), database.NewContext(), ana, fair.ID(), ana.ID())

	// assert
	require.ErrorIs(t, listed, projects.ErrNotFound)
	require.ErrorIs(t, accepted, projects.ErrNotFound)
	require.ErrorIs(t, declined, projects.ErrNotFound, "the asker can't answer their own request")
}

func TestAfterANoTheAskerMayAskAgain(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)
	require.NoError(t, projects.DeclineRequest(t.Context(), database.NewContext(), karo, fair.ID(), ana.ID()))

	listed, err := projects.ListPublicFor(t.Context(), database.NewContext(), ana)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.False(t, listed[0].Asked, "nothing tells the asker it was a no")

	// act
	_, err = projects.AskToJoin(t.Context(), database.NewContext(), ana, fair.ID())

	// assert
	require.NoError(t, err)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Len(t, waiting, 1)
}

func TestSomeoneElsesRequestDoesntCountAsYours(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, _, _ = askedProject(t, database)
	tomas := savedUser(t, database, "Tomás")

	// act
	listed, err := projects.ListPublicFor(t.Context(), database.NewContext(), tomas)

	// assert
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.False(t, listed[0].Asked)
}

func TestAYesForSomeoneWhoIsAMemberByNowFindsNoRequest(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)
	_, err := projects.AddMember(t.Context(), database.NewContext(), karo, fair.ID(), ana.ID())
	require.NoError(t, err)

	// act
	_, err = projects.AcceptRequest(t.Context(), database.NewContext(), karo, fair.ID(), ana.ID())

	// assert
	require.ErrorIs(t, err, projects.ErrNoRequest)
}

func TestAddingSomeoneWhoAskedRemovesTheirRequest(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)

	// act
	_, err := projects.AddMember(t.Context(), database.NewContext(), karo, fair.ID(), ana.ID())

	// assert
	require.NoError(t, err)

	_, err = projects.Get(t.Context(), database.NewContext(), ana, fair.ID())
	require.NoError(t, err)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}

func TestJoiningAfterAskingRemovesTheRequest(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := openedAskedProject(t, database)

	// act
	_, err := projects.Join(t.Context(), database.NewContext(), ana, fair.ID())

	// assert
	require.NoError(t, err)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}

func TestARefusedJoinKeepsTheRequest(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := askedProject(t, database)

	// act
	_, err := projects.Join(t.Context(), database.NewContext(), ana, fair.ID())

	// assert
	require.ErrorIs(t, err, projects.ErrNotFound)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Len(t, waiting, 1)
}

// decliningUnit says no to the asker's request through another unit of
// work right before each of its saves, like a member answering while the
// asker joins.
type decliningUnit struct {
	db.Context

	database  db.Database
	member    *repositories.User
	projectID uuid.UUID
	askerID   uuid.UUID
}

func (unit decliningUnit) SaveChanges(ctx context.Context) error {
	err := projects.DeclineRequest(ctx, unit.database.NewContext(), unit.member, unit.projectID, unit.askerID)
	if err != nil {
		return err
	}

	return unit.Context.SaveChanges(ctx)
}

// acceptingUnit is decliningUnit with a yes.
type acceptingUnit struct {
	db.Context

	database  db.Database
	member    *repositories.User
	projectID uuid.UUID
	askerID   uuid.UUID
}

func (unit acceptingUnit) SaveChanges(ctx context.Context) error {
	_, err := projects.AcceptRequest(ctx, unit.database.NewContext(), unit.member, unit.projectID, unit.askerID)
	if err != nil {
		return err
	}

	return unit.Context.SaveChanges(ctx)
}

// openedAskedProject is askedProject, opened to join since Ana asked.
func openedAskedProject(t *testing.T, database db.Database) (karo, ana *repositories.User, fair *repositories.Project) {
	t.Helper()

	karo, ana, fair = askedProject(t, database)
	_, err := projects.ChangeJoining(t.Context(), database.NewContext(), karo, fair.ID(), repositories.JoiningOpen)
	require.NoError(t, err)

	return karo, ana, fair
}

func TestAJoinWhileAMemberSaysNoToTheRequestIsAConflict(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := openedAskedProject(t, database)
	unit := decliningUnit{Context: database.NewContext(), database: database, member: karo, projectID: fair.ID(), askerID: ana.ID()}

	// act
	_, err := projects.Join(t.Context(), unit, ana, fair.ID())

	// assert
	require.ErrorIs(t, err, repositories.ErrConcurrentModification)

	listed, err := projects.ListFor(t.Context(), database.NewContext(), ana)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestAJoinAfterAConflictWithANoGetsIn(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := openedAskedProject(t, database)
	unit := decliningUnit{Context: database.NewContext(), database: database, member: karo, projectID: fair.ID(), askerID: ana.ID()}
	_, err := projects.Join(t.Context(), unit, ana, fair.ID())
	require.ErrorIs(t, err, repositories.ErrConcurrentModification, "the conflict to try again after")

	// act
	joined, err := projects.Join(t.Context(), database.NewContext(), ana, fair.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, fair.ID(), joined.ID())
}

func TestAJoinWhileAMemberSaysYesToTheRequestGetsIn(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, ana, fair := openedAskedProject(t, database)
	unit := acceptingUnit{Context: database.NewContext(), database: database, member: karo, projectID: fair.ID(), askerID: ana.ID()}

	// act
	joined, err := projects.Join(t.Context(), unit, ana, fair.ID())

	// assert
	require.NoError(t, err)
	assert.Equal(t, fair.ID(), joined.ID())

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}
