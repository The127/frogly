// Package garden is what people do with a project's ideas, whatever way
// they ask for it.
package garden

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

// ErrIdeaNotFound means the project has no idea with the ID. It wraps
// projects.ErrNotFound, so it gets the same answer as a project nobody
// may see.
var ErrIdeaNotFound = fmt.Errorf("idea not found: %w", projects.ErrNotFound)

// Plant writes an idea down in the project's garden. A project that
// doesn't exist or that planter isn't a member of gives
// projects.ErrNotFound, and a text frogly can't use a
// *repositories.InvalidError.
func Plant(ctx context.Context, unit db.Context, planter *repositories.User, projectID uuid.UUID, text string) (*repositories.Idea, error) {
	_, err := projects.Get(ctx, unit, planter, projectID)
	if err != nil {
		return nil, err
	}

	idea, err := repositories.NewIdea(projectID, planter.ID(), text)
	if err != nil {
		return nil, err
	}

	unit.Ideas().Insert(idea)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saving the idea: %w", err)
	}

	return idea, nil
}

// List lists the project's garden for one of its members, newest first,
// with the name of whoever planted each idea. A project that doesn't exist
// or that user isn't a member of gives projects.ErrNotFound.
func List(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) ([]repositories.PlantedIdea, error) {
	_, err := projects.Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	return unit.Ideas().ListInProject(ctx, projectID)
}

// MeToo puts user behind an idea in the project's garden and gives the
// idea back with how many people want it. Saying it again, or as the
// planter, changes nothing, since they count already. A project user may
// not see gives projects.ErrNotFound, and an idea that isn't in it
// ErrIdeaNotFound.
func MeToo(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, ideaID uuid.UUID) (repositories.PlantedIdea, error) {
	_, err := projects.Get(ctx, unit, user, projectID)
	if err != nil {
		return repositories.PlantedIdea{}, err
	}

	idea, err := unit.Ideas().ByIDInProject(ctx, projectID, ideaID)
	if errors.Is(err, repositories.ErrNotFound) {
		return repositories.PlantedIdea{}, fmt.Errorf("idea %s in project %s: %w", ideaID, projectID, ErrIdeaNotFound)
	}

	if err != nil {
		return repositories.PlantedIdea{}, err
	}

	if idea.Idea.PlanterID() == user.ID() {
		return idea, nil
	}

	said, err := unit.MeToos().Said(ctx, ideaID, user.ID())
	if err != nil || said {
		return idea, err
	}

	meToo, err := repositories.NewMeToo(ideaID, user.ID())
	if err != nil {
		return repositories.PlantedIdea{}, err
	}

	unit.MeToos().Insert(meToo)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return repositories.PlantedIdea{}, fmt.Errorf("saving the me too of user %s for idea %s: %w", user.ID(), ideaID, err)
	}

	// loaded again, so the count has the me toos others saved meanwhile too
	return unit.Ideas().ByIDInProject(ctx, projectID, ideaID)
}
