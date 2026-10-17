package postgres

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

// ProjectRepository loads projects from Postgres.
type ProjectRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewProjectRepository creates the repository.
func NewProjectRepository(reader Reader, tracker *change.Tracker) *ProjectRepository {
	return &ProjectRepository{reader: reader, tracker: tracker}
}

// ListForMember lists the user's projects. Postgres sorts text by the
// database's locale unless told otherwise. The C collation compares byte
// by byte, which is the order the interface promises.
func (repository *ProjectRepository) ListForMember(ctx context.Context, userID uuid.UUID) ([]*repositories.Project, error) {
	selection := sqlbuilder.Select("p.id", "p.version", "p.name", "p.visibility").From("projects p")
	selection.Join("memberships m", "m.project_id = p.id")
	selection.Where(selection.Equal("m.user_id", userID))
	selection.OrderBy(`p.name collate "C"`, "p.id")

	return repository.list(ctx, selection, "the projects of user "+userID.String())
}

// ListPublicForNonMember lists the public projects the user isn't a
// member of, in the order of ListForMember.
func (repository *ProjectRepository) ListPublicForNonMember(ctx context.Context, userID uuid.UUID) ([]*repositories.Project, error) {
	membership := sqlbuilder.Select("1").From("memberships m")
	membership.Where("m.project_id = p.id", membership.Equal("m.user_id", userID))

	selection := sqlbuilder.Select("p.id", "p.version", "p.name", "p.visibility").From("projects p")
	selection.Where(
		selection.Equal("p.visibility", string(repositories.VisibilityPublic)),
		selection.NotExists(membership),
	)
	selection.OrderBy(`p.name collate "C"`, "p.id")

	return repository.list(ctx, selection, "the public projects for user "+userID.String())
}

func (repository *ProjectRepository) list(ctx context.Context, selection *sqlbuilder.SelectBuilder, what string) ([]*repositories.Project, error) {
	query, args := selection.BuildWithFlavor(sqlbuilder.PostgreSQL)

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
	selection := sqlbuilder.Select("p.id", "p.version", "p.name", "p.visibility").From("projects p")
	selection.Join("memberships m", "m.project_id = p.id")
	selection.Where(selection.Equal("p.id", projectID), selection.Equal("m.user_id", userID))

	query, args := selection.BuildWithFlavor(sqlbuilder.PostgreSQL)

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
	)

	err := row.Scan(&id, &version, &name, &visibility)
	if err != nil {
		return nil, fmt.Errorf("reading a project: %w", err)
	}

	return repositories.NewProjectFromDB(repositories.NewBaseFromDB(id, version), name, repositories.Visibility(visibility)), nil
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
		Cols("id", "version", "name", "visibility").
		Values(project.ID(), 1, project.Name(), string(project.Visibility()))

	query, args := insert.BuildWithFlavor(sqlbuilder.PostgreSQL)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting project "+project.ID().String())
}

// ExecuteUpdate writes the project as its next version.
func (repository *ProjectRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	update := sqlbuilder.Update("projects")
	update.Set(
		update.Assign("version", project.Version()+1),
		update.Assign("name", project.Name()),
		update.Assign("visibility", string(project.Visibility())),
	)
	update.Where(update.Equal("id", project.ID()))

	query, args := update.BuildWithFlavor(sqlbuilder.PostgreSQL)

	result, err := tx.ExecContext(ctx, query, args...)

	err = written(err, "updating project "+project.ID().String())
	if err != nil {
		return err
	}

	// the version trigger only fires for a row that exists, so a missing
	// row would otherwise pass as written
	return requireRow(result, "project "+project.ID().String())
}
