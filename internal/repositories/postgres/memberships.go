package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
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

// Delete records the user's membership of the project to be removed.
func (repository *MembershipRepository) Delete(projectID uuid.UUID, userID uuid.UUID) {
	repository.tracker.Add(change.NewEntry(change.Deleted, repositories.MembershipOf{ProjectID: projectID, UserID: userID}))
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
		project.Select(project.Var(membership.ID()), "1", project.Var(membership.ProjectID()), project.Var(membership.UserID()))
		project.From("projects")
		project.Where(
			project.Equal("id", membership.ProjectID()),
			// 'public' and 'open' are fixed filters, not input, so writing
			// them in can't change what the statement does
			"visibility = 'public'",
			"joining = 'open'",
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

// ExecuteDelete removes the membership, if there is one.
func (repository *MembershipRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, membership repositories.MembershipOf) error {
	deletion := sqlbuilder.DeleteFrom("memberships")
	deletion.Where(deletion.Equal("project_id", membership.ProjectID), deletion.Equal("user_id", membership.UserID))

	query, args := deletion.BuildWithFlavor(sqlbuilder.PostgreSQL)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "deleting the membership of user "+membership.UserID.String()+" in project "+membership.ProjectID.String())
}
