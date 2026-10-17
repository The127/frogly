// Package projects is what people do with projects, whatever way they ask
// for it.
package projects

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

// ErrNotFound means the project doesn't exist or the user isn't a member
// of it. It doesn't say which, so nobody learns which projects exist.
var ErrNotFound = errors.New("project not found")

// Create makes a project called name with owner as its only member. A name
// frogly can't use gives a *repositories.InvalidError.
func Create(ctx context.Context, unit db.Context, owner *repositories.User, name string) (*repositories.Project, error) {
	project, err := repositories.NewProject(name, repositories.VisibilityPrivate)
	if err != nil {
		return nil, err
	}

	membership, err := repositories.NewMembership(project.ID(), owner.ID(), repositories.RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("making %s the owner: %w", owner.ID(), err)
	}

	unit.Projects().Insert(project)
	unit.Memberships().Insert(membership)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saving the project: %w", err)
	}

	return project, nil
}

// ListFor lists the projects user is a member of.
func ListFor(ctx context.Context, unit db.Context, user *repositories.User) ([]*repositories.Project, error) {
	return unit.Projects().ListForMember(ctx, user.ID())
}

// Get gives user the project if they're a member of it, and ErrNotFound
// otherwise.
func Get(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) (*repositories.Project, error) {
	project, err := unit.Projects().ByIDForMember(ctx, projectID, user.ID())
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("project %s for user %s: %w", projectID, user.ID(), ErrNotFound)
	}

	return project, err
}
