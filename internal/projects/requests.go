package projects

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

// ErrNoRequest means the person has no waiting request for the project.
// It wraps ErrNotFound, so it answers like a project nobody may see.
var ErrNoRequest = fmt.Errorf("request not found: %w", ErrNotFound)

// AskToJoin lets user ask to join a public project that takes requests,
// and says whether a request of theirs waits now. Asking again changes
// nothing, and a member asking has nothing to wait for, including someone
// who joined a moment before. A project that doesn't exist or doesn't
// take requests gives ErrNotFound, decided by the database as it saves
// the request.
func AskToJoin(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) (bool, error) {
	_, err := unit.Projects().ByIDForMember(ctx, projectID, user.ID())
	if err == nil {
		return false, nil
	}

	if !errors.Is(err, repositories.ErrNotFound) {
		return false, err
	}

	request, err := repositories.NewJoinRequest(projectID, user.ID())
	if err != nil {
		return false, err
	}

	unit.JoinRequests().Insert(request)

	err = unit.SaveChanges(ctx)

	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return askedAsMember(ctx, unit, user, projectID)

	// a request of theirs exists already, which the unique pair refuses
	// like a race, and they asked either way
	case err != nil && !errors.Is(err, repositories.ErrConcurrentModification):
		return false, fmt.Errorf("saving the request of user %s for project %s: %w", user.ID(), projectID, err)
	}

	return true, nil
}

// askedAsMember answers a request the database refused. The user may have
// got in a moment before, and then has nothing to wait for. Otherwise
// the project doesn't take requests.
func askedAsMember(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) (bool, error) {
	_, err := unit.Projects().ByIDForMember(ctx, projectID, user.ID())
	if errors.Is(err, repositories.ErrNotFound) {
		return false, fmt.Errorf("project %s to ask: %w", projectID, ErrNotFound)
	}

	return false, err
}

// Requests lists the project's waiting requests for one of its members,
// the oldest first. Someone who isn't a member gets ErrNotFound, like Get.
func Requests(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID) ([]repositories.WaitingRequest, error) {
	_, err := Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	return unit.JoinRequests().ListForProject(ctx, projectID)
}

// dropRequest queues the removal of userID's request, if one waits. A
// member has nothing to wait for, however they got in.
func dropRequest(ctx context.Context, unit db.Context, projectID uuid.UUID, userID uuid.UUID) error {
	request, err := unit.JoinRequests().ByProjectAndUser(ctx, projectID, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	unit.JoinRequests().Delete(request)

	return nil
}

// AcceptRequest lets a member say yes to a request, which makes the asker
// a member and removes the request in the same save. It gives the asker
// back. Someone who isn't a member gets ErrNotFound, and a request that
// doesn't wait ErrNoRequest. If the request is answered or the asker gets
// in another way at the same moment, AcceptRequest gives
// repositories.ErrConcurrentModification.
func AcceptRequest(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, askerID uuid.UUID) (*repositories.User, error) {
	_, err := waitingRequest(ctx, unit, user, projectID, askerID)
	if err != nil {
		return nil, err
	}

	asker, err := unit.Users().ByID(ctx, askerID)
	if err != nil {
		return nil, err
	}

	err = addMembership(ctx, unit, projectID, askerID)
	if err != nil {
		return nil, err
	}

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("saying yes to user %s for project %s: %w", askerID, projectID, err)
	}

	return asker, nil
}

// DeclineRequest lets a member say no to a request, which removes it, so
// the asker may ask again. Someone who isn't a member gets ErrNotFound,
// and a request that doesn't wait ErrNoRequest. If the request is answered
// or the asker gets in at the same moment, DeclineRequest gives
// repositories.ErrConcurrentModification.
func DeclineRequest(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, askerID uuid.UUID) error {
	request, err := waitingRequest(ctx, unit, user, projectID, askerID)
	if err != nil {
		return err
	}

	unit.JoinRequests().Delete(request)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("saying no to user %s for project %s: %w", askerID, projectID, err)
	}

	return nil
}

// waitingRequest checks that user is a member before it looks for the
// request, so someone outside learns nothing about who asked.
func waitingRequest(ctx context.Context, unit db.Context, user *repositories.User, projectID uuid.UUID, askerID uuid.UUID) (*repositories.JoinRequest, error) {
	_, err := Get(ctx, unit, user, projectID)
	if err != nil {
		return nil, err
	}

	request, err := unit.JoinRequests().ByProjectAndUser(ctx, projectID, askerID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("request of user %s for project %s: %w", askerID, projectID, ErrNoRequest)
	}

	return request, err
}
