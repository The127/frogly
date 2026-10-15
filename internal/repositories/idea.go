package repositories

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// MaxIdeaText keeps an idea to what fits in one line, in characters.
const MaxIdeaText = 500

// Idea is something someone wrote down in a project's garden.
type Idea struct {
	Base

	projectID uuid.UUID
	planterID uuid.UUID
	text      string
	plantedAt time.Time
}

// NewIdea plants an idea that was never saved. Its text is one line of up
// to 500 characters.
func NewIdea(projectID uuid.UUID, planterID uuid.UUID, text string) (*Idea, error) {
	text = strings.TrimSpace(text)

	switch {
	case projectID == uuid.Nil || planterID == uuid.Nil:
		return nil, invalid("an idea needs a project and someone who plants it")

	case text == "":
		return nil, invalid("an idea needs a text")

	case strings.ContainsAny(text, "\r\n"):
		return nil, invalid("an idea is one line")

	case utf8.RuneCountInString(text) > MaxIdeaText:
		return nil, invalid("an idea has at most %d characters", MaxIdeaText)
	}

	return &Idea{
		Base:      newBase(),
		projectID: projectID,
		planterID: planterID,
		text:      text,
		// both databases keep microseconds, so the idea looks the same
		// before and after it's saved
		plantedAt: time.Now().UTC().Truncate(time.Microsecond),
	}, nil
}

// NewIdeaFromDB rebuilds an idea as loaded.
func NewIdeaFromDB(base Base, projectID uuid.UUID, planterID uuid.UUID, text string, plantedAt time.Time) *Idea {
	return &Idea{
		Base:      base,
		projectID: projectID,
		planterID: planterID,
		text:      text,
		plantedAt: plantedAt,
	}
}

// ProjectID is the project whose garden the idea is in.
func (idea *Idea) ProjectID() uuid.UUID {
	return idea.projectID
}

// PlanterID is the user who planted the idea.
func (idea *Idea) PlanterID() uuid.UUID {
	return idea.planterID
}

// Text is the idea itself.
func (idea *Idea) Text() string {
	return idea.text
}

// PlantedAt is when the idea was planted, in UTC.
func (idea *Idea) PlantedAt() time.Time {
	return idea.plantedAt
}

// IdeaRepository loads and saves ideas.
type IdeaRepository interface {
	// ListInProject lists the project's garden, newest first. Ideas planted
	// at the same time are sorted by ID, the newest first as well.
	ListInProject(ctx context.Context, projectID uuid.UUID) ([]*Idea, error)
	Insert(idea *Idea)
}
