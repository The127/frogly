// Package people turns the person behind a token into a frogly user.
// There's no sign-up: the first request with a token creates the user.
package people

import (
	"context"
	"errors"
	"fmt"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

// errNoIdentityAfterRace is a lost race whose winner can't be found, which
// only a deleted identity could cause.
var errNoIdentityAfterRace = errors.New("lost the race to create the user, but found no identity")

// Resolve finds the user the person is, by the issuer and subject of their
// login, and stores the name and email from it when they changed. A person
// frogly hasn't seen yet becomes a new user with a new identity.
//
// Resolve saves the unit of work itself, since recovering from a lost race
// needs a save that failed. Give it a unit of work of its own, so it can't
// save or drop what someone else recorded.
func Resolve(ctx context.Context, unit db.Context, person authn.Person) (*repositories.User, error) {
	user, err := existing(ctx, unit, person)
	if err != nil || user != nil {
		return user, err
	}

	user, err = create(ctx, unit, person)
	if !errors.Is(err, repositories.ErrConcurrentModification) {
		return user, err
	}

	// a second request of the same person, like a page loading two things
	// at once, created the user first
	user, err = existing(ctx, unit, person)
	if err == nil && user == nil {
		return nil, fmt.Errorf("%s at %s: %w", person.Subject, person.Issuer, errNoIdentityAfterRace)
	}

	return user, err
}

func existing(ctx context.Context, unit db.Context, person authn.Person) (*repositories.User, error) {
	identity, err := unit.Identities().FirstOrNil(ctx, person.Issuer, person.Subject)
	if err != nil || identity == nil {
		return nil, err
	}

	user, err := unit.Users().First(ctx, identity.UserID())
	if err != nil {
		return nil, fmt.Errorf("the user of %s at %s: %w", person.Subject, person.Issuer, err)
	}

	if !user.Refresh(person.Name, person.Email) {
		return user, nil
	}

	unit.Users().Update(user)

	err = unit.SaveChanges(ctx)
	if errors.Is(err, repositories.ErrConcurrentModification) {
		// another request stored a profile first. Answering with that one
		// keeps the request in line with what's stored.
		return unit.Users().First(ctx, identity.UserID())
	}

	return user, err
}

func create(ctx context.Context, unit db.Context, person authn.Person) (*repositories.User, error) {
	user, err := repositories.NewUserFromLogin(person.Name, person.Subject, person.Email)
	if err != nil {
		return nil, err
	}

	identity, err := repositories.NewIdentity(user.ID(), person.Issuer, person.Subject)
	if err != nil {
		return nil, err
	}

	unit.Users().Insert(user)
	unit.Identities().Insert(identity)

	err = unit.SaveChanges(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}
