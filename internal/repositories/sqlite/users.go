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

// UserRepository loads users from SQLite.
type UserRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewUserRepository creates the repository.
func NewUserRepository(reader Reader, tracker *change.Tracker) *UserRepository {
	return &UserRepository{reader: reader, tracker: tracker}
}

// First loads the user with the ID, or returns repositories.ErrNotFound.
func (repository *UserRepository) First(ctx context.Context, id uuid.UUID) (*repositories.User, error) {
	var (
		rowID   uuid.UUID
		version int
		name    string
		email   string
	)

	err := repository.reader.QueryRowContext(ctx,
		"select id, version, name, email from users where id = ?", id.String(),
	).Scan(&rowID, &version, &name, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("user %s: %w", id, repositories.ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("loading user %s: %w", id, err)
	}

	return repositories.NewUserFromDB(repositories.NewBaseFromDB(rowID, version), name, email), nil
}

// Insert records the user to be written.
func (repository *UserRepository) Insert(user *repositories.User) {
	repository.tracker.Add(change.NewEntry(change.Added, user))
}

// Update records the user to be written again.
func (repository *UserRepository) Update(user *repositories.User) {
	repository.tracker.Add(change.NewEntry(change.Updated, user))
}

// ExecuteInsert writes the user as version 1.
func (repository *UserRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, user *repositories.User) error {
	_, err := tx.ExecContext(ctx,
		"insert into users (id, version, name, email) values (?, 1, ?, ?)",
		user.ID().String(), user.Name(), user.Email(),
	)

	return written(err, "inserting user "+user.ID().String())
}

// ExecuteUpdate writes the whole user as the version after the one it was
// loaded at. The schema refuses it when the row changed since.
func (repository *UserRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, user *repositories.User) error {
	_, err := tx.ExecContext(ctx,
		"update users set version = ?, name = ?, email = ? where id = ?",
		user.Version()+1, user.Name(), user.Email(), user.ID().String(),
	)

	return written(err, "updating user "+user.ID().String())
}
