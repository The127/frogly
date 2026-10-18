package dbtest

import (
	"slices"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/repositories"
)

// projectsSuite checks projects and their memberships: who sees which
// project, and in which order.
type projectsSuite struct {
	base
}

// TestProjectsAreListedForTheirMembersOnly checks that a user's list holds their
// projects only, sorted by name byte by byte.
func (s *projectsSuite) TestProjectsAreListedForTheirMembersOnly() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProject("garden", karo)
	board := s.savedProject("Board", karo)
	s.savedProject("Kitchen", ana)

	// act
	projects, err := s.database.NewContext().Projects().ListForMember(s.T().Context(), karo.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(projects, 2)
	s.Equal(board.ID(), projects[0].ID(), "upper case sorts before lower case")
	s.Equal("Board", projects[0].Name())
	s.Equal(1, projects[0].Version())
	s.Equal(garden.ID(), projects[1].ID())
}

// TestProjectsWithTheSameNameAreListedByID checks that projects with the same name
// come in the order of their IDs.
func (s *projectsSuite) TestProjectsWithTheSameNameAreListedByID() {
	// arrange
	karo := s.savedUser("Karo")
	first := s.savedProject("Garden", karo)
	second := s.savedProject("Garden", karo)

	// act
	projects, err := s.database.NewContext().Projects().ListForMember(s.T().Context(), karo.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(projects, 2)
	s.Equal(first.ID(), projects[0].ID(), "version 7 IDs grow with time")
	s.Equal(second.ID(), projects[1].ID())
}

// TestAProjectIsFoundForItsMembers checks that a member finds the project
// by its ID.
func (s *projectsSuite) TestAProjectIsFoundForItsMembers() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)

	// act
	found, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), garden.ID(), karo.ID())

	// assert
	s.Require().NoError(err)
	s.Equal(garden.ID(), found.ID())
	s.Equal(1, found.Version())
	s.Equal("Garden", found.Name())
}

// TestAProjectIsntFoundForOthers checks that someone who isn't a member
// gets the same answer as for a project that doesn't exist.
func (s *projectsSuite) TestAProjectIsntFoundForOthers() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProject("Garden", karo)
	// a member of another project, so a check of only one of the two IDs
	// would find something
	s.savedProject("Kitchen", ana)

	tests := []struct {
		name      string
		projectID uuid.UUID
		userID    uuid.UUID
	}{
		{name: "not a member", projectID: garden.ID(), userID: ana.ID()},
		{name: "no such project", projectID: uuid.Must(uuid.NewV7()), userID: karo.ID()},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			// act
			_, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), test.projectID, test.userID)

			// assert
			s.Require().ErrorIs(err, repositories.ErrNotFound)
		})
	}
}

// TestAUserIsAMemberOfAProjectOnlyOnce checks that a second membership of the same
// user in the same project fails.
func (s *projectsSuite) TestAUserIsAMemberOfAProjectOnlyOnce() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	again, err := repositories.NewMembership(garden.ID(), karo.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Memberships().Insert(again)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}

// TestAMembershipNeedsAnExistingProject checks that a membership can't
// point to a project the database doesn't have, and that this isn't
// mistaken for a race.
func (s *projectsSuite) TestAMembershipNeedsAnExistingProject() {
	// arrange
	karo := s.savedUser("Karo")
	nowhere, err := repositories.NewProject("Nowhere", repositories.VisibilityPrivate, repositories.JoiningClosed)
	s.Require().NoError(err)

	membership, err := repositories.NewMembership(nowhere.ID(), karo.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Memberships().Insert(membership)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)
	s.NotErrorIs(err, repositories.ErrConcurrentModification)
}

// TestAProjectKeepsItsVisibility checks that a project loads as public or
// private, as it was saved.
func (s *projectsSuite) TestAProjectKeepsItsVisibility() {
	for _, visibility := range []repositories.Visibility{repositories.VisibilityPublic, repositories.VisibilityPrivate} {
		s.Run(string(visibility), func() {
			// arrange
			karo := s.savedUser("Karo")
			garden := s.savedProjectWith("Garden", karo, visibility)

			// act
			found, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), garden.ID(), karo.ID())

			// assert
			s.Require().NoError(err)
			s.Equal(visibility, found.Visibility())
		})
	}
}

