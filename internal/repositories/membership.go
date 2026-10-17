package repositories

import (
	"github.com/google/uuid"
)

// Membership puts a user into a project. All members of a project are
// equal. A user is a member of a project at most once.
type Membership struct {
	Base

	projectID  uuid.UUID
	userID     uuid.UUID
	onlyIfOpen bool
}

// NewMembership creates a membership that was never saved.
func NewMembership(projectID uuid.UUID, userID uuid.UUID) (*Membership, error) {
	if projectID == uuid.Nil || userID == uuid.Nil {
		return nil, Invalid("a membership needs a project and a user")
	}

	return &Membership{
		Base:      newBase(),
		projectID: projectID,
		userID:    userID,
	}, nil
}

// NewJoiningMembership creates the membership of someone who joins a
// project themselves. The database only saves it while the project is
// public and open, and refuses it with ErrNotFound otherwise, so a member
// closing the project at the same moment can't be outrun.
func NewJoiningMembership(projectID uuid.UUID, userID uuid.UUID) (*Membership, error) {
	membership, err := NewMembership(projectID, userID)
	if err != nil {
		return nil, err
	}

	membership.onlyIfOpen = true

	return membership, nil
}

// OnlyIfOpen says whether the membership needs the project to be open to
// join when it's saved.
func (membership *Membership) OnlyIfOpen() bool {
	return membership.onlyIfOpen
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
