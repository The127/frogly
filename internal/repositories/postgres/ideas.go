package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

// IdeaRepository loads ideas from Postgres.
type IdeaRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewIdeaRepository creates the repository.
func NewIdeaRepository(reader Reader, tracker *change.Tracker) *IdeaRepository {
	return &IdeaRepository{reader: reader, tracker: tracker}
}

// ListInProject lists the project's garden, newest first.
func (repository *IdeaRepository) ListInProject(ctx context.Context, projectID uuid.UUID) ([]*repositories.Idea, error) {
	selection := sqlbuilder.Select("id", "version", "project_id", "planter_id", "text", "planted_at").From("ideas")
	selection.Where(selection.Equal("project_id", projectID))
	selection.OrderByDesc("planted_at").OrderByDesc("id")

	query, args := selection.BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing the ideas of project %s: %w", projectID, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var ideas []*repositories.Idea

	for rows.Next() {
		idea, err := scanIdea(rows)
		if err != nil {
			return nil, err
		}

		ideas = append(ideas, idea)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing the ideas of project %s: %w", projectID, err)
	}

	return ideas, nil
}

func scanIdea(row scanner) (*repositories.Idea, error) {
	var (
		id        uuid.UUID
		version   int
		projectID uuid.UUID
		planterID uuid.UUID
		text      string
		plantedAt time.Time
	)

	err := row.Scan(&id, &version, &projectID, &planterID, &text, &plantedAt)
	if err != nil {
		return nil, fmt.Errorf("reading an idea: %w", err)
	}

	base := repositories.NewBaseFromDB(id, version)

	return repositories.NewIdeaFromDB(base, projectID, planterID, text, plantedAt), nil
}

// Insert records the idea to be written.
func (repository *IdeaRepository) Insert(idea *repositories.Idea) {
	repository.tracker.Add(change.NewEntry(change.Added, idea))
}

// ExecuteInsert writes the idea as version 1.
func (repository *IdeaRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, idea *repositories.Idea) error {
	insert := sqlbuilder.InsertInto("ideas").
		Cols("id", "version", "project_id", "planter_id", "text", "planted_at").
		Values(idea.ID(), 1, idea.ProjectID(), idea.PlanterID(), idea.Text(), idea.PlantedAt())

	query, args := insert.BuildWithFlavor(sqlbuilder.PostgreSQL)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting idea "+idea.ID().String())
}
