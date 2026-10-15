package sqlite

import (
	"context"
	"database/sql"

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

// ExecuteInsert writes the membership as version 1.
func (repository *MembershipRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, membership *repositories.Membership) error {
	_, err := tx.ExecContext(ctx,
		"insert into memberships (id, version, project_id, user_id, role) values (?, 1, ?, ?, ?)",
		membership.ID().String(), membership.ProjectID().String(), membership.UserID().String(), string(membership.Role()),
	)

	return written(err, "inserting membership "+membership.ID().String())
}
