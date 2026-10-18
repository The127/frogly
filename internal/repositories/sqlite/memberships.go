package sqlite

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// MembershipRepository saves memberships to SQLite.
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

// Delete records the user's membership of the project to be removed.
func (repository *MembershipRepository) Delete(projectID uuid.UUID, userID uuid.UUID) {
	repository.tracker.Add(change.NewEntry(change.Deleted, membershipDeletion{projectID: projectID, userID: userID}))
}

// ExecuteInsert writes the membership as version 1. A joining membership
// is only written while its project is open to join, checked in the same
// statement, and gives repositories.ErrNotFound otherwise.
func (repository *MembershipRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, membership *repositories.Membership) error {
	what := "inserting membership " + membership.ID().String()

	if membership.OnlyIfOpen() {
		joining := sqlbuilder.InsertInto("memberships").Cols("id", "version", "project_id", "user_id")

		// the row comes from the project only while it's open, so a closed,
		// private or missing project writes nothing
		project := joining.Select()
		project.Select(project.Var(membership.ID().String()), "1", project.Var(membership.ProjectID().String()), project.Var(membership.UserID().String()))
		project.From("projects")
		project.Where(
			project.Equal("id", membership.ProjectID().String()),
			// 'public' and 'open' are fixed filters, not input, so writing
			// them in can't change what the statement does
			"visibility = 'public'",
			"joining = 'open'",
		)

		query, args := joining.BuildWithFlavor(sqlbuilder.SQLite)

		result, err := tx.ExecContext(ctx, query, args...)

		err = written(err, what)
		if err != nil {
			return err
		}

		return requireRow(result, "an open project "+membership.ProjectID().String()+" to join")
	}

	insert := sqlbuilder.InsertInto("memberships").
		Cols("id", "version", "project_id", "user_id").
		Values(membership.ID().String(), 1, membership.ProjectID().String(), membership.UserID().String())

	query, args := insert.BuildWithFlavor(sqlbuilder.SQLite)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, what)
}

// membershipDeletion removes one user's membership of one project, if
// there is one.
type membershipDeletion struct {
	projectID uuid.UUID
	userID    uuid.UUID
}

// Execute removes the membership.
func (deletion membershipDeletion) Execute(ctx context.Context, tx *sql.Tx) error {
	statement := sqlbuilder.DeleteFrom("memberships")
	statement.Where(statement.Equal("project_id", deletion.projectID.String()), statement.Equal("user_id", deletion.userID.String()))

	query, args := statement.BuildWithFlavor(sqlbuilder.SQLite)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "deleting the membership of user "+deletion.userID.String()+" in project "+deletion.projectID.String())
}
