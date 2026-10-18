package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JoinRequest says that a user asked to join a public project that takes
// requests. A member says yes or no, and either way the request is gone.
// A user has at most one request per project.
type JoinRequest struct {
	Base

	projectID uuid.UUID
	userID    uuid.UUID
	askedAt   time.Time
}

// NewJoinRequest creates a request that was never saved. The database only
// saves it while the project takes requests, and refuses it with
// ErrNotFound otherwise, like a joining membership.
func NewJoinRequest(projectID uuid.UUID, userID uuid.UUID) (*JoinRequest, error) {
	if projectID == uuid.Nil || userID == uuid.Nil {
		return nil, Invalid("a request to join needs a project and a user")
	}

	return &JoinRequest{
		Base:      newBase(),
		projectID: projectID,
		userID:    userID,
		// both databases keep microseconds, like an idea's planting time
		askedAt: time.Now().UTC().Truncate(time.Microsecond),
	}, nil
}

// NewJoinRequestFromDB rebuilds a request as loaded.
func NewJoinRequestFromDB(base Base, projectID uuid.UUID, userID uuid.UUID, askedAt time.Time) *JoinRequest {
	return &JoinRequest{Base: base, projectID: projectID, userID: userID, askedAt: askedAt}
}

// ProjectID is the project the user asked to join.
func (request *JoinRequest) ProjectID() uuid.UUID {
	return request.projectID
}

// UserID is who asked.
func (request *JoinRequest) UserID() uuid.UUID {
	return request.userID
}

// AskedAt is when they asked.
func (request *JoinRequest) AskedAt() time.Time {
	return request.askedAt
}

// WaitingRequest is a request with who asked, as members see it.
type WaitingRequest struct {
	Request *JoinRequest
	Asker   *User
}

// JoinRequestRepository loads and saves requests to join.
type JoinRequestRepository interface {
	// ByProjectAndUser returns ErrNotFound when the user has no request
	// for the project.
	ByProjectAndUser(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*JoinRequest, error)
	// ListForProject lists the project's waiting requests, the oldest
	// first, then by ID.
	ListForProject(ctx context.Context, projectID uuid.UUID) ([]WaitingRequest, error)
	Insert(request *JoinRequest)
	Delete(request *JoinRequest)
	// DeleteForProject removes every waiting request of the project, for
	// a project that doesn't take requests any more.
	DeleteForProject(projectID uuid.UUID)
}
