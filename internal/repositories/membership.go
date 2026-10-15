package repositories

import (
	"fmt"

	"github.com/google/uuid"
)

// Role is what a member may do in a project.
type Role string

// RoleOwner is the role of whoever created the project.
const RoleOwner Role = "owner"

// Membership puts a user into a project with a role. A user is a member of
// a project at most once.
type Membership struct {
	Base

	projectID uuid.UUID
	userID    uuid.UUID
	role      Role
}

// NewMembership creates a membership that was never saved.
func NewMembership(projectID uuid.UUID, userID uuid.UUID, role Role) (*Membership, error) {
	if role != RoleOwner {
		return nil, fmt.Errorf("%w: %q isn't a role", ErrInvalid, role)
	}

	return &Membership{
		Base:      newBase(),
		projectID: projectID,
		userID:    userID,
		role:      role,
	}, nil
}

// NewMembershipFromDB rebuilds a membership as loaded.
func NewMembershipFromDB(base Base, projectID uuid.UUID, userID uuid.UUID, role Role) *Membership {
	return &Membership{Base: base, projectID: projectID, userID: userID, role: role}
}

// ProjectID is the project the user is a member of.
func (membership *Membership) ProjectID() uuid.UUID {
	return membership.projectID
}

// UserID is the member.
func (membership *Membership) UserID() uuid.UUID {
	return membership.userID
}

// Role is what the member may do.
func (membership *Membership) Role() Role {
	return membership.role
}

// MembershipRepository saves memberships.
type MembershipRepository interface {
	Insert(membership *Membership)
}
