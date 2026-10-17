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

	name       string
	visibility Visibility
}

// Visibility says who besides its members sees a project.
type Visibility string

const (
	// VisibilityPublic shows the project to everyone who may use frogly.
	VisibilityPublic Visibility = "public"
	// VisibilityPrivate shows the project to its members only.
	VisibilityPrivate Visibility = "private"
)

func checkVisibility(visibility Visibility) error {
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		return invalid("a project is public or private")
	}

	return nil
}

// NewProject creates a project that was never saved. A project needs a
// name of up to 100 characters.
func NewProject(name string, visibility Visibility) (*Project, error) {
	name = strings.TrimSpace(name)

	switch {
	case name == "":
		return nil, invalid("a project needs a name")

	case !validName(name):
		return nil, invalid("a project's name has at most %d characters", MaxName)
	}

	err := checkVisibility(visibility)
	if err != nil {
		return nil, err
	}

	return &Project{Base: newBase(), name: name, visibility: visibility}, nil
}

// NewProjectFromDB rebuilds a project as loaded.
func NewProjectFromDB(base Base, name string, visibility Visibility) *Project {
	return &Project{Base: base, name: name, visibility: visibility}
}

// Name is what the project is called.
func (project *Project) Name() string {
	return project.name
}

// Visibility says who besides its members sees the project.
func (project *Project) Visibility() Visibility {
	return project.visibility
}

// ChangeVisibility keeps the old visibility when it refuses the new one.
func (project *Project) ChangeVisibility(visibility Visibility) error {
	err := checkVisibility(visibility)
	if err != nil {
		return err
	}

	project.visibility = visibility

	return nil
}

// ProjectRepository loads and saves projects.
type ProjectRepository interface {
	// ListForMember lists the projects the user is a member of, sorted by
	// name byte by byte, so upper case comes first, then by ID. Both
	// databases sort the same way.
	ListForMember(ctx context.Context, userID uuid.UUID) ([]*Project, error)
	// ByIDForMember returns the project if the user is a member of it, and
	// ErrNotFound if it doesn't exist or the user isn't a member, so a
	// caller can't tell the two apart.
	ByIDForMember(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*Project, error)
	Insert(project *Project)
}
