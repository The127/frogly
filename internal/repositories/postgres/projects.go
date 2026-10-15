package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

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
	rows, err := repository.reader.QueryContext(ctx,
		`select p.id, p.version, p.name
		from projects p
		join memberships m on m.project_id = p.id
		where m.user_id = $1
		order by p.name collate "C", p.id`,
		userID,
	)
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
	_, err := tx.ExecContext(ctx,
		"insert into projects (id, version, name) values ($1, 1, $2)",
		project.ID(), project.Name(),
	)

	return written(err, "inserting project "+project.ID().String())
}
