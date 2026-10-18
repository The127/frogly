package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/projects"
)

type requestJSON struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	AskedAt time.Time `json:"askedAt"`
}

// askerID reads whose request is answered from the path. An ID that can't
// be one is a request nobody made.
func askerID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("asker %q: %w", r.PathValue("userId"), projects.ErrNoRequest)
	}

	return id, nil
}

func askToJoin(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	asked, err := projects.AskToJoin(r.Context(), caller.unit, caller.user, id)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, map[string]bool{"asked": asked}, nil
}

func listRequests(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	waiting, err := projects.Requests(r.Context(), caller.unit, caller.user, id)
	if err != nil {
		return 0, nil, err
	}

	listed := make([]requestJSON, 0, len(waiting))
	for _, entry := range waiting {
		listed = append(listed, requestJSON{
			ID:      entry.Asker.ID().String(),
			Name:    entry.Asker.Name(),
			Email:   entry.Asker.Email(),
			AskedAt: entry.Request.AskedAt(),
		})
	}

	return http.StatusOK, listed, nil
}

func acceptRequest(r *http.Request, caller caller) (int, any, error) {
	id, asker, err := projectAndAsker(r)
	if err != nil {
		return 0, nil, err
	}

	accepted, err := projects.AcceptRequest(r.Context(), caller.unit, caller.user, id, asker)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toPersonJSON(accepted), nil
}

func declineRequest(r *http.Request, caller caller) (int, any, error) {
	id, asker, err := projectAndAsker(r)
	if err != nil {
		return 0, nil, err
	}

	err = projects.DeclineRequest(r.Context(), caller.unit, caller.user, id, asker)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusNoContent, nil, nil
}

func projectAndAsker(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	id, err := projectID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	asker, err := askerID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return id, asker, nil
}
