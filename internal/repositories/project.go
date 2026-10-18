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
	joining    Joining
}

// Visibility says who besides its members sees a project.
type Visibility string

const (
	// VisibilityPublic shows the project to everyone who may use frogly.
	VisibilityPublic Visibility = "public"
	// VisibilityPrivate shows the project to its members only.
	VisibilityPrivate Visibility = "private"
)

// Joining says whether people outside a public project can join it
// themselves.
type Joining string

const (
	// JoiningOpen lets anyone who may use frogly join right away.
	JoiningOpen Joining = "open"
	// JoiningRequest lets anyone who may use frogly ask, and a member says
	// yes or no.
	JoiningRequest Joining = "request"
	// JoiningClosed lets only a member add someone.
	JoiningClosed Joining = "closed"
)

// checkJoining keeps a private project closed, since nobody outside can
// see it to join.
func checkJoining(visibility Visibility, joining Joining) error {
	switch {
	case joining != JoiningOpen && joining != JoiningRequest && joining != JoiningClosed:
		return Invalid("a project is open to join, takes requests or is closed")

	case joining != JoiningClosed && visibility != VisibilityPublic:
		return Invalid("only a public project can be open to join or take requests")
	}

	return nil
}

func checkVisibility(visibility Visibility) error {
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		return Invalid("a project is public or private")
	}

	return nil
}

// NewProject creates a project that was never saved. A project needs a
// name of up to 100 characters. A public one needs to say how people join
// it, and a private one without starts closed.
func NewProject(name string, visibility Visibility, joining Joining) (*Project, error) {
	name = strings.TrimSpace(name)

	switch {
	case name == "":
		return nil, Invalid("a project needs a name")

	case !validName(name):
		return nil, Invalid("a project's name has at most %d characters", MaxName)
	}

	err := checkVisibility(visibility)
	if err != nil {
		return nil, err
	}

	if joining == "" && visibility == VisibilityPrivate {
		joining = JoiningClosed
	}

	err = checkJoining(visibility, joining)
	if err != nil {
		return nil, err
	}

	return &Project{Base: newBase(), name: name, visibility: visibility, joining: joining}, nil
}

// NewProjectFromDB rebuilds a project as loaded.
func NewProjectFromDB(base Base, name string, visibility Visibility, joining Joining) *Project {
	return &Project{Base: base, name: name, visibility: visibility, joining: joining}
}

// Name is what the project is called.
func (project *Project) Name() string {
	return project.name
}

// Visibility says who besides its members sees the project.
func (project *Project) Visibility() Visibility {
	return project.visibility
}

// Joining says whether people outside the project can join it themselves.
// A private project is always closed.
func (project *Project) Joining() Joining {
	return project.joining
}

// ChangeJoining keeps the old setting when it refuses the new one.
func (project *Project) ChangeJoining(joining Joining) error {
	err := checkJoining(project.visibility, joining)
	if err != nil {
		return err
	}

	project.joining = joining

	return nil
}

// ChangeVisibility keeps the old visibility when it refuses the new one.
// Making a project private closes it, and it stays closed when it's made
// public again, so nobody gets in without a member opening it.
func (project *Project) ChangeVisibility(visibility Visibility) error {
	err := checkVisibility(visibility)
	if err != nil {
		return err
	}

	project.visibility = visibility
	if visibility == VisibilityPrivate {
		project.joining = JoiningClosed
	}

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
	// ListPublicForNonMember lists the public projects the user isn't a
	// member of, in the same order as ListForMember.
	ListPublicForNonMember(ctx context.Context, userID uuid.UUID) ([]*Project, error)
	Insert(project *Project)
	Update(project *Project)
}
