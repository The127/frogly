package repositories

import (
	"github.com/google/uuid"
)

// Membership puts a user into a project. All members of a project are
// equal. A user is a member of a project at most once.
type Membership struct {
	Base

	projectID uuid.UUID
	userID    uuid.UUID
}

// NewMembership creates a membership that was never saved.
func NewMembership(projectID uuid.UUID, userID uuid.UUID) (*Membership, error) {
	if projectID == uuid.Nil || userID == uuid.Nil {
		return nil, invalid("a membership needs a project and a user")
	}

	return &Membership{
		Base:      newBase(),
		projectID: projectID,
		userID:    userID,
	}, nil
}

// ProjectID is the project the user is a member of.
func (membership *Membership) ProjectID() uuid.UUID {
	return membership.projectID
}

// UserID is the member.
func (membership *Membership) UserID() uuid.UUID {
	return membership.userID
}

// MembershipRepository saves memberships.
type MembershipRepository interface {
	Insert(membership *Membership)
}
