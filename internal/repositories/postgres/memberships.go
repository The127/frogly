package postgres

import (
	"context"
	"database/sql"

	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// MembershipRepository saves memberships to Postgres.
type MembershipRepository struct {
	tracker *change.Tracker
}

// NewMembershipRepository creates the repository.
func NewMembershipRepository(tracker *change.Tracker) *MembershipRepository {
	return &MembershipRepository{tracker: tracker}
}

// Insert records the membership to be written.
func (repository *MembershipRepository) Insert(membership *repositories.Membership) {
	repository.tracker.Add(change.NewEntry(change.Added, membership))
}

// ExecuteInsert writes the membership as version 1. A joining membership
// is only written while its project is open to join, checked in the same
// statement, and gives repositories.ErrNotFound otherwise.
func (repository *MembershipRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, membership *repositories.Membership) error {
	what := "inserting membership " + membership.ID().String()

	if membership.OnlyIfOpen() {
		// constants, not parameters, like the list of public projects
		joining := sqlbuilder.Build(
			"insert into memberships (id, version, project_id, user_id) "+
				"select $?, 1, $?, $? from projects "+
				"where id = $? and visibility = 'public' and joining = 'open'",
			membership.ID(), membership.ProjectID(), membership.UserID(), membership.ProjectID(),
		)

		query, args := joining.BuildWithFlavor(sqlbuilder.PostgreSQL)

		result, err := tx.ExecContext(ctx, query, args...)

		err = written(err, what)
		if err != nil {
			return err
		}

		return requireRow(result, "an open project "+membership.ProjectID().String()+" to join")
	}

	insert := sqlbuilder.InsertInto("memberships").
		Cols("id", "version", "project_id", "user_id").
		Values(membership.ID(), 1, membership.ProjectID(), membership.UserID())

	query, args := insert.BuildWithFlavor(sqlbuilder.PostgreSQL)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, what)
}
