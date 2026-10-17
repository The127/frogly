package dbtest

import (
	"time"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/repositories"
)

// ideasSuite checks ideas: a garden's contents, its order, and what an idea
// refers to.
type ideasSuite struct {
	base
}

// TestAnIdeaComesBackAsPlanted checks that a saved idea loads with every
// field, as version 1.
func (s *ideasSuite) TestAnIdeaComesBackAsPlanted() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	idea := s.savedIdea(garden, karo, "grüne Ideen 🌱\tmit Tab")

	// act
	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(ideas, 1)
	s.Equal(idea.ID(), ideas[0].Idea.ID())
	s.Equal(1, ideas[0].Idea.Version())
	s.Equal(garden.ID(), ideas[0].Idea.ProjectID())
	s.Equal(karo.ID(), ideas[0].Idea.PlanterID())
	s.Equal("grüne Ideen 🌱\tmit Tab", ideas[0].Idea.Text())
	s.Equal(idea.PlantedAt(), ideas[0].Idea.PlantedAt())
	s.Equal("Karo", ideas[0].PlantedBy)
}

// TestEachIdeaIsListedWithItsOwnPlanter checks that every idea in a
// garden carries the name of whoever planted it, not someone else's.
func (s *ideasSuite) TestEachIdeaIsListedWithItsOwnPlanter() {
	// arrange
	karo, ana := s.savedUser("Karo"), s.savedUser("Ana")
	garden := s.savedProject("Garden", karo)
	s.savedIdea(garden, karo, "from Karo")
	s.savedIdea(garden, ana, "from Ana")

	// act
	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(ideas, 2)
	s.Equal("from Ana", ideas[0].Idea.Text())
	s.Equal("Ana", ideas[0].PlantedBy)
	s.Equal("from Karo", ideas[1].Idea.Text())
	s.Equal("Karo", ideas[1].PlantedBy)
}

// TestAGardenListsItsOwnIdeasNewestFirst checks that a project's garden
// holds only its own ideas, the newest first.
func (s *ideasSuite) TestAGardenListsItsOwnIdeasNewestFirst() {
	// arrange
	karo := s.savedUser("Karo")
	garden, kitchen := s.savedProject("Garden", karo), s.savedProject("Kitchen", karo)
	now := time.Now().UTC().Truncate(time.Microsecond)
	// the newer idea is saved first and gets the smaller ID, so only the
	// time can put it first
	newer := s.savedIdeaAt(garden, karo, "newer", now)
	older := s.savedIdeaAt(garden, karo, "older", now.Add(-time.Hour))
	s.savedIdea(kitchen, karo, "a bigger fridge")

	// act
	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(ideas, 2)
	s.Equal(newer.ID(), ideas[0].Idea.ID())
	s.Equal(older.ID(), ideas[1].Idea.ID())
}

// TestIdeasPlantedAtTheSameTimeAreListedByIDNewestFirst checks that ideas
// with the same time come in the reverse order of their IDs.
func (s *ideasSuite) TestIdeasPlantedAtTheSameTimeAreListedByIDNewestFirst() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	now := time.Now().UTC().Truncate(time.Microsecond)
	first := s.savedIdeaAt(garden, karo, "first", now)
	second := s.savedIdeaAt(garden, karo, "second", now)

	// act
	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())

	// assert
	s.Require().NoError(err)
	s.Require().Len(ideas, 2)
	s.Equal(second.ID(), ideas[0].Idea.ID(), "version 7 IDs grow with time")
	s.Equal(first.ID(), ideas[1].Idea.ID())
}

// TestAnIdeaNeedsAnExistingProject checks that an idea can't be planted in
// a project the database doesn't have, and that this isn't mistaken for a
// race.
func (s *ideasSuite) TestAnIdeaNeedsAnExistingProject() {
	// arrange
	karo := s.savedUser("Karo")
	nowhere := uuid.Must(uuid.NewV7())
	idea, err := repositories.NewIdea(nowhere, karo.ID(), "nowhere to grow")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Ideas().Insert(idea)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)
	s.Require().NotErrorIs(err, repositories.ErrConcurrentModification)

	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), nowhere)
	s.Require().NoError(err)
	s.Empty(ideas)
}

// TestAnIdeaNeedsAnExistingPlanter checks that an idea can't name a
// planter the database doesn't have.
func (s *ideasSuite) TestAnIdeaNeedsAnExistingPlanter() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	idea, err := repositories.NewIdea(garden.ID(), uuid.Must(uuid.NewV7()), "from nobody")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Ideas().Insert(idea)

	// act
	err = unit.SaveChanges(s.T().Context())

	// assert
	s.Require().Error(err)
	s.Require().NotErrorIs(err, repositories.ErrConcurrentModification)

	ideas, err := s.database.NewContext().Ideas().ListInProject(s.T().Context(), garden.ID())
	s.Require().NoError(err)
	s.Empty(ideas)
}

// TestOneIdeaComesBackWithItsPlanter checks that a single idea loads with
// who planted it.
func (s *ideasSuite) TestOneIdeaComesBackWithItsPlanter() {
	// arrange
	karo := s.savedUser("Karo")
	garden := s.savedProject("Garden", karo)
	idea := s.savedIdea(garden, karo, "search by any word")

	// act
	one, err := s.database.NewContext().Ideas().ByIDInProject(s.T().Context(), garden.ID(), idea.ID())

	// assert
	s.Require().NoError(err)
	s.Equal(idea.ID(), one.Idea.ID())
	s.Equal("search by any word", one.Idea.Text())
	s.Equal("Karo", one.PlantedBy)
}

// TestAnIdeaIsFoundOnlyInItsProject checks that an idea's ID doesn't find
// it through another project, nor an ID that no idea has.
func (s *ideasSuite) TestAnIdeaIsFoundOnlyInItsProject() {
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
