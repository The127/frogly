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

// PublicProject is a public project as someone outside it sees it.
type PublicProject struct {
	Project *repositories.Project
	// Asked says whether they asked to join and wait for an answer.
	Asked bool
}

// searchLimit keeps a search to a list someone reads at a glance, and
// stops a short text from listing everyone who uses frogly at once.
const searchLimit = 10

// maxSearch is the longest an email can be, so a longer text can't match
// anything, and the database never gets a pattern that large.
const maxSearch = 320

// Create makes a project called name with creator as its only member. A
// name, visibility or way to join frogly can't use gives a
// *repositories.InvalidError.
func Create(ctx context.Context, unit db.Context, creator *repositories.User, name string, visibility repositories.Visibility, joining repositories.Joining) (*repositories.Project, error) {
	project, err := repositories.NewProject(name, visibility, joining)
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

// ListPublicFor lists the public projects user isn't a member of, each
// with whether they asked to join it.
func ListPublicFor(ctx context.Context, unit db.Context, user *repositories.User) ([]PublicProject, error) {
	found, err := unit.Projects().ListPublicForNonMember(ctx, user.ID())
	if err != nil {
		return nil, err
	}

	asked, err := unit.JoinRequests().ProjectsAskedBy(ctx, user.ID())
	if err != nil {
		return nil, err
	}

	askedSet := make(map[uuid.UUID]bool, len(asked))
	for _, projectID := range asked {
		askedSet[projectID] = true
	}

	listed := make([]PublicProject, 0, len(found))
	for _, project := range found {
		listed = append(listed, PublicProject{Project: project, Asked: askedSet[project.ID()]})
	}

	return listed, nil
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
// gives them back. Adding a member again changes nothing. Someone who
// isn't a member gets ErrNotFound, like Get.
//
// If two members add the same person at once, or a member adds someone
// while another answers that person's request, one of them gets
// repositories.ErrConcurrentModification. Trying again works.
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

	err = addMembership(ctx, unit, projectID, userID)
	if err != nil {
		return nil, err
	}

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("adding user %s to project %s: %w", userID, projectID, err)
	}

	return added, nil
}

// Join makes user a member of a public project that is open, and gives the
// project back. Joining a project you're a member of changes nothing, and
// so does joining twice at the same moment. A project that doesn't exist,
// isn't public or isn't open gives ErrNotFound, so nobody learns which
// closed or private projects exist. The database decides that as it saves
// the membership, so a member closing the project at the same moment
// can't be outrun.
//
// If a member says no to the user's waiting request while they join, Join
// gives repositories.ErrConcurrentModification. Trying again works. A yes
// at the same moment lets the join through.
func Join(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) (*repositories.Project, error) {
	member, err := unit.Projects().ByIDForMember(ctx, projectID, user.ID())
	if !errors.Is(err, repositories.ErrNotFound) {
		return member, err
	}

	err = dropRequest(ctx, unit, projectID, user.ID())
	if err != nil {
		return nil, err
	}

	membership, err := repositories.NewJoiningMembership(projectID, user.ID())
	if err != nil {
		return nil, err
	}

	unit.Memberships().Insert(membership)

	err = unit.SaveChanges(ctx)

	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return nil, fmt.Errorf("project %s to join: %w", projectID, ErrNotFound)

	case err != nil && !errors.Is(err, repositories.ErrConcurrentModification):
		return nil, fmt.Errorf("adding user %s to project %s: %w", user.ID(), projectID, err)
	}

	// a race is the same person getting in from elsewhere, a yes to their
	// request included, and then they're a member, or a member saying no
	// to it in between, and then they aren't and may try again
	joined, err := unit.Projects().ByIDForMember(ctx, projectID, user.ID())
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("joining project %s: %w", projectID, repositories.ErrConcurrentModification)
	}

	return joined, err
}

// ChangeJoining lets a member open a public project to join or close it.
// It gives ErrNotFound to someone who isn't a member, like Get.
func ChangeJoining(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, joining repositories.Joining) (*repositories.Project, error) {
	project, err := Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	err = project.ChangeJoining(joining)
	if err != nil {
		return nil, err
	}

	unit.Projects().Update(project)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saving how people join project %s: %w", projectID, err)
	}

	return project, nil
}

// addMembership queues a membership for userID, unless they already are a
// member, and removes their request.
func addMembership(ctx context.Context, unit db.Context, projectID uuid.UUID, userID uuid.UUID) error {
	err := dropRequest(ctx, unit, projectID, userID)
	if err != nil {
		return err
	}

	_, err = unit.Projects().ByIDForMember(ctx, projectID, userID)
	if err == nil {
		return nil
	}

	if !errors.Is(err, repositories.ErrNotFound) {
		return err
	}

	membership, err := repositories.NewMembership(projectID, userID)
	if err != nil {
		return err
	}

	unit.Memberships().Insert(membership)

	return nil
}
