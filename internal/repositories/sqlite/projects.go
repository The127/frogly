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

// ProjectRepository loads projects from SQLite.
type ProjectRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewProjectRepository creates the repository.
func NewProjectRepository(reader Reader, tracker *change.Tracker) *ProjectRepository {
	return &ProjectRepository{reader: reader, tracker: tracker}
}

// ListForMember lists the user's projects. SQLite compares text byte by
// byte unless told otherwise, which is the order the interface promises.
func (repository *ProjectRepository) ListForMember(ctx context.Context, userID uuid.UUID) ([]*repositories.Project, error) {
	selection := sqlbuilder.Select("p.id", "p.version", "p.name", "p.visibility", "p.joining").From("projects p")
	selection.Join("memberships m", "m.project_id = p.id")
	selection.Where(selection.Equal("m.user_id", userID.String()))
	selection.OrderBy("p.name", "p.id")

	return repository.list(ctx, selection, "the projects of user "+userID.String())
}

// ListPublicForNonMember lists the public projects the user isn't a
// member of, each with whether they asked to join it, in the order of
// ListForMember.
func (repository *ProjectRepository) ListPublicForNonMember(ctx context.Context, userID uuid.UUID) ([]repositories.PublicProject, error) {
	membership := sqlbuilder.Select("1").From("memberships m")
	membership.Where("m.project_id = p.id", membership.Equal("m.user_id", userID.String()))

	request := sqlbuilder.Select("1").From("join_requests r")
	request.Where("r.project_id = p.id", request.Equal("r.user_id", userID.String()))

	selection := sqlbuilder.Select("p.id", "p.version", "p.name", "p.visibility", "p.joining")
	selection.SelectMore(selection.Exists(request))
	selection.From("projects p")
	// a constant, not a parameter: the database only uses the partial
	// index projects_public when it sees the value in the query
	selection.Where(
		"p.visibility = 'public'",
		selection.NotExists(membership),
	)
	selection.OrderBy("p.name", "p.id")

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing the public projects for user %s: %w", userID, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var listed []repositories.PublicProject

	for rows.Next() {
		var (
			id         uuid.UUID
			version    int
			name       string
			visibility string
			joining    string
			asked      bool
		)

		err := rows.Scan(&id, &version, &name, &visibility, &joining, &asked)
		if err != nil {
			return nil, fmt.Errorf("reading a public project for user %s: %w", userID, err)
		}

		project := repositories.NewProjectFromDB(repositories.NewBaseFromDB(id, version), name, repositories.Visibility(visibility), repositories.Joining(joining))
		listed = append(listed, repositories.PublicProject{Project: project, Asked: asked})
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing the public projects for user %s: %w", userID, err)
	}

	return listed, nil
}

func (repository *ProjectRepository) list(ctx context.Context, selection *sqlbuilder.SelectBuilder, what string) ([]*repositories.Project, error) {
	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", what, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var projects []*repositories.Project

	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", what, err)
	}

	return projects, nil
}

// ByIDForMember loads the project if the user is a member of it, or
// returns repositories.ErrNotFound.
func (repository *ProjectRepository) ByIDForMember(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*repositories.Project, error) {
	selection := sqlbuilder.Select("p.id", "p.version", "p.name", "p.visibility", "p.joining").From("projects p")
	selection.Join("memberships m", "m.project_id = p.id")
	selection.Where(selection.Equal("p.id", projectID.String()), selection.Equal("m.user_id", userID.String()))

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	project, err := scanProject(repository.reader.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("project %s for user %s: %w", projectID, userID, repositories.ErrNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("loading project %s for user %s: %w", projectID, userID, err)
	}

	return project, nil
}

func scanProject(row scanner) (*repositories.Project, error) {
	var (
		id         uuid.UUID
		version    int
		name       string
		visibility string
		joining    string
	)

	err := row.Scan(&id, &version, &name, &visibility, &joining)
	if err != nil {
		return nil, fmt.Errorf("reading a project: %w", err)
	}

	return repositories.NewProjectFromDB(repositories.NewBaseFromDB(id, version), name, repositories.Visibility(visibility), repositories.Joining(joining)), nil
}

// Insert records the project to be written.
func (repository *ProjectRepository) Insert(project *repositories.Project) {
	repository.tracker.Add(change.NewEntry(change.Added, project))
}

// Update records the project to be written over the saved one.
func (repository *ProjectRepository) Update(project *repositories.Project) {
	repository.tracker.Add(change.NewEntry(change.Updated, project))
}

// ExecuteInsert writes the project as version 1.
func (repository *ProjectRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	insert := sqlbuilder.InsertInto("projects").
		Cols("id", "version", "name", "visibility", "joining").
		Values(project.ID().String(), 1, project.Name(), string(project.Visibility()), string(project.Joining()))

	query, args := insert.BuildWithFlavor(sqlbuilder.SQLite)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting project "+project.ID().String())
}

// ExecuteUpdate writes the whole project as the version after the one it
// was loaded at. The schema refuses it when the row changed since.
func (repository *ProjectRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	update := sqlbuilder.Update("projects")
	update.Set(
		update.Assign("version", project.Version()+1),
		update.Assign("name", project.Name()),
		update.Assign("visibility", string(project.Visibility())),
		update.Assign("joining", string(project.Joining())),
	)
	update.Where(update.Equal("id", project.ID().String()))

	query, args := update.BuildWithFlavor(sqlbuilder.SQLite)

	result, err := tx.ExecContext(ctx, query, args...)

	err = written(err, "updating project "+project.ID().String())
	if err != nil {
		return err
	}

	// the version trigger only fires for a row that exists, so a missing
	// row would otherwise pass as written
	return requireRow(result, "project "+project.ID().String())
}
