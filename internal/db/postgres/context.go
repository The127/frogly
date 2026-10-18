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
	ideas       *repos.IdeaRepository
	meToos      *repos.MeTooRepository
	requests    *repos.JoinRequestRepository
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
		ideas:       repos.NewIdeaRepository(connection, tracker),
		meToos:      repos.NewMeTooRepository(connection, tracker),
		requests:    repos.NewJoinRequestRepository(connection, tracker),
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

// Ideas is the unit of work's idea repository.
func (unit *Context) Ideas() repositories.IdeaRepository {
	return unit.ideas
}

// MeToos is the unit of work's me too repository.
func (unit *Context) MeToos() repositories.MeTooRepository {
	return unit.meToos
}

// JoinRequests is the unit of work's repository of requests to join.
func (unit *Context) JoinRequests() repositories.JoinRequestRepository {
	return unit.requests
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
	switch entry.Type() {
	case change.Added:
		return unit.insert(ctx, tx, entry.Item())
	case change.Updated:
		return unit.update(ctx, tx, entry.Item())
	case change.Deleted:
		return unit.delete(ctx, tx, entry.Item())
	default:
		return fmt.Errorf("%w: change type %d", errUnsupportedChange, entry.Type())
	}
}

func (unit *Context) insert(ctx context.Context, tx *sql.Tx, item any) error {
	switch item := item.(type) {
	case *repositories.User:
		return unit.users.ExecuteInsert(ctx, tx, item)
	case *repositories.Identity:
		return unit.identities.ExecuteInsert(ctx, tx, item)
	case *repositories.Project:
		return unit.projects.ExecuteInsert(ctx, tx, item)
	case *repositories.Membership:
		return unit.memberships.ExecuteInsert(ctx, tx, item)
	case *repositories.Idea:
		return unit.ideas.ExecuteInsert(ctx, tx, item)
	case *repositories.MeToo:
		return unit.meToos.ExecuteInsert(ctx, tx, item)
	case *repositories.JoinRequest:
		return unit.requests.ExecuteInsert(ctx, tx, item)
	default:
		return fmt.Errorf("%w: inserting %T", errUnsupportedChange, item)
	}
}

func (unit *Context) update(ctx context.Context, tx *sql.Tx, item any) error {
	switch item := item.(type) {
	case *repositories.User:
		return unit.users.ExecuteUpdate(ctx, tx, item)
	case *repositories.Project:
		return unit.projects.ExecuteUpdate(ctx, tx, item)
	default:
		return fmt.Errorf("%w: updating %T", errUnsupportedChange, item)
	}
}

func (unit *Context) delete(ctx context.Context, tx *sql.Tx, item any) error {
	switch item := item.(type) {
	case *repositories.JoinRequest:
		return unit.requests.ExecuteDelete(ctx, tx, item)
	case repositories.ProjectRequests:
		return unit.requests.ExecuteDeleteForProject(ctx, tx, item)
	default:
		return fmt.Errorf("%w: deleting %T", errUnsupportedChange, item)
	}
}
