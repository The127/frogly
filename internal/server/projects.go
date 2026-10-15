package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

// a project's name is at most 100 characters, so anything bigger than this
// is not a project
const maxProjectBody = 4 << 10

var errMoreThanAProject = errors.New("the body holds more than a project")

type projectJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toProjectJSON(project *repositories.Project) projectJSON {
	return projectJSON{ID: project.ID().String(), Name: project.Name()}
}

func listProjects(w http.ResponseWriter, r *http.Request, caller caller) {
	found, err := projects.ListFor(r.Context(), caller.unit, caller.user)
	if err != nil {
		fail(w, r, "listing the projects failed", err)

		return
	}

	// a list, even an empty one, never null
	listed := make([]projectJSON, 0, len(found))
	for _, project := range found {
		listed = append(listed, toProjectJSON(project))
	}

	writeJSON(w, http.StatusOK, listed)
}

func createProject(w http.ResponseWriter, r *http.Request, caller caller) {
	var body struct {
		Name string `json:"name"`
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProjectBody))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&body)
	if err == nil {
		// the decoder stops after the first value, and anything after it
		// means the body was meant as something else
		err = decoder.Decode(&struct{}{})
		if errors.Is(err, io.EOF) {
			err = nil
		} else if err == nil {
			err = errMoreThanAProject
		}
	}

	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		writeProblem(w, http.StatusRequestEntityTooLarge)

		return
	}

	if err != nil {
		writeProblemDetail(w, http.StatusBadRequest, `the body isn't a project in JSON, like {"name": "Garden"}`)

		return
	}

	project, err := projects.Create(r.Context(), caller.unit, caller.user, body.Name)
	if invalid, ok := errors.AsType[*repositories.InvalidError](err); ok {
		writeProblemDetail(w, http.StatusBadRequest, invalid.Reason())

		return
	}

	if err != nil {
		fail(w, r, "creating the project failed", err)

		return
	}

	writeJSON(w, http.StatusCreated, toProjectJSON(project))
}
