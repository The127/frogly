package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

type projectJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
	Joining    string `json:"joining"`
}

func toProjectJSON(project *repositories.Project) projectJSON {
	return projectJSON{
		ID:         project.ID().String(),
		Name:       project.Name(),
		Visibility: string(project.Visibility()),
		Joining:    string(project.Joining()),
	}
}

// publicProjectJSON is what someone outside a project sees of it.
type publicProjectJSON struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Joining string `json:"joining"`
}

func toPublicProjectJSON(project *repositories.Project) publicProjectJSON {
	return publicProjectJSON{ID: project.ID().String(), Name: project.Name(), Joining: string(project.Joining())}
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
		Name       string                  `json:"name"`
		Visibility repositories.Visibility `json:"visibility"`
		Joining    repositories.Joining    `json:"joining"`
	}

	err := decode(r, &body)
	if err != nil {
		return 0, nil, err
	}

	project, err := projects.Create(r.Context(), caller.unit, caller.user, body.Name, body.Visibility, body.Joining)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusCreated, toProjectJSON(project), nil
}

func changeVisibility(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	var body struct {
		Visibility repositories.Visibility `json:"visibility"`
	}

	err = decode(r, &body)
	if err != nil {
		return 0, nil, err
	}

	project, err := projects.ChangeVisibility(r.Context(), caller.unit, caller.user, id, body.Visibility)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toProjectJSON(project), nil
}

func listPublicProjects(r *http.Request, caller caller) (int, any, error) {
	found, err := projects.ListPublicFor(r.Context(), caller.unit, caller.user)
	if err != nil {
		return 0, nil, err
	}

	listed := make([]publicProjectJSON, 0, len(found))
	for _, project := range found {
		listed = append(listed, toPublicProjectJSON(project))
	}

	return http.StatusOK, listed, nil
}

func changeJoining(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	var body struct {
		Joining repositories.Joining `json:"joining"`
	}

	err = decode(r, &body)
	if err != nil {
		return 0, nil, err
	}

	project, err := projects.ChangeJoining(r.Context(), caller.unit, caller.user, id, body.Joining)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toProjectJSON(project), nil
}

func joinProject(r *http.Request, caller caller) (int, any, error) {
	id, err := projectID(r)
	if err != nil {
		return 0, nil, err
	}

	project, err := projects.Join(r.Context(), caller.unit, caller.user, id)
	if err != nil {
		return 0, nil, err
	}

	return http.StatusOK, toProjectJSON(project), nil
}
