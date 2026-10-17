package dbtest

import (
	"github.com/the127/frogly/internal/repositories"
)

// unitOfWorkSuite checks the unit of work: what it writes, when, and all or
// nothing.
type unitOfWorkSuite struct {
	base
}

// TestNothingIsWrittenBeforeSaveChanges checks that recorded writes stay in the unit
// of work until it saves.
func (s *unitOfWorkSuite) TestNothingIsWrittenBeforeSaveChanges() {
	// arrange
	user, err := repositories.NewUser("Karo", "karo@frogly.local")
	s.Require().NoError(err)

	writing := s.database.NewContext()
	writing.Users().Insert(user)

	// act
	_, err = s.database.NewContext().Users().ByID(s.T().Context(), user.ID())

	// assert
	s.Require().ErrorIs(err, repositories.ErrNotFound)
}

// TestSaveChangesWritesAllOrNothing checks that when one write fails, none of the
// unit of work's writes stays.
func (s *unitOfWorkSuite) TestSaveChangesWritesAllOrNothing() {
	// arrange
	taken := s.savedUser("Karo")
	s.savedIdentity(taken)

	newcomer, err := repositories.NewUser("Ana", "ana@frogly.local")
	s.Require().NoError(err)

	duplicate, err := repositories.NewIdentity(newcomer.ID(), issuer, subject)
	s.Require().NoError(err)

	failing := s.database.NewContext()
	failing.Users().Insert(newcomer)
	failing.Identities().Insert(duplicate)

	// act
	err = failing.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)

	_, err = s.database.NewContext().Users().ByID(s.T().Context(), newcomer.ID())
	s.Require().ErrorIs(err, repositories.ErrNotFound)
}

// TestAFailedSaveForgetsItsChanges checks that a unit of work can go on
// after a failed save, for example to look up what won a race, without
// trying the failed writes again.
func (s *unitOfWorkSuite) TestAFailedSaveForgetsItsChanges() {
	// arrange
	karo := s.savedUser("Karo")
	s.savedIdentity(karo)

	duplicate, err := repositories.NewIdentity(karo.ID(), issuer, subject)
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Identities().Insert(duplicate)
	s.Require().ErrorIs(unit.SaveChanges(s.T().Context()), repositories.ErrConcurrentModification)

	ana, err := repositories.NewUser("Ana", "ana@frogly.local")
	s.Require().NoError(err)

	unit.Users().Insert(ana)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().NoError(err)
}
