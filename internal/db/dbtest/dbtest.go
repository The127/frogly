// Package dbtest is the contract every database adapter meets. An
// adapter's tests run Suite against a fresh database of their own.
package dbtest

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

// Suite checks an adapter against the contract. Open returns a fresh,
// empty, migrated database for every test, and registers its cleanup with
// the test it gets.
type Suite struct {
	suite.Suite

	Open func(t *testing.T) db.Database

	database db.Database
}

// SetupTest opens a fresh database for the test.
func (s *Suite) SetupTest() {
	s.database = s.Open(s.T())
}

func (s *Suite) save(unit db.Context) {
	s.Require().NoError(unit.SaveChanges(s.T().Context()))
}

func (s *Suite) savedUser(name string) *repositories.User {
	user, err := repositories.NewUser(name, name+"@frogly.local")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Users().Insert(user)
	s.save(unit)

	return user
}

func (s *Suite) savedProject(name string, owner *repositories.User) *repositories.Project {
	project, err := repositories.NewProject(name)
	s.Require().NoError(err)

	membership, err := repositories.NewMembership(project.ID(), owner.ID(), repositories.RoleOwner)
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Projects().Insert(project)
	unit.Memberships().Insert(membership)
	s.save(unit)

	return project
}

// TestNothingIsWrittenBeforeSaveChanges checks that recorded writes stay in the unit
// of work until it saves.
func (s *Suite) TestNothingIsWrittenBeforeSaveChanges() {
	// arrange
	user, err := repositories.NewUser("Karo", "karo@frogly.local")
	s.Require().NoError(err)

	writing := s.database.NewContext()
	writing.Users().Insert(user)

	// act
	_, err = s.database.NewContext().Users().First(s.T().Context(), user.ID())

	// assert
	s.Require().ErrorIs(err, repositories.ErrNotFound)
}

// TestAUserComesBackAsSaved checks that a saved user loads with every field, as
// version 1.
func (s *Suite) TestAUserComesBackAsSaved() {
	// arrange
	user := s.savedUser("Karo")

	// act
	loaded, err := s.database.NewContext().Users().First(s.T().Context(), user.ID())

	// assert
	s.Require().NoError(err)
	s.Equal(user.ID(), loaded.ID())
	s.Equal(1, loaded.Version())
	s.Equal("Karo", loaded.Name())
	s.Equal("Karo@frogly.local", loaded.Email())
}

// TestAnUpdateWritesTheWholeUser checks that an update writes every field and the next
// version.
func (s *Suite) TestAnUpdateWritesTheWholeUser() {
	// arrange
	user := s.savedUser("Karo")
	unit := s.database.NewContext()
	loaded, err := unit.Users().First(s.T().Context(), user.ID())
	s.Require().NoError(err)

	loaded.Refresh("Karolin", "karolin@frogly.local")
	unit.Users().Update(loaded)

	// act
	s.save(unit)

	// assert
	reloaded, err := s.database.NewContext().Users().First(s.T().Context(), user.ID())
	s.Require().NoError(err)
	s.Equal(2, reloaded.Version())
	s.Equal("Karolin", reloaded.Name())
	s.Equal("karolin@frogly.local", reloaded.Email())
}

// TestAStaleUpdateIsAConcurrentModification checks that an update of a row that
// changed since it was loaded fails, and the other change stays.
func (s *Suite) TestAStaleUpdateIsAConcurrentModification() {
	// arrange
	user := s.savedUser("Karo")
	first, second := s.database.NewContext(), s.database.NewContext()

	firstCopy, err := first.Users().First(s.T().Context(), user.ID())
	s.Require().NoError(err)

	secondCopy, err := second.Users().First(s.T().Context(), user.ID())
	s.Require().NoError(err)

	firstCopy.Refresh("Karolin", "")
	first.Users().Update(firstCopy)
	s.save(first)

	secondCopy.Refresh("Kara", "")
	second.Users().Update(secondCopy)

	// act
	err = second.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)

	reloaded, err := s.database.NewContext().Users().First(s.T().Context(), user.ID())
	s.Require().NoError(err)
	s.Equal("Karolin", reloaded.Name())
}

