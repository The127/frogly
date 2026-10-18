package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// JoinRequestRepository loads and saves requests to join in SQLite.
type JoinRequestRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewJoinRequestRepository creates the repository.
func NewJoinRequestRepository(reader Reader, tracker *change.Tracker) *JoinRequestRepository {
	return &JoinRequestRepository{reader: reader, tracker: tracker}
}

// ByProjectAndUser loads the user's request for the project, or returns
// repositories.ErrNotFound.
func (repository *JoinRequestRepository) ByProjectAndUser(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*repositories.JoinRequest, error) {
	selection := sqlbuilder.Select("id", "version", "project_id", "user_id", "asked_at").From("join_requests")
	selection.Where(selection.Equal("project_id", projectID.String()), selection.Equal("user_id", userID.String()))

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	var (
		id      uuid.UUID
		version int
		project uuid.UUID
		user    uuid.UUID
		askedAt int64
	)

	err := repository.reader.QueryRowContext(ctx, query, args...).Scan(&id, &version, &project, &user, &askedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("request of user %s for project %s: %w", userID, projectID, repositories.ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("loading the request of user %s for project %s: %w", userID, projectID, err)
	}

	return repositories.NewJoinRequestFromDB(repositories.NewBaseFromDB(id, version), project, user, time.UnixMicro(askedAt).UTC()), nil
}

// ListForProject lists the project's waiting requests with who asked, in
// one query.
func (repository *JoinRequestRepository) ListForProject(ctx context.Context, projectID uuid.UUID) ([]repositories.WaitingRequest, error) {
	selection := sqlbuilder.Select(
		"r.id", "r.version", "r.project_id", "r.user_id", "r.asked_at", "u.version", "u.name", "u.email",
	).From("join_requests r").Join("users u", "u.id = r.user_id")
	selection.Where(selection.Equal("r.project_id", projectID.String()))
	selection.OrderBy("r.asked_at", "r.id")

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing the requests for project %s: %w", projectID, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var waiting []repositories.WaitingRequest

	for rows.Next() {
		var (
			id          uuid.UUID
			version     int
			project     uuid.UUID
			user        uuid.UUID
			askedAt     int64
			userVersion int
			name        string
			email       string
		)

		err := rows.Scan(&id, &version, &project, &user, &askedAt, &userVersion, &name, &email)
		if err != nil {
			return nil, fmt.Errorf("reading a request: %w", err)
		}

		waiting = append(waiting, repositories.WaitingRequest{
			Request: repositories.NewJoinRequestFromDB(repositories.NewBaseFromDB(id, version), project, user, time.UnixMicro(askedAt).UTC()),
			Asker:   repositories.NewUserFromDB(repositories.NewBaseFromDB(user, userVersion), name, email),
		})
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing the requests for project %s: %w", projectID, err)
	}

	return waiting, nil
}

// ProjectsAskedBy gives the projects the user has a waiting request for.
func (repository *JoinRequestRepository) ProjectsAskedBy(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	selection := sqlbuilder.Select("project_id").From("join_requests")
	selection.Where(selection.Equal("user_id", userID.String()))
	selection.OrderBy("project_id")

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing the projects user %s asked: %w", userID, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var asked []uuid.UUID

	for rows.Next() {
		var projectID uuid.UUID

		err := rows.Scan(&projectID)
		if err != nil {
			return nil, fmt.Errorf("reading a project user %s asked: %w", userID, err)
		}

		asked = append(asked, projectID)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing the projects user %s asked: %w", userID, err)
	}

	return asked, nil
}

// Insert records the request to be written.
func (repository *JoinRequestRepository) Insert(request *repositories.JoinRequest) {
	repository.tracker.Add(change.NewEntry(change.Added, request))
}

// Delete records the request to be removed.
func (repository *JoinRequestRepository) Delete(request *repositories.JoinRequest) {
	repository.tracker.Add(change.NewEntry(change.Deleted, request))
}

// ExecuteInsert writes the request as version 1, only while its project
// takes requests and the user isn't a member, checked in the same
// statement. It gives repositories.ErrNotFound otherwise.
func (repository *JoinRequestRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, request *repositories.JoinRequest) error {
	insert := sqlbuilder.InsertInto("join_requests").Cols("id", "version", "project_id", "user_id", "asked_at")

	membership := sqlbuilder.Select("1").From("memberships")
	membership.Where("memberships.project_id = projects.id", membership.Equal("memberships.user_id", request.UserID().String()))

	// the row comes from the project only while it takes requests from
	// the user, so an open, closed, private or missing project writes
	// nothing, and neither does a member
	project := insert.Select()
	project.Select(project.Var(request.ID().String()), "1", project.Var(request.ProjectID().String()), project.Var(request.UserID().String()), project.Var(request.AskedAt().UnixMicro()))
	project.From("projects")
	project.Where(
		project.Equal("id", request.ProjectID().String()),
		// 'public' and 'request' are fixed filters, not input, so writing
		// them in can't change what the statement does
		"visibility = 'public'",
		"joining = 'request'",
		project.NotExists(membership),
	)

	query, args := insert.BuildWithFlavor(sqlbuilder.SQLite)

	result, err := tx.ExecContext(ctx, query, args...)

	err = written(err, "inserting request "+request.ID().String())
	if err != nil {
		return err
	}

	return requireRow(result, "a project "+request.ProjectID().String()+" that takes requests")
}

// ExecuteDelete removes the request it loaded. A request that's gone, or
// was saved again, since means someone else answered it first.
func (repository *JoinRequestRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, request *repositories.JoinRequest) error {
	deletion := sqlbuilder.DeleteFrom("join_requests")
	deletion.Where(deletion.Equal("id", request.ID().String()), deletion.Equal("version", request.Version()))

	query, args := deletion.BuildWithFlavor(sqlbuilder.SQLite)

	result, err := tx.ExecContext(ctx, query, args...)

	err = written(err, "deleting request "+request.ID().String())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("counting the rows deleted for request %s: %w", request.ID(), err)
	}

	if rows == 0 {
		return fmt.Errorf("%w: request %s was answered already", repositories.ErrConcurrentModification, request.ID())
	}

	return nil
}
