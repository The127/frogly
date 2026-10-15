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

// ErrProjectNotFound means the project doesn't exist or the user isn't a
// member of it. It doesn't say which, so nobody learns which projects
// exist.
var ErrProjectNotFound = errors.New("project not found")

// Plant writes an idea down in the project's garden. A project that
// doesn't exist or that planter isn't a member of gives
// ErrProjectNotFound, and a text frogly can't use a
// *repositories.InvalidError.
func Plant(ctx context.Context, unit db.Context, planter *repositories.User, projectID uuid.UUID, text string) (*repositories.Idea, error) {
	err := requireMember(ctx, unit, projectID, planter.ID())
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
// or that user isn't a member of gives ErrProjectNotFound.
func List(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) ([]repositories.PlantedIdea, error) {
	err := requireMember(ctx, unit, projectID, user.ID())
	if err != nil {
		return nil, err
	}

	return unit.Ideas().ListInProject(ctx, projectID)
}

func requireMember(ctx context.Context, unit db.Context, projectID uuid.UUID, userID uuid.UUID) error {
	_, err := unit.Projects().ByIDForMember(ctx, projectID, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return fmt.Errorf("project %s for user %s: %w", projectID, userID, ErrProjectNotFound)
	}

	return err
}
