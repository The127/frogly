package dbtest

import (
	"time"

	"github.com/the127/frogly/internal/repositories"
)

// joinRequestsSuite checks requests to join: who may ask, what members
// see, and that an answered request is gone.
type joinRequestsSuite struct {
	base
}

func (s *joinRequestsSuite) asking(project *repositories.Project, user *repositories.User) (*repositories.JoinRequest, error) {
	request, err := repositories.NewJoinRequest(project.ID(), user.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.JoinRequests().Insert(request)

	return request, unit.SaveChanges(s.T().Context())
}

// askingAt saves a request asked at the given time, for a test of the
// order
func (s *joinRequestsSuite) askingAt(project *repositories.Project, user *repositories.User, askedAt time.Time) {
	request, err := repositories.NewJoinRequest(project.ID(), user.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.JoinRequests().Insert(repositories.NewJoinRequestFromDB(request.Base, project.ID(), user.ID(), askedAt))
	s.save(unit)
}

func (s *joinRequestsSuite) takingRequests(name string, creator *repositories.User) *repositories.Project {
	return s.savedProjectJoining(name, creator, repositories.VisibilityPublic, repositories.JoiningRequest)
}

// TestARequestComesBackAsAsked checks that a saved request loads with
// every field.
func (s *joinRequestsSuite) TestARequestComesBackAsAsked() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.takingRequests("Summer fair", karo)
	asked, err := s.asking(fair, ana)
	s.Require().NoError(err)

	// act
	found, err := s.database.NewContext().JoinRequests().ByProjectAndUser(s.T().Context(), fair.ID(), ana.ID())

	// assert
	s.Require().NoError(err)
	s.Equal(asked.ID(), found.ID())
	s.Equal(1, found.Version())
	s.Equal(fair.ID(), found.ProjectID())
	s.Equal(ana.ID(), found.UserID())
	s.Equal(asked.AskedAt(), found.AskedAt())
}

// TestOnlyAProjectThatTakesRequestsCanBeAsked checks that a request for an
// open, closed, private or missing project is refused as not found and
// saves nothing.
func (s *joinRequestsSuite) TestOnlyAProjectThatTakesRequestsCanBeAsked() {
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	nowhere, err := repositories.NewProject("Nowhere", repositories.VisibilityPrivate, "")
	s.Require().NoError(err)

	for _, test := range []struct {
		name    string
		project *repositories.Project
	}{
		{name: "open", project: s.savedProjectJoining("Fair", karo, repositories.VisibilityPublic, repositories.JoiningOpen)},
		{name: "closed", project: s.savedProjectJoining("Board", karo, repositories.VisibilityPublic, repositories.JoiningClosed)},
		{name: "private", project: s.savedProject("Secret", karo)},
		{name: "missing", project: nowhere},
	} {
		s.Run(test.name, func() {
			// act
			_, err := s.asking(test.project, ana)

			// assert
			s.Require().ErrorIs(err, repositories.ErrNotFound)

			_, err = s.database.NewContext().JoinRequests().ByProjectAndUser(s.T().Context(), test.project.ID(), ana.ID())
			s.Require().ErrorIs(err, repositories.ErrNotFound)
		})
	}
}

// TestAMemberCantAsk checks that a member's request is refused as not
// found and saves nothing, so a request can't slip in for someone who
// got in a moment before.
func (s *joinRequestsSuite) TestAMemberCantAsk() {
	// arrange
	karo := s.savedUser("Karo")
	fair := s.takingRequests("Summer fair", karo)

	// act
	_, err := s.asking(fair, karo)

	// assert
	s.Require().ErrorIs(err, repositories.ErrNotFound)

	_, err = s.database.NewContext().JoinRequests().ByProjectAndUser(s.T().Context(), fair.ID(), karo.ID())
	s.Require().ErrorIs(err, repositories.ErrNotFound)
}

// TestAUserAsksAProjectOnlyOnce checks that a second request of the same
// user is refused like a race.
func (s *joinRequestsSuite) TestAUserAsksAProjectOnlyOnce() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.takingRequests("Summer fair", karo)
	_, err := s.asking(fair, ana)
	s.Require().NoError(err)

	// act
	_, err = s.asking(fair, ana)

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}

