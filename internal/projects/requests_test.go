package projects_test

import (
	"context"
	"testing"

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

func TestAProjectThatStopsTakingRequestsDropsThem(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(ctx context.Context, unit db.Context, karo *repositories.User, fair *repositories.Project) error
	}{
		{name: "closed", change: func(ctx context.Context, unit db.Context, karo *repositories.User, fair *repositories.Project) error {
			_, err := projects.ChangeJoining(ctx, unit, karo, fair.ID(), repositories.JoiningClosed)

			return err
		}},
		{name: "open", change: func(ctx context.Context, unit db.Context, karo *repositories.User, fair *repositories.Project) error {
			_, err := projects.ChangeJoining(ctx, unit, karo, fair.ID(), repositories.JoiningOpen)

			return err
		}},
		{name: "private", change: func(ctx context.Context, unit db.Context, karo *repositories.User, fair *repositories.Project) error {
			_, err := projects.ChangeVisibility(ctx, unit, karo, fair.ID(), repositories.VisibilityPrivate)

			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			database := sqlitetest.Open(t)
			karo, _, fair := askedProject(t, database)

			// act
			err := test.change(t.Context(), database.NewContext(), karo, fair)

			// assert
			require.NoError(t, err)

			waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
			require.NoError(t, err)
			assert.Empty(t, waiting)
		})
	}
}

func TestAProjectThatStillTakesRequestsKeepsThem(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	karo, _, fair := askedProject(t, database)

	// act
	_, err := projects.ChangeJoining(t.Context(), database.NewContext(), karo, fair.ID(), repositories.JoiningRequest)

	// assert
	require.NoError(t, err)

	waiting, err := projects.Requests(t.Context(), database.NewContext(), karo, fair.ID())
	require.NoError(t, err)
	assert.Len(t, waiting, 1)
}
