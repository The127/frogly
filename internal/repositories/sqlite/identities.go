package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

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

// FirstOrNil loads the identity with the issuer and subject, or nil.
func (repository *IdentityRepository) FirstOrNil(ctx context.Context, issuer string, subject string) (*repositories.Identity, error) {
	var (
		id      uuid.UUID
		version int
		userID  uuid.UUID
	)

	err := repository.reader.QueryRowContext(ctx,
		"select id, version, user_id from identities where issuer = ? and subject = ?", issuer, subject,
	).Scan(&id, &version, &userID)
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
	_, err := tx.ExecContext(ctx,
		"insert into identities (id, version, user_id, issuer, subject) values (?, 1, ?, ?, ?)",
		identity.ID().String(), identity.UserID().String(), identity.Issuer(), identity.Subject(),
	)

	return written(err, "inserting identity "+identity.ID().String())
}
