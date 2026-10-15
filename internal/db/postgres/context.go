package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
	repos "github.com/the127/frogly/internal/repositories/postgres"
)

// Context is one unit of work on a Postgres database.
type Context struct {
	connection  *sql.DB
	tracker     *change.Tracker
	users       *repos.UserRepository
	identities  *repos.IdentityRepository
	projects    *repos.ProjectRepository
	memberships *repos.MembershipRepository
}

func newContext(connection *sql.DB) *Context {
	tracker := change.NewTracker()

	return &Context{
		connection:  connection,
		tracker:     tracker,
		users:       repos.NewUserRepository(connection, tracker),
		identities:  repos.NewIdentityRepository(connection, tracker),
		projects:    repos.NewProjectRepository(connection, tracker),
		memberships: repos.NewMembershipRepository(tracker),
	}
}

// Users is the unit of work's user repository.
func (unit *Context) Users() repositories.UserRepository {
	return unit.users
}

// Identities is the unit of work's identity repository.
func (unit *Context) Identities() repositories.IdentityRepository {
	return unit.identities
}

// Projects is the unit of work's project repository.
func (unit *Context) Projects() repositories.ProjectRepository {
	return unit.projects
}

// Memberships is the unit of work's membership repository.
func (unit *Context) Memberships() repositories.MembershipRepository {
	return unit.memberships
}

// SaveChanges writes every recorded change in one transaction, in the
// order they were recorded.
func (unit *Context) SaveChanges(ctx context.Context) error {
	changes := unit.tracker.Changes()
	if len(changes) == 0 {
		return nil
	}

	// written or rolled back, the changes are done with. Keeping them would
	// try failed writes again on the next save, which a lost race fails
	// again.
	defer unit.tracker.Clear()

	tx, err := unit.connection.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning a transaction: %w", err)
	}

	defer func() {
		// after a commit there's nothing left to roll back
		_ = tx.Rollback()
	}()

	for _, entry := range changes {
		err = unit.apply(ctx, tx, entry)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	return nil
}

var errUnsupportedChange = errors.New("unsupported change")

func (unit *Context) apply(ctx context.Context, tx *sql.Tx, entry *change.Entry) error {
	switch item := entry.Item().(type) {
	case *repositories.User:
		if entry.Type() == change.Updated {
			return unit.users.ExecuteUpdate(ctx, tx, item)
		}

		return unit.users.ExecuteInsert(ctx, tx, item)

	case *repositories.Identity:
		return unit.identities.ExecuteInsert(ctx, tx, item)

	case *repositories.Project:
		return unit.projects.ExecuteInsert(ctx, tx, item)

	case *repositories.Membership:
		return unit.memberships.ExecuteInsert(ctx, tx, item)

	default:
		return fmt.Errorf("%w: %T", errUnsupportedChange, entry.Item())
	}
}
