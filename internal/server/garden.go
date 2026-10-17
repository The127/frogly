package server

import (
	"net/http"
	"time"

	"github.com/the127/frogly/internal/garden"
	"github.com/the127/frogly/internal/repositories"
)

type ideaJSON struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	PlantedBy string    `json:"plantedBy"`
	PlantedAt time.Time `json:"plantedAt"`
}

func toIdeaJSON(idea *repositories.Idea, plantedBy string) ideaJSON {
	return ideaJSON{
		ID:        idea.ID().String(),
		Text:      idea.Text(),
		PlantedBy: plantedBy,
		PlantedAt: idea.PlantedAt(),
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
		listed = append(listed, toIdeaJSON(entry.Idea, entry.PlantedBy))
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

	return http.StatusCreated, toIdeaJSON(idea, caller.user.Name()), nil
}
