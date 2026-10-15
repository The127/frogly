package repositories

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Project is an area of work. What that is, a product, a team or a
// household, is up to the people in it.
type Project struct {
	Base

	name string
}

// NewProject creates a project that was never saved. A project needs a
// name of up to 100 characters.
func NewProject(name string) (*Project, error) {
	name = strings.TrimSpace(name)

	switch {
	case name == "":
		return nil, invalid("a project needs a name")

	case !validName(name):
		return nil, invalid("a project's name has at most %d characters", MaxName)
	}

	return &Project{Base: newBase(), name: name}, nil
}

// NewProjectFromDB rebuilds a project as loaded.
func NewProjectFromDB(base Base, name string) *Project {
	return &Project{Base: base, name: name}
}

// Name is what the project is called.
func (project *Project) Name() string {
	return project.name
}

// ProjectRepository loads and saves projects.
type ProjectRepository interface {
	// ListForMember lists the projects the user is a member of, sorted by
	// name byte by byte, so upper case comes first, then by ID. Both
	// databases sort the same way.
	ListForMember(ctx context.Context, userID uuid.UUID) ([]*Project, error)
	Insert(project *Project)
}
