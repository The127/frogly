package repositories

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// maxProjectName keeps names short enough for a list and a page title.
const maxProjectName = 100

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
		return nil, fmt.Errorf("%w: a project needs a name", ErrInvalid)

	case utf8.RuneCountInString(name) > maxProjectName:
		return nil, fmt.Errorf("%w: a project's name has at most %d characters", ErrInvalid, maxProjectName)
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
	// ListForMember lists the projects the user is a member of, by name.
	ListForMember(ctx context.Context, userID uuid.UUID) ([]*Project, error)
	Insert(project *Project)
}
