package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// MeTooRepository saves me toos to Postgres.
type MeTooRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewMeTooRepository creates the repository.
func NewMeTooRepository(reader Reader, tracker *change.Tracker) *MeTooRepository {
	return &MeTooRepository{reader: reader, tracker: tracker}
}

// Said says whether the user already said me too to the idea.
func (repository *MeTooRepository) Said(ctx context.Context, ideaID uuid.UUID, userID uuid.UUID) (bool, error) {
	selection := sqlbuilder.Select("count(*)").From("me_toos")
	selection.Where(selection.Equal("idea_id", ideaID), selection.Equal("user_id", userID))

	query, args := selection.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var count int

	err := repository.reader.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("looking up a me too of user %s for idea %s: %w", userID, ideaID, err)
	}

	return count > 0, nil
}

// Insert records the me too to be written.
func (repository *MeTooRepository) Insert(meToo *repositories.MeToo) {
	repository.tracker.Add(change.NewEntry(change.Added, meToo))
}

// ExecuteInsert writes the me too as version 1.
func (repository *MeTooRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, meToo *repositories.MeToo) error {
	insert := sqlbuilder.InsertInto("me_toos").
		Cols("id", "version", "idea_id", "user_id").
		Values(meToo.ID(), 1, meToo.IdeaID(), meToo.UserID())

	query, args := insert.BuildWithFlavor(sqlbuilder.PostgreSQL)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting me too "+meToo.ID().String())
}
