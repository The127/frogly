package dbtest

import (
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
	nowhere, err := repositories.NewProject("Nowhere", repositories.VisibilityPrivate)
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
	s.Equal(board.ID(), projects[0].ID(), "upper case sorts before lower case")
	s.Equal("Board", projects[0].Name())
	s.Equal(repositories.VisibilityPublic, projects[0].Visibility())
	s.Equal(garden.ID(), projects[1].ID())
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
	s.Equal(others.ID(), projects[0].ID())
}
