package dbtest

import (
	"github.com/the127/frogly/internal/repositories"
)

// usersSuite checks users: loading, and updates with their version check.
type usersSuite struct {
	base
}

// TestAUserComesBackAsSaved checks that a saved user loads with every field, as
// version 1.
func (s *usersSuite) TestAUserComesBackAsSaved() {
	// arrange
	user := s.savedUser("Karo")

	// act
	loaded, err := s.database.NewContext().Users().ByID(s.T().Context(), user.ID())

	// assert
	s.Require().NoError(err)
	s.Equal(user.ID(), loaded.ID())
	s.Equal(1, loaded.Version())
	s.Equal("Karo", loaded.Name())
	s.Equal("Karo@frogly.local", loaded.Email())
}

// TestAnUpdateWritesTheWholeUser checks that an update writes every field and the next
// version.
func (s *usersSuite) TestAnUpdateWritesTheWholeUser() {
	// arrange
	user := s.savedUser("Karo")
	unit := s.database.NewContext()
	loaded, err := unit.Users().ByID(s.T().Context(), user.ID())
	s.Require().NoError(err)

	loaded.Refresh("Karolin", "karolin@frogly.local")
	unit.Users().Update(loaded)

	// act
	s.save(unit)

	// assert
	reloaded, err := s.database.NewContext().Users().ByID(s.T().Context(), user.ID())
	s.Require().NoError(err)
	s.Equal(2, reloaded.Version())
	s.Equal("Karolin", reloaded.Name())
	s.Equal("karolin@frogly.local", reloaded.Email())
}

// TestAStaleUpdateIsAConcurrentModification checks that an update of a row that
// changed since it was loaded fails, and the other change stays.
func (s *usersSuite) TestAStaleUpdateIsAConcurrentModification() {
	// arrange
	user := s.savedUser("Karo")
	first, second := s.database.NewContext(), s.database.NewContext()

	firstCopy, err := first.Users().ByID(s.T().Context(), user.ID())
	s.Require().NoError(err)

	secondCopy, err := second.Users().ByID(s.T().Context(), user.ID())
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

	reloaded, err := s.database.NewContext().Users().ByID(s.T().Context(), user.ID())
	s.Require().NoError(err)
	s.Equal("Karolin", reloaded.Name())
}

// TestUpdatingAUserThatIsntStoredIsNotFound checks that an update of a
// row the database doesn't have fails instead of writing nothing.
func (s *usersSuite) TestUpdatingAUserThatIsntStoredIsNotFound() {
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