// TestPublicProjectsAreListedForNonMembers checks that the list holds the
// public projects of others only, sorted like a member's list.
func (s *projectsSuite) TestPublicProjectsAreListedForNonMembers() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProjectWith("garden", ana, repositories.VisibilityPublic)
	board := s.savedProjectWith("Board", ana, repositories.VisibilityPublic)
	s.savedProjectWith("Secret", ana, repositories.VisibilityPrivate)
	s.savedProjectWith("Kitchen", karo, repositories.VisibilityPublic)

	// act
	projects, err := s.database.NewContext().Projects().ListPublicForNonMember(s.T().Context(), karo.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(projects, 2)
	s.Equal(board.ID(), projects[0].Project.ID(), "upper case sorts before lower case")
	s.Equal("Board", projects[0].Project.Name())
	s.Equal(repositories.VisibilityPublic, projects[0].Project.Visibility())
	s.Equal(garden.ID(), projects[1].Project.ID())
}

// TestAChangedVisibilityIsSaved checks that an update writes the new
// visibility and the next version.
func (s *projectsSuite) TestAChangedVisibilityIsSaved() {
	// arrange
	karo := s.savedUser("Karo")
	saved := s.savedProject("Garden", karo)
	garden, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), saved.ID(), karo.ID())
	s.Require().NoError(err)
	s.Require().NoError(garden.ChangeVisibility(repositories.VisibilityPublic))

	unit := s.database.NewContext()
	unit.Projects().Update(garden)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)

	found, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), garden.ID(), karo.ID())
	s.Require().NoError(err)
	s.Equal(repositories.VisibilityPublic, found.Visibility())
	s.Equal(2, found.Version())
}

// TestAStaleProjectUpdateIsAConcurrentModification checks that a change
// to a project someone saved in between is refused.
func (s *projectsSuite) TestAStaleProjectUpdateIsAConcurrentModification() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)

	first, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), garden.ID(), karo.ID())
	s.Require().NoError(err)
	second, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), garden.ID(), karo.ID())
	s.Require().NoError(err)

	s.Require().NoError(first.ChangeVisibility(repositories.VisibilityPublic))
	unit := s.database.NewContext()
	unit.Projects().Update(first)
	s.save(unit)

	s.Require().NoError(second.ChangeVisibility(repositories.VisibilityPublic))
	stale := s.database.NewContext()
	stale.Projects().Update(second)

	// act
	err = stale.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}

// TestAPublicProjectWithSeveralMembersIsListedOnce checks that a project
// with several members shows up once for someone outside it, and not at all
// for one of its members. A query that joins the memberships could get
// either wrong.
func (s *projectsSuite) TestAPublicProjectWithSeveralMembersIsListedOnce() {
	// arrange
	karo, ana, tomas := s.savedUser("Karo"), s.savedUser("Ana"), s.savedUser("Tomás")
	shared := s.savedProjectWith("Shared", ana, repositories.VisibilityPublic)
	s.savedMember(shared, karo)
	others := s.savedProjectWith("Others", ana, repositories.VisibilityPublic)
	s.savedMember(others, tomas)

	// act
	projects, err := s.database.NewContext().Projects().ListPublicForNonMember(s.T().Context(), karo.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(projects, 1)
	s.Equal(others.ID(), projects[0].Project.ID())
}

// TestAProjectKeepsHowPeopleJoin checks that a public project loads open
// or closed, as it was saved, also in the list of public projects.
func (s *projectsSuite) TestAProjectKeepsHowPeopleJoin() {
	for _, joining := range []repositories.Joining{repositories.JoiningOpen, repositories.JoiningRequest, repositories.JoiningClosed} {
		s.Run(string(joining), func() {
			// arrange
			karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
			fair := s.savedProjectJoining("Summer fair", karo, repositories.VisibilityPublic, joining)

			// act
			found, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), fair.ID(), karo.ID())
			public, publicErr := s.database.NewContext().Projects().ListPublicForNonMember(s.T().Context(), ana.ID())

			// assert
			s.Require().NoError(err)
			s.Require().NoError(publicErr)
			s.Equal(joining, found.Joining())

			// the subtests share the database, so the list holds the other's project too
			listed := slices.IndexFunc(public, func(listed repositories.PublicProject) bool { return listed.Project.ID() == fair.ID() })
			s.Require().GreaterOrEqual(listed, 0)
			s.Equal(joining, public[listed].Project.Joining())
		})
	}
}

// TestAChangedJoiningIsSaved checks that an update writes the new setting.
func (s *projectsSuite) TestAChangedJoiningIsSaved() {
	// arrange
	karo := s.savedUser("Karo")
	saved := s.savedProjectJoining("Summer fair", karo, repositories.VisibilityPublic, repositories.JoiningClosed)
	fair, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), saved.ID(), karo.ID())
	s.Require().NoError(err)
	s.Require().NoError(fair.ChangeJoining(repositories.JoiningOpen))

	unit := s.database.NewContext()
	unit.Projects().Update(fair)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)

	found, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), fair.ID(), karo.ID())
	s.Require().NoError(err)
	s.Equal(repositories.JoiningOpen, found.Joining())
}

