package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Identity is a login that belongs to a user: an issuer and the subject
// it knows the person by. A subject is only unique within its issuer, so
// the two together identify a person. One user can have several
// identities, for example one per issuer.
type Identity struct {
	Base

	userID  uuid.UUID
	issuer  string
	subject string
}

// NewIdentity creates an identity that was never saved.
func NewIdentity(userID uuid.UUID, issuer string, subject string) (*Identity, error) {
	if userID == uuid.Nil || issuer == "" || subject == "" {
		return nil, fmt.Errorf("%w: an identity needs a user, an issuer and a subject", ErrInvalid)
	}

	return &Identity{
		Base:    newBase(),
		userID:  userID,
		issuer:  issuer,
		subject: subject,
	}, nil
}

// NewIdentityFromDB rebuilds an identity as loaded.
func NewIdentityFromDB(base Base, userID uuid.UUID, issuer string, subject string) *Identity {
	return &Identity{Base: base, userID: userID, issuer: issuer, subject: subject}
}

// UserID is the user the identity belongs to.
func (identity *Identity) UserID() uuid.UUID {
	return identity.userID
}

// Issuer is who vouches for the identity.
func (identity *Identity) Issuer() string {
	return identity.issuer
}

// Subject is what the issuer calls the person.
func (identity *Identity) Subject() string {
	return identity.subject
}

// IdentityRepository loads and saves identities. An identity never
// changes once saved.
type IdentityRepository interface {
	FirstOrNil(ctx context.Context, issuer string, subject string) (*Identity, error)
	Insert(identity *Identity)
}
