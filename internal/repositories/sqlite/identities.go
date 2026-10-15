package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// IdentityRepository loads identities from SQLite.
type IdentityRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewIdentityRepository creates the repository.
func NewIdentityRepository(reader Reader, tracker *change.Tracker) *IdentityRepository {
	return &IdentityRepository{reader: reader, tracker: tracker}
}

// ByLogin loads the identity with the issuer and subject, or nil when
// nobody has that login.
func (repository *IdentityRepository) ByLogin(ctx context.Context, issuer string, subject string) (*repositories.Identity, error) {
	var (
		id      uuid.UUID
		version int
		userID  uuid.UUID
	)

	selection := sqlbuilder.Select("id", "version", "user_id").From("identities")
	selection.Where(selection.Equal("issuer", issuer), selection.Equal("subject", subject))

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	err := repository.reader.QueryRowContext(ctx, query, args...).Scan(&id, &version, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("loading the identity of %s at %s: %w", subject, issuer, err)
	}

	return repositories.NewIdentityFromDB(repositories.NewBaseFromDB(id, version), userID, issuer, subject), nil
}

// Insert records the identity to be written.
func (repository *IdentityRepository) Insert(identity *repositories.Identity) {
	repository.tracker.Add(change.NewEntry(change.Added, identity))
}

// ExecuteInsert writes the identity as version 1.
func (repository *IdentityRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, identity *repositories.Identity) error {
	insert := sqlbuilder.InsertInto("identities").
		Cols("id", "version", "user_id", "issuer", "subject").
		Values(identity.ID().String(), 1, identity.UserID().String(), identity.Issuer(), identity.Subject())

	query, args := insert.BuildWithFlavor(sqlbuilder.SQLite)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting identity "+identity.ID().String())
}