// TestWaitingRequestsAreListedWithWhoAsked checks a project's list, the
// oldest first, with each asker, and only that project's requests.
func (s *joinRequestsSuite) TestWaitingRequestsAreListedWithWhoAsked() {
	// arrange
	karo, ana, tomas, lea := s.savedUser("Karo"), s.savedUser("Ana"), s.savedUser("Tomás"), s.savedUser("Lea")
	fair := s.takingRequests("Summer fair", karo)
	book := s.takingRequests("Book club", karo)
	start := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	s.askingAt(fair, ana, start.Add(2*time.Minute))
	s.askingAt(fair, tomas, start)
	s.askingAt(fair, lea, start.Add(time.Minute))
	s.askingAt(book, ana, start)

	// act
	waiting, err := s.database.NewContext().JoinRequests().ListForProject(s.T().Context(), fair.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(waiting, 3)
	s.Equal([]string{"Tomás", "Lea", "Ana"}, []string{waiting[0].Asker.Name(), waiting[1].Asker.Name(), waiting[2].Asker.Name()}, "the oldest first")
	s.Equal(tomas.ID(), waiting[0].Request.UserID())
	s.Equal("Tomás@frogly.local", waiting[0].Asker.Email())
	s.Equal(start, waiting[0].Request.AskedAt())
}

// TestThePublicListSaysWhichProjectsTheUserAsked checks the asked flag
// for someone with a request and someone without.
func (s *joinRequestsSuite) TestThePublicListSaysWhichProjectsTheUserAsked() {
	// arrange
	karo, ana, tomas := s.savedUser("Karo"), s.savedUser("Ana"), s.savedUser("Tomás")
	fair := s.takingRequests("Summer fair", karo)
	club := s.takingRequests("Book club", karo)
	_, err := s.asking(fair, ana)
	s.Require().NoError(err)
	projects := s.database.NewContext().Projects()

	// act
	anaList, anaErr := projects.ListPublicForNonMember(s.T().Context(), ana.ID())
	tomasList, tomasErr := projects.ListPublicForNonMember(s.T().Context(), tomas.ID())

	// assert
	s.Require().NoError(anaErr)
	s.Require().NoError(tomasErr)
	s.Require().Len(anaList, 2)
	s.Equal(club.ID(), anaList[0].Project.ID())
	s.False(anaList[0].Asked)
	s.Equal(fair.ID(), anaList[1].Project.ID())
	s.True(anaList[1].Asked)
	s.Require().Len(tomasList, 2)
	s.False(tomasList[0].Asked || tomasList[1].Asked, "Ana's request isn't Tomás's")
}

// TestAnAnsweredRequestIsGone checks that a deleted request is gone and
// can be asked again.
func (s *joinRequestsSuite) TestAnAnsweredRequestIsGone() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.takingRequests("Summer fair", karo)
	_, err := s.asking(fair, ana)
	s.Require().NoError(err)

	found, err := s.database.NewContext().JoinRequests().ByProjectAndUser(s.T().Context(), fair.ID(), ana.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.JoinRequests().Delete(found)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)

	_, err = s.database.NewContext().JoinRequests().ByProjectAndUser(s.T().Context(), fair.ID(), ana.ID())
	s.Require().ErrorIs(err, repositories.ErrNotFound)

	_, err = s.asking(fair, ana)
	s.Require().NoError(err, "asking again")
}

// TestAProjectsRequestsAreRemovedAtOnce checks that removing a project's
// requests removes all of them and only them.
func (s *joinRequestsSuite) TestAProjectsRequestsAreRemovedAtOnce() {
	// arrange
	karo, ana, tomas := s.savedUser("Karo"), s.savedUser("Ana"), s.savedUser("Tomás")
	fair := s.takingRequests("Summer fair", karo)
	club := s.takingRequests("Book club", karo)

	for _, asked := range []struct {
		project *repositories.Project
		user    *repositories.User
	}{{fair, ana}, {fair, tomas}, {club, ana}} {
		_, err := s.asking(asked.project, asked.user)
		s.Require().NoError(err)
	}

	unit := s.database.NewContext()
	unit.JoinRequests().DeleteForProject(fair.ID())

	// act
	err := unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)

	requests := s.database.NewContext().JoinRequests()
	fairWaiting, fairErr := requests.ListForProject(s.T().Context(), fair.ID())
	clubWaiting, clubErr := requests.ListForProject(s.T().Context(), club.ID())
	s.Require().NoError(fairErr)
	s.Require().NoError(clubErr)
	s.Empty(fairWaiting)
	s.Len(clubWaiting, 1, "another project's request stays")
}

// TestRemovingTheRequestsOfAProjectWithNoneIsNoError checks that removing
// the requests of a project nobody asked doesn't fail, unlike deleting
// one request that is gone.
func (s *joinRequestsSuite) TestRemovingTheRequestsOfAProjectWithNoneIsNoError() {
	// arrange
	fair := s.takingRequests("Summer fair", s.savedUser("Karo"))
	unit := s.database.NewContext()
	unit.JoinRequests().DeleteForProject(fair.ID())

	// act
	err := unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)
}

// TestAProjectNobodyIsInDropsItsRequests checks that an empty project
// that opens drops the requests nobody can answer any more.
func (s *joinRequestsSuite) TestAProjectNobodyIsInDropsItsRequests() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.takingRequests("Summer fair", karo)
	_, err := s.asking(fair, ana)
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Memberships().Delete(fair.ID(), karo.ID())
	unit.Projects().OpenIfEmpty(fair.ID())

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)

	waiting, err := s.database.NewContext().JoinRequests().ListForProject(s.T().Context(), fair.ID())
	s.Require().NoError(err)
	s.Empty(waiting)
}

// TestDeletingARequestThatIsGoneIsARace checks that a request someone else
// answered in between can't be answered again.
func (s *joinRequestsSuite) TestDeletingARequestThatIsGoneIsARace() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.takingRequests("Summer fair", karo)
	_, err := s.asking(fair, ana)
	s.Require().NoError(err)

	found, err := s.database.NewContext().JoinRequests().ByProjectAndUser(s.T().Context(), fair.ID(), ana.ID())
	s.Require().NoError(err)

	first := s.database.NewContext()
	first.JoinRequests().Delete(found)
	s.save(first)

	again := s.database.NewContext()
	again.JoinRequests().Delete(found)

	// act
	err = again.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}
