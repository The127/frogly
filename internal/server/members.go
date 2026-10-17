package server

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

type personJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func toPersonJSON(user *repositories.User) personJSON {
	return personJSON{ID: user.ID().String(), Name: user.Name(), Email: user.Email()}
}

func findPeople(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	found, err := projects.FindPeople(r.Context(), caller.unit, caller.user, id, r.URL.Query().Get("search"))
	if err != nil {
		return 0, nil, err
	}

	listed := make([]personJSON, 0, len(found))
	for _, user := range found {
		listed = append(listed, toPersonJSON(user))
	}

	return http.StatusOK, listed, nil
}

func addMember(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	var body struct {
		UserID string `json:"userId"`
	}

	err = decode(r, &body)
	if err != nil {
		return 0, nil, err
	}

	// an ID that can't be one gets the answer of an ID nobody has
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		return 0, nil, repositories.Invalid("nobody with that ID uses frogly")
	}

	added, err := projects.AddMember(r.Context(), caller.unit, caller.user, id, userID)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toPersonJSON(added), nil
}
