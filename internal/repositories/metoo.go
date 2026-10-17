package repositories

import (
	"github.com/google/uuid"
)

// MeToo says that a user wants an idea someone else planted. A user is
// behind an idea at most once, and the planter always is.
type MeToo struct {
	Base

	ideaID uuid.UUID
	userID uuid.UUID
}

// NewMeToo creates a me too that was never saved.
func NewMeToo(ideaID uuid.UUID, userID uuid.UUID) (*MeToo, error) {
	if ideaID == uuid.Nil || userID == uuid.Nil {
		return nil, Invalid("a me too needs an idea and a user")
	}

	return &MeToo{Base: newBase(), ideaID: ideaID, userID: userID}, nil
}

// IdeaID is the idea the user wants.
func (meToo *MeToo) IdeaID() uuid.UUID {
	return meToo.ideaID
}

// UserID is who wants it.
func (meToo *MeToo) UserID() uuid.UUID {
	return meToo.userID
}
