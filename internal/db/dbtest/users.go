package dbtest

import (
	"github.com/the127/frogly/internal/repositories"
)

func names(users []*repositories.User) []string {
	listed := make([]string, 0, len(users))
	for _, user := range users {
		listed = append(listed, user.Name())
	}

	return listed
}

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

// TestPeopleAreFoundByPartOfTheirNameOrEmail checks that the search looks
// anywhere in the name and the email, without minding upper or lower case.
func (s *usersSuite) TestPeopleAreFoundByPartOfTheirNameOrEmail() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	s.savedUserWith("Ana Costa", "ana@school.org")
	s.savedUserWith("Diana Berg", "berg@school.org")
	s.savedUserWith("Tomás", "tomas@shop.example")
	s.savedUserWith("Lea", "anabel@mail.example")

	// act
	found, err := s.database.NewContext().Users().SearchNonMembers(s.T().Context(), garden.ID(), "ANA", 10)

	// assert
	s.Require().NoError(err)
	s.Equal([]string{"Ana Costa", "Diana Berg", "Lea"}, names(found))
}

// TestMembersArentFound checks that a project's members don't show up
// among the people to add to it.
func (s *usersSuite) TestMembersArentFound() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProject("Garden", karo)
	s.savedMember(garden, ana)
	// a member of another project only, who mustn't be left out
	kitchen := s.savedProject("Kitchen", s.savedUser("Anabel"))
	s.Require().NotNil(kitchen)

	// act
	found, err := s.database.NewContext().Users().SearchNonMembers(s.T().Context(), garden.ID(), "a", 10)

	// assert
	s.Require().NoError(err)
	s.Equal([]string{"Anabel"}, names(found))
}

// TestPeopleAreFoundByNameUpToTheLimit checks the order and the limit.
func (s *usersSuite) TestPeopleAreFoundByNameUpToTheLimit() {
	// arrange
	garden := s.savedProject("Garden", s.savedUser("Zoe"))
	s.savedUser("bert")
	s.savedUser("Carla")
	s.savedUser("Anna")

	// act
	found, err := s.database.NewContext().Users().SearchNonMembers(s.T().Context(), garden.ID(), "", 2)

	// assert
	s.Require().NoError(err)
	s.Equal([]string{"Anna", "Carla"}, names(found), "upper case sorts before lower case")
}

// TestWildcardsInTheSearchAreJustCharacters checks that % and _ only find
// themselves, so nobody can list every user with them.
func (s *usersSuite) TestWildcardsInTheSearchAreJustCharacters() {
	// arrange
	garden := s.savedProject("Garden", s.savedUser("Karo"))
	s.savedUserWith("Ana", "ana@school.org")
	s.savedUserWith("100% Lea", "lea@school.org")

	for _, test := range []struct {
		text string
		want []string
	}{
		{text: "%", want: []string{"100% Lea"}},
		{text: "_", want: []string{}},
	} {
		s.Run(test.text, func() {
			// act
			found, err := s.database.NewContext().Users().SearchNonMembers(s.T().Context(), garden.ID(), test.text, 10)

			// assert
			s.Require().NoError(err)
			s.Equal(test.want, names(found))
		})
	}
}
