// Package db is frogly's database as a unit of work: one context per
// request, which reads right away and writes everything at once.
package db

import (
	"context"

	"github.com/the127/frogly/internal/repositories"
)

// Database is one database, migrated once at start, handing out a context
// per unit of work.
type Database interface {
	Migrate(ctx context.Context) error
	NewContext() Context
	Close() error
}

// Context is one unit of work. Reads through its repositories go to the
// database right away and don't see writes recorded in the same context.
// SaveChanges writes every recorded change in one transaction, all or
// nothing, and forgets the changes either way. A caller who wants to try
// again records them again. It returns
// repositories.ErrConcurrentModification when a change would overwrite
// someone else's, like an update to a row that changed since it was
// loaded, or a second identity with the same issuer and subject.
type Context interface {
	Users() repositories.UserRepository
	Identities() repositories.IdentityRepository
	Projects() repositories.ProjectRepository
	Memberships() repositories.MembershipRepository
	Ideas() repositories.IdeaRepository
	MeToos() repositories.MeTooRepository
	SaveChanges(ctx context.Context) error
}