func (s *projectsSuite) joining(project *repositories.Project, user *repositories.User) error {
	membership, err := repositories.NewJoiningMembership(project.ID(), user.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Memberships().Insert(membership)

	return unit.SaveChanges(s.T().Context())
}

// TestSomeoneJoinsAnOpenProject checks that a joining membership is saved
// for a public project that is open.
func (s *projectsSuite) TestSomeoneJoinsAnOpenProject() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.savedProjectJoining("Summer fair", karo, repositories.VisibilityPublic, repositories.JoiningOpen)

	// act
	err := s.joining(fair, ana)

	// assert
	s.Require().NoError(err)

	found, err := s.database.NewContext().Projects().ByIDForMember(s.T().Context(), fair.ID(), ana.ID())
	s.Require().NoError(err)
	s.Equal(fair.ID(), found.ID())
}

// TestOnlyAnOpenProjectCanBeJoined checks that a joining membership for a
// closed, private or missing project is refused as not found and saves
// nothing, which the database decides as it writes.
func (s *projectsSuite) TestOnlyAnOpenProjectCanBeJoined() {
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	nowhere, err := repositories.NewProject("Nowhere", repositories.VisibilityPrivate, "")
	s.Require().NoError(err)

	for _, test := range []struct {
		name    string
		project *repositories.Project
	}{
		{name: "closed", project: s.savedProjectJoining("Board", karo, repositories.VisibilityPublic, repositories.JoiningClosed)},
		{name: "taking requests", project: s.savedProjectJoining("Book club", karo, repositories.VisibilityPublic, repositories.JoiningRequest)},
		{name: "private", project: s.savedProject("Secret", karo)},
		{name: "missing", project: nowhere},
	} {
		s.Run(test.name, func() {
			// act
			err := s.joining(test.project, ana)

			// assert
			s.Require().ErrorIs(err, repositories.ErrNotFound)

			_, err = s.database.NewContext().Projects().ByIDForMember(s.T().Context(), test.project.ID(), ana.ID())
			s.Require().ErrorIs(err, repositories.ErrNotFound)
		})
	}
}

// TestJoiningTwiceIsARace checks that the same person joining again is
// refused like a race, as their membership exists already.
func (s *projectsSuite) TestJoiningTwiceIsARace() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	fair := s.savedProjectJoining("Summer fair", karo, repositories.VisibilityPublic, repositories.JoiningOpen)
	s.Require().NoError(s.joining(fair, ana))

	// act
	err := s.joining(fair, ana)

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}

// TestAPrivateProjectThatOthersCouldGetIntoIsRefused checks that the
// database refuses what the model never builds, a private project anyone
// could join or ask.
func (s *projectsSuite) TestAPrivateProjectThatOthersCouldGetIntoIsRefused() {
	karo := s.savedUser("Karo")

	for _, joining := range []repositories.Joining{repositories.JoiningOpen, repositories.JoiningRequest} {
		s.Run(string(joining), func() {
			// arrange
			project, err := repositories.NewProject("Secret", repositories.VisibilityPrivate, "")
			s.Require().NoError(err)

			reachable := repositories.NewProjectFromDB(project.Base, "Secret", repositories.VisibilityPrivate, joining)
			membership, err := repositories.NewMembership(reachable.ID(), karo.ID())
			s.Require().NoError(err)

			unit := s.database.NewContext()
			unit.Projects().Insert(reachable)
			unit.Memberships().Insert(membership)

			// act
			err = unit.SaveChanges(s.T().Context())

			// assert
			s.Require().Error(err)
			s.NotErrorIs(err, repositories.ErrConcurrentModification)
		})
	}
}

// TestARemovedMembershipIsGone checks that a member who is taken out
// doesn't find the project any more, and the others still do.
func (s *projectsSuite) TestARemovedMembershipIsGone() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProject("Garden", karo)
	s.savedMember(garden, ana)

	unit := s.database.NewContext()
	unit.Memberships().Delete(garden.ID(), ana.ID())

	// act
	err := unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)

	projects := s.database.NewContext().Projects()
	_, err = projects.ByIDForMember(s.T().Context(), garden.ID(), ana.ID())
	s.Require().ErrorIs(err, repositories.ErrNotFound)

	_, err = projects.ByIDForMember(s.T().Context(), garden.ID(), karo.ID())
	s.Require().NoError(err, "the other member stays")
}

// TestRemovingAMembershipThatIsGoneIsNoError checks that taking out
// someone who isn't a member doesn't fail, so leaving twice at once
// leaves once.
func (s *projectsSuite) TestRemovingAMembershipThatIsGoneIsNoError() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProject("Garden", karo)

	unit := s.database.NewContext()
	unit.Memberships().Delete(garden.ID(), ana.ID())

	// act
	err := unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)
}
