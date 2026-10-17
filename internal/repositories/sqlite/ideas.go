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

// IdeaRepository loads ideas from SQLite.
type IdeaRepository struct {
	reader  Reader
	tracker *change.Tracker
}

// NewIdeaRepository creates the repository.
func NewIdeaRepository(reader Reader, tracker *change.Tracker) *IdeaRepository {
	return &IdeaRepository{reader: reader, tracker: tracker}
}

// ListInProject lists the project's garden with the planters' names,
// newest first.
func (repository *IdeaRepository) ListInProject(ctx context.Context, projectID uuid.UUID) ([]repositories.PlantedIdea, error) {
	selection := selectPlanted()
	selection.Where(selection.Equal("i.project_id", projectID.String()))
	selection.OrderByDesc("i.planted_at").OrderByDesc("i.id")

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	rows, err := repository.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing the ideas of project %s: %w", projectID, err)
	}

	defer func() {
		_ = rows.Close() // a read-only query, closing can't lose anything
	}()

	var ideas []repositories.PlantedIdea

	for rows.Next() {
		idea, err := scanPlantedIdea(rows)
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

// selectPlanted counts the people behind each idea in the same query, so
// a garden costs one query however many ideas it has. The planter counts
// once, even if a me too of theirs got stored.
func selectPlanted() *sqlbuilder.SelectBuilder {
	return sqlbuilder.Select(
		"i.id", "i.version", "i.project_id", "i.planter_id", "i.text", "i.planted_at", "u.name",
		"1 + (select count(*) from me_toos t where t.idea_id = i.id and t.user_id <> i.planter_id)",
	).From("ideas i").Join("users u", "u.id = i.planter_id")
}

// ByIDInProject loads one idea of the project, or returns
// repositories.ErrNotFound.
func (repository *IdeaRepository) ByIDInProject(ctx context.Context, projectID uuid.UUID, ideaID uuid.UUID) (repositories.PlantedIdea, error) {
	selection := selectPlanted()
	selection.Where(selection.Equal("i.project_id", projectID.String()), selection.Equal("i.id", ideaID.String()))

	query, args := selection.BuildWithFlavor(sqlbuilder.SQLite)

	idea, err := scanPlantedIdea(repository.reader.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return repositories.PlantedIdea{}, fmt.Errorf("idea %s in project %s: %w", ideaID, projectID, repositories.ErrNotFound)
	}

	if err != nil {
		return repositories.PlantedIdea{}, fmt.Errorf("loading idea %s: %w", ideaID, err)
	}

	return idea, nil
}

func scanPlantedIdea(row scanner) (repositories.PlantedIdea, error) {
	var (
		id        uuid.UUID
		version   int
		projectID uuid.UUID
		planterID uuid.UUID
		text      string
		plantedBy string
		people    int
		plantedAt int64
	)

	err := row.Scan(&id, &version, &projectID, &planterID, &text, &plantedAt, &plantedBy, &people)
	if err != nil {
		return repositories.PlantedIdea{}, fmt.Errorf("reading an idea: %w", err)
	}

	base := repositories.NewBaseFromDB(id, version)
	idea := repositories.NewIdeaFromDB(base, projectID, planterID, text, time.UnixMicro(plantedAt))

	return repositories.PlantedIdea{Idea: idea, PlantedBy: plantedBy, People: people}, nil
}

// Insert records the idea to be written.
func (repository *IdeaRepository) Insert(idea *repositories.Idea) {
	repository.tracker.Add(change.NewEntry(change.Added, idea))
}

// ExecuteInsert writes the idea as version 1.
func (repository *IdeaRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, idea *repositories.Idea) error {
	insert := sqlbuilder.InsertInto("ideas").
		Cols("id", "version", "project_id", "planter_id", "text", "planted_at").
		Values(idea.ID().String(), 1, idea.ProjectID().String(), idea.PlanterID().String(), idea.Text(), idea.PlantedAt().UnixMicro())

	query, args := insert.BuildWithFlavor(sqlbuilder.SQLite)

	_, err := tx.ExecContext(ctx, query, args...)

	return written(err, "inserting idea "+idea.ID().String())
}