// TestAnIdentityIsFoundByIssuerAndSubject checks that issuer and subject together find
// an identity, and nothing else does.
func (s *Suite) TestAnIdentityIsFoundByIssuerAndSubject() {
	// arrange
	user := s.savedUser("Karo")
	identity, err := repositories.NewIdentity(user.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Identities().Insert(identity)
	s.save(unit)

	reading := s.database.NewContext().Identities()

	// act
	found, foundErr := reading.FirstOrNil(s.T().Context(), "http://issuer.example", "karo-id")
	otherIssuer, otherIssuerErr := reading.FirstOrNil(s.T().Context(), "http://elsewhere.example", "karo-id")

	// assert
	s.Require().NoError(foundErr)
	s.Require().NotNil(found)
	s.Equal(identity.ID(), found.ID())
	s.Equal(user.ID(), found.UserID())

	s.Require().NoError(otherIssuerErr)
	s.Nil(otherIssuer)
}

// TestASecondIdentityForTheSameLoginIsAConcurrentModification checks that two
// identities can't share an issuer and subject.
func (s *Suite) TestASecondIdentityForTheSameLoginIsAConcurrentModification() {
	// arrange
	user := s.savedUser("Karo")
	first, err := repositories.NewIdentity(user.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Identities().Insert(first)
	s.save(unit)

	second, err := repositories.NewIdentity(user.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	racing := s.database.NewContext()
	racing.Identities().Insert(second)

	// act
	err = racing.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}

// TestSaveChangesWritesAllOrNothing checks that when one write fails, none of the
// unit of work's writes stays.
func (s *Suite) TestSaveChangesWritesAllOrNothing() {
	// arrange
	taken := s.savedUser("Karo")
	identity, err := repositories.NewIdentity(taken.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Identities().Insert(identity)
	s.save(unit)

	newcomer, err := repositories.NewUser("Ana", "ana@frogly.local")
	s.Require().NoError(err)

	duplicate, err := repositories.NewIdentity(newcomer.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	failing := s.database.NewContext()
	failing.Users().Insert(newcomer)
	failing.Identities().Insert(duplicate)

	// act
	err = failing.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)

	_, err = s.database.NewContext().Users().First(s.T().Context(), newcomer.ID())
	s.Require().ErrorIs(err, repositories.ErrNotFound)
}

// TestProjectsAreListedForTheirMembersOnly checks that a user's list holds their
// projects only, sorted by name byte by byte.
func (s *Suite) TestProjectsAreListedForTheirMembersOnly() {
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
func (s *Suite) TestProjectsWithTheSameNameAreListedByID() {
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

// TestAUserIsAMemberOfAProjectOnlyOnce checks that a second membership of the same
// user in the same project fails.
func (s *Suite) TestAUserIsAMemberOfAProjectOnlyOnce() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	again, err := repositories.NewMembership(garden.ID(), karo.ID(), repositories.RoleOwner)
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
func (s *Suite) TestAMembershipNeedsAnExistingProject() {
	// arrange
	karo := s.savedUser("Karo")
	nowhere, err := repositories.NewProject("Nowhere")
	s.Require().NoError(err)

	membership, err := repositories.NewMembership(nowhere.ID(), karo.ID(), repositories.RoleOwner)
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Memberships().Insert(membership)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)
	s.NotErrorIs(err, repositories.ErrConcurrentModification)
}

// TestUpdatingAUserThatIsntStoredIsNotFound checks that an update of a
// row the database doesn't have fails instead of writing nothing.
func (s *Suite) TestUpdatingAUserThatIsntStoredIsNotFound() {
	// arrange
	user, err := repositories.NewUser("Karo", "karo@frogly.local")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Users().Update(user)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrNotFound)
}

// TestAFailedSaveForgetsItsChanges checks that a unit of work can go on
// after a failed save, for example to look up what won a race, without
// trying the failed writes again.
func (s *Suite) TestAFailedSaveForgetsItsChanges() {
	// arrange
	karo := s.savedUser("Karo")
	identity, err := repositories.NewIdentity(karo.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	first := s.database.NewContext()
	first.Identities().Insert(identity)
	s.save(first)

	duplicate, err := repositories.NewIdentity(karo.ID(), "http://issuer.example", "karo-id")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Identities().Insert(duplicate)
	s.Require().Error(unit.SaveChanges(s.T().Context()))

	ana, err := repositories.NewUser("Ana", "ana@frogly.local")
	s.Require().NoError(err)

	unit.Users().Insert(ana)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)
}
