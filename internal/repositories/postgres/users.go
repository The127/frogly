package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// UserRepository loads users from Postgres.
type UserRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewUserRepository creates the repository.
func NewUserRepository(reader Reader, tracker *change.Tracker) *UserRepository {
	return &UserRepository{reader: reader, tracker: tracker}
}

// ByID loads the user with the ID, or returns repositories.ErrNotFound.
func (repository *UserRepository) ByID(ctx context.Context, id uuid.UUID) (*repositories.User, error) {
	var (
		rowID   uuid.UUID
		version int
		name    string
		email   string
	)

	selection := sqlbuilder.Select("id", "version", "name", "email").From("users")
	selection.Where(selection.Equal("id", id))

	query, args := selection.BuildWithFlavor(sqlbuilder.PostgreSQL)

	err := repository.reader.QueryRowContext(ctx, query, args...).Scan(&rowID, &version, &name, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("user %s: %w", id, repositories.ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("loading user %s: %w", id, err)
	}

	return repositories.NewUserFromDB(repositories.NewBaseFromDB(rowID, version), name, email), nil
}

// SearchNonMembers finds the users to add to a project. ilike ignores case.
func (repository *UserRepository) SearchNonMembers(ctx context.Context, projectID uuid.UUID, text string, limit int) ([]*repositories.User, error) {
	membership := sqlbuilder.Select("1").From("memberships m")
	membership.Where("m.user_id = u.id", membership.Equal("m.project_id", projectID))

	pattern := "%" + escapeLike(text) + "%"

	selection := sqlbuilder.Select("u.id", "u.version", "u.name", "u.email").From("users u")
	selection.Where(
		selection.NotExists(membership),
		selection.Or(
			"u.name ilike "+selection.Var(pattern)+` escape '\'`,
			"u.email ilike "+selection.Var(pattern)+` escape '\'`,
		),
	)
	selection.OrderBy(`u.name collate "C"`, "u.id")
	selection.Limit(limit)

	query, args := selection.BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("searching people for project %s: %w", projectID, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var users []*repositories.User

	for rows.Next() {
		var (
			id      uuid.UUID
			version int
			name    string
			email   string
		)

		err := rows.Scan(&id, &version, &name, &email)
		if err != nil {
			return nil, fmt.Errorf("reading a user: %w", err)
		}

		users = append(users, repositories.NewUserFromDB(repositories.NewBaseFromDB(id, version), name, email))
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("searching people for project %s: %w", projectID, err)
	}

	return users, nil
}

// escapeLike makes %, _ and the escape character itself stand for
// themselves, so a search can't match more than it says.
func escapeLike(text string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text)
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
	insert := sqlbuilder.InsertInto("users").
		Cols("id", "version", "name", "email").
		Values(user.ID(), 1, user.Name(), user.Email())

	query, args := insert.BuildWithFlavor(sqlbuilder.PostgreSQL)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting user "+user.ID().String())
}

// ExecuteUpdate writes the whole user as the version after the one it was
// loaded at. The schema refuses it when the row changed since.
func (repository *UserRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, user *repositories.User) error {
	update := sqlbuilder.Update("users")
	update.Set(
		update.Assign("version", user.Version()+1),
		update.Assign("name", user.Name()),
		update.Assign("email", user.Email()),
	)
	update.Where(update.Equal("id", user.ID()))

	query, args := update.BuildWithFlavor(sqlbuilder.PostgreSQL)

	result, err := tx.ExecContext(ctx, query, args...)

	err = written(err, "updating user "+user.ID().String())
	if err != nil {
		return err
	}

	// the version trigger only fires for a row that exists, so a missing
	// row would otherwise pass as written
	return requireRow(result, "user "+user.ID().String())
}
