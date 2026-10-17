package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

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

// projectID reads the project's ID from the path. An ID that can't be one
// is a project that doesn't exist.
func projectID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("project %q: %w", r.PathValue("id"), projects.ErrNotFound)
	}

	return id, nil
}

func getProject(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	project, err := projects.Get(r.Context(), caller.unit, caller.user, id)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toProjectJSON(project), nil
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

	project, err := projects.Create(r.Context(), caller.unit, caller.user, body.Name, repositories.VisibilityPrivate)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusCreated, toProjectJSON(project), nil
}
