package server

import (
	"net/http"

	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

type projectJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toProjectJSON(project *repositories.Project) projectJSON {
	return projectJSON{ID: project.ID().String(), Name: project.Name()}
}

func listProjects(r *http.Request, caller caller) (int, any, error) {
	found, err := projects.ListFor(r.Context(), caller.unit, caller.user)
	if err != nil {
		return 0, nil, err
	}

	// a list, even an empty one, never null
	listed := make([]projectJSON, 0, len(found))
	for _, project := range found {
		listed = append(listed, toProjectJSON(project))
	}

	return http.StatusOK, listed, nil
}

func createProject(r *http.Request, caller caller) (int, any, error) {
	var body struct {
		Name string `json:"name"`
	}

	err := decode(r, &body)
	if err != nil {
		return 0, nil, err
	}

	project, err := projects.Create(r.Context(), caller.unit, caller.user, body.Name)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusCreated, toProjectJSON(project), nil
}
