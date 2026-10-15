// Package garden is what people do with a project's ideas, whatever way
// they ask for it.
package garden

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

var errPlanterMissing = errors.New("the idea's planter is missing")

// PlantedIdea is an idea with the name of whoever planted it.
type PlantedIdea struct {
	Idea      *repositories.Idea
	PlantedBy string
}

// Plant writes an idea down in the project's garden. A project that
// doesn't exist or that planter isn't a member of gives
// repositories.ErrNotFound, and a text frogly can't use a
// *repositories.InvalidError.
func Plant(ctx context.Context, unit db.Context, planter *repositories.User, projectID uuid.UUID, text string) (*repositories.Idea, error) {
	_, err := unit.Projects().ByIDForMember(ctx, projectID, planter.ID())
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

// List lists the project's garden for one of its members, newest first. A
// project that doesn't exist or that user isn't a member of gives
// repositories.ErrNotFound.
func List(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) ([]PlantedIdea, error) {
	_, err := unit.Projects().ByIDForMember(ctx, projectID, user.ID())
	if err != nil {
		return nil, err
	}

	ideas, err := unit.Ideas().ListInProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// a garden has many ideas but few planters, so each is loaded once
	names := map[uuid.UUID]string{}
	planted := make([]PlantedIdea, 0, len(ideas))

	for _, idea := range ideas {
		name, known := names[idea.PlanterID()]
		if !known {
			planter, err := unit.Users().ByID(ctx, idea.PlanterID())
			if errors.Is(err, repositories.ErrNotFound) {
				// the database's foreign key rules this out, and the caller's
				// ErrNotFound means a project they can't see
				return nil, fmt.Errorf("idea %s: %w", idea.ID(), errPlanterMissing)
			}

			if err != nil {
				return nil, fmt.Errorf("loading the planter of idea %s: %w", idea.ID(), err)
			}

			name = planter.Name()
			names[idea.PlanterID()] = name
		}

		planted = append(planted, PlantedIdea{Idea: idea, PlantedBy: name})
	}

	return planted, nil
}
