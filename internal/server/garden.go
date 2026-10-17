package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/garden"
	"github.com/the127/frogly/internal/repositories"
)

type ideaJSON struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	PlantedBy string    `json:"plantedBy"`
	PlantedAt time.Time `json:"plantedAt"`
	People    int       `json:"people"`
}

func toIdeaJSON(planted repositories.PlantedIdea) ideaJSON {
	return ideaJSON{
		ID:        planted.Idea.ID().String(),
		Text:      planted.Idea.Text(),
		PlantedBy: planted.PlantedBy,
		PlantedAt: planted.Idea.PlantedAt(),
		People:    planted.People,
	}
}

func listGarden(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	planted, err := garden.List(r.Context(), caller.unit, caller.user, id)
	if err != nil {
		return 0, nil, err
	}

	// a list, even an empty one, never null
	listed := make([]ideaJSON, 0, len(planted))
	for _, entry := range planted {
		listed = append(listed, toIdeaJSON(entry))
	}

	return http.StatusOK, listed, nil
}

func plantIdea(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	var body struct {
		Text string `json:"text"`
	}

	err = decode(r, &body)
	if err != nil {
		return 0, nil, err
	}

	idea, err := garden.Plant(r.Context(), caller.unit, caller.user, id, body.Text)
	if err != nil {
		return 0, nil, err
	}

	// a new idea has nobody behind it but its planter
	return http.StatusCreated, toIdeaJSON(repositories.PlantedIdea{Idea: idea, PlantedBy: caller.user.Name(), People: 1}), nil
}

func sayMeToo(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	// an ID that can't be one is an idea that doesn't exist
	ideaID, err := uuid.Parse(r.PathValue("ideaId"))
	if err != nil {
		return 0, nil, fmt.Errorf("idea %q: %w", r.PathValue("ideaId"), garden.ErrIdeaNotFound)
	}

	said, err := garden.MeToo(r.Context(), caller.unit, caller.user, id, ideaID)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toIdeaJSON(said), nil
}
