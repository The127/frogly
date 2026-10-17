// Package projects is what people do with projects, whatever way they ask
// for it.
package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

// ErrNotFound means the project doesn't exist or the user isn't a member
// of it. It doesn't say which, so nobody learns which projects exist.
var ErrNotFound = errors.New("project not found")

// ErrUnknownPerson refuses to add someone frogly doesn't know.
var ErrUnknownPerson = repositories.Invalid("nobody with that ID uses frogly")

// searchLimit keeps a search to a list someone reads at a glance, and
// stops a short text from listing everyone who uses frogly at once.
const searchLimit = 10

// maxSearch is the longest an email can be, so a longer text can't match
// anything, and the database never gets a pattern that large.
const maxSearch = 320

// Create makes a project called name with creator as its only member. A name
// or visibility frogly can't use gives a *repositories.InvalidError.
func Create(ctx context.Context, unit db.Context, creator *repositories.User, name string, visibility repositories.Visibility) (*repositories.Project, error) {
	project, err := repositories.NewProject(name, visibility)
	if err != nil {
		return nil, err
	}

	membership, err := repositories.NewMembership(project.ID(), creator.ID())
	if err != nil {
		return nil, fmt.Errorf("making %s a member: %w", creator.ID(), err)
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

// ChangeVisibility lets a member make the project public or private. It
// gives ErrNotFound to someone who isn't a member, like Get.
func ChangeVisibility(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, visibility repositories.Visibility) (*repositories.Project, error) {
	project, err := Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	err = project.ChangeVisibility(visibility)
	if err != nil {
		return nil, err
	}

	unit.Projects().Update(project)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saving the visibility of project %s: %w", projectID, err)
	}

	return project, nil
}

// ListPublicFor lists the public projects user isn't a member of.
func ListPublicFor(ctx context.Context, unit db.Context, user *repositories.User) ([]*repositories.Project, error) {
	return unit.Projects().ListPublicForNonMember(ctx, user.ID())
}

// FindPeople lets a member find the people who could join the project, by
// part of their name or email. Someone who isn't a member gets
// ErrNotFound, like Get.
func FindPeople(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, text string) ([]*repositories.User, error) {
	_, err := Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	text = strings.TrimSpace(text)

	switch {
	case text == "":
		return nil, repositories.Invalid("type part of a name or an email")

	case utf8.RuneCountInString(text) > maxSearch:
		return nil, repositories.Invalid("a search has at most %d characters", maxSearch)
	}

	return unit.Users().SearchNonMembers(ctx, projectID, text, searchLimit)
}

// AddMember lets a member add someone who has logged in to frogly, and
// gives them back. Adding a member again changes nothing. When two members
// add the same person at once, one of them gets
// repositories.ErrConcurrentModification, and trying again works. Someone
// who isn't a member gets ErrNotFound, like Get.
func AddMember(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, userID uuid.UUID) (*repositories.User, error) {
	_, err := Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	added, err := unit.Users().ByID(ctx, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrUnknownPerson
	}

	if err != nil {
		return nil, err
	}

	_, err = unit.Projects().ByIDForMember(ctx, projectID, userID)
	if err == nil {
		return added, nil
	}

	if !errors.Is(err, repositories.ErrNotFound) {
		return nil, err
	}

	membership, err := repositories.NewMembership(projectID, userID)
	if err != nil {
		return nil, err
	}

	unit.Memberships().Insert(membership)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("adding user %s to project %s: %w", userID, projectID, err)
	}

	return added, nil
}
