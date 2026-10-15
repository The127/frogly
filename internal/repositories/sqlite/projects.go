package sqlite

import (
	"context"
	"database/sql"
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
	selection := sqlbuilder.Select("p.id", "p.version", "p.name").From("projects p")
	selection.Join("memberships m", "m.project_id = p.id")
	selection.Where(selection.Equal("m.user_id", userID.String()))
	selection.OrderBy("p.name", "p.id")

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing the projects of user %s: %w", userID, err)
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
		return nil, fmt.Errorf("listing the projects of user %s: %w", userID, err)
	}

	return projects, nil
}

func scanProject(row scanner) (*repositories.Project, error) {
	var (
		id      uuid.UUID
		version int
		name    string
	)

	err := row.Scan(&id, &version, &name)
	if err != nil {
		return nil, fmt.Errorf("reading a project: %w", err)
	}

	return repositories.NewProjectFromDB(repositories.NewBaseFromDB(id, version), name), nil
}

// Insert records the project to be written.
func (repository *ProjectRepository) Insert(project *repositories.Project) {
	repository.tracker.Add(change.NewEntry(change.Added, project))
}

// ExecuteInsert writes the project as version 1.
func (repository *ProjectRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	insert := sqlbuilder.InsertInto("projects").
		Cols("id", "version", "name").
		Values(project.ID().String(), 1, project.Name())

	query, args := insert.BuildWithFlavor(sqlbuilder.SQLite)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting project "+project.ID().String())
}
