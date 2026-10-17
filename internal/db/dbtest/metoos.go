package dbtest

import (
	"github.com/google/uuid"

	"github.com/the127/frogly/internal/repositories"
)

// meToosSuite checks me toos: how they count the people behind an idea,
// and that a user counts once.
type meToosSuite struct {
	base
}

func (s *meToosSuite) savedMeToo(idea *repositories.Idea, user *repositories.User) {
	meToo, err := repositories.NewMeToo(idea.ID(), user.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.MeToos().Insert(meToo)
	s.save(unit)
}

// TestAnIdeaCountsItsPlanter checks that an idea nobody else wants yet has
// one person behind it.
func (s *meToosSuite) TestAnIdeaCountsItsPlanter() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	s.savedIdea(garden, karo, "search by any word")

	// act
	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(ideas, 1)
	s.Equal(1, ideas[0].People)
}

// TestEachMeTooCountsOnePersonMore checks the count in the garden and for
// one idea, each idea with its own me toos.
func (s *meToosSuite) TestEachMeTooCountsOnePersonMore() {
	// arrange
	karo, ana, tomas := s.savedUser("Karo"), s.savedUser("Ana"), s.savedUser("Tomás")
	garden := s.savedProject("Garden", karo)
	search := s.savedIdea(garden, karo, "search by any word")
	s.savedIdea(garden, karo, "a calmer color for done tickets")
	s.savedMeToo(search, ana)
	s.savedMeToo(search, tomas)

	// act
	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())
	one, oneErr := s.database.NewContext().Ideas().ByIDInProject(s.T().Context(), garden.ID(), search.ID())

	// assert
	s.Require().NoError(err)
	s.Require().NoError(oneErr)
	s.Require().Len(ideas, 2)
	s.Equal(1, ideas[0].People, "the newer idea has nobody but its planter")
	s.Equal(3, ideas[1].People)
	s.Equal(3, one.People)
	s.Equal("Karo", one.PlantedBy)
	s.Equal(search.ID(), one.Idea.ID())
}

// TestAPlantersMeTooDoesntCount checks that the planter counts once, even
// with a me too of their own stored.
func (s *meToosSuite) TestAPlantersMeTooDoesntCount() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	idea := s.savedIdea(garden, karo, "search by any word")
	s.savedMeToo(idea, karo)

	// act
	one, err := s.database.NewContext().Ideas().ByIDInProject(s.T().Context(), garden.ID(), idea.ID())

	// assert
	s.Require().NoError(err)
	s.Equal(1, one.People)
}

// TestAUserSaysMeTooToAnIdeaOnlyOnce checks that a second me too of the
// same user is refused like a race.
func (s *meToosSuite) TestAUserSaysMeTooToAnIdeaOnlyOnce() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	idea := s.savedIdea(s.savedProject("Garden", karo), karo, "search by any word")
	s.savedMeToo(idea, ana)

	again, err := repositories.NewMeToo(idea.ID(), ana.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.MeToos().Insert(again)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().ErrorIs(err, repositories.ErrConcurrentModification)
}

// TestWhoSaidMeTooIsKnown checks Said for someone who said it and someone
// who didn't.
func (s *meToosSuite) TestWhoSaidMeTooIsKnown() {
	// arrange
	karo, ana, tomas := s.savedUser("Karo"), s.savedUser("Ana"), s.savedUser("Tomás")
	idea := s.savedIdea(s.savedProject("Garden", karo), karo, "search by any word")
	s.savedMeToo(idea, ana)
	meToos := s.database.NewContext().MeToos()

	// act
	anaSaid, anaErr := meToos.Said(s.T().Context(), idea.ID(), ana.ID())
	tomasSaid, tomasErr := meToos.Said(s.T().Context(), idea.ID(), tomas.ID())

	// assert
	s.Require().NoError(anaErr)
	s.Require().NoError(tomasErr)
	s.True(anaSaid)
	s.False(tomasSaid)
}

// TestAnIdeaIsFoundOnlyInItsProject checks that an idea's ID doesn't find
// it through another project, nor an ID that no idea has.
func (s *meToosSuite) TestAnIdeaIsFoundOnlyInItsProject() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	kitchen := s.savedProject("Kitchen", karo)
	idea := s.savedIdea(garden, karo, "search by any word")

	for _, test := range []struct {
		name      string
		projectID uuid.UUID
		ideaID    uuid.UUID
	}{
		{name: "another project", projectID: kitchen.ID(), ideaID: idea.ID()},
		{name: "no such idea", projectID: garden.ID(), ideaID: uuid.Must(uuid.NewV7())},
	} {
		s.Run(test.name, func() {
			// act
			_, err := s.database.NewContext().Ideas().ByIDInProject(s.T().Context(), test.projectID, test.ideaID)

			// assert
			s.Require().ErrorIs(err, repositories.ErrNotFound)
		})
	}
}

// TestAMeTooNeedsAnExistingIdea checks that a me too can't point to an
// idea the database doesn't have, and that this isn't taken for a race.
func (s *meToosSuite) TestAMeTooNeedsAnExistingIdea() {
	// arrange
	karo := s.savedUser("Karo")
	meToo, err := repositories.NewMeToo(uuid.Must(uuid.NewV7()), karo.ID())
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.MeToos().Insert(meToo)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)
	s.NotErrorIs(err, repositories.ErrConcurrentModification)
}
