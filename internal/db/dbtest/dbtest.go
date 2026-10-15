// Package dbtest is the contract every database adapter meets. An
// adapter's tests call Run with a way to open a fresh database.
package dbtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/repositories"
)

// Opener returns a fresh, empty, migrated database for a test, and
// registers its cleanup with that test.
type Opener func(t *testing.T) db.Database

// Run checks an adapter against the whole contract, with one subtest per
// area, so `-run 'Contract/ideas'` checks only the ideas.
func Run(t *testing.T, open Opener) {
	t.Helper()

	t.Run("unit of work", func(t *testing.T) { suite.Run(t, &unitOfWorkSuite{base{open: open}}) })
	t.Run("users", func(t *testing.T) { suite.Run(t, &usersSuite{base{open: open}}) })
	t.Run("identities", func(t *testing.T) { suite.Run(t, &identitiesSuite{base{open: open}}) })
	t.Run("projects", func(t *testing.T) { suite.Run(t, &projectsSuite{base{open: open}}) })
	t.Run("ideas", func(t *testing.T) { suite.Run(t, &ideasSuite{base{open: open}}) })
}

// base is what every suite shares: a fresh database for each test, and
// fixtures that build a model, save it and return it.
type base struct {
	suite.Suite

	open     Opener
	database db.Database
}

// SetupTest opens a fresh database for the test.
func (s *base) SetupTest() {
	s.database = s.open(s.T())
}

func (s *base) save(unit db.Context) {
	s.Require().NoError(unit.SaveChanges(s.T().Context()))
}

func (s *base) savedUser(name string) *repositories.User {
	user, err := repositories.NewUser(name, name+"@frogly.local")
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Users().Insert(user)
	s.save(unit)

	return user
}

// the login savedIdentity saves, which tests use again for a second
// identity with the same login
const (
	issuer  = "http://issuer.example"
	subject = "karo-id"
)

func (s *base) savedIdentity(user *repositories.User) *repositories.Identity {
	identity, err := repositories.NewIdentity(user.ID(), issuer, subject)
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Identities().Insert(identity)
	s.save(unit)

	return identity
}

// savedProject saves the project with owner as its only member.
func (s *base) savedProject(name string, owner *repositories.User) *repositories.Project {
	project, err := repositories.NewProject(name)
	s.Require().NoError(err)

	membership, err := repositories.NewMembership(project.ID(), owner.ID(), repositories.RoleOwner)
	s.Require().NoError(err)

	unit := s.database.NewContext()
	unit.Projects().Insert(project)
	unit.Memberships().Insert(membership)
	s.save(unit)

	return project
}

func (s *base) savedIdea(project *repositories.Project, planter *repositories.User, text string) *repositories.Idea {
	return s.savedIdeaAt(project, planter, text, time.Time{})
}

// savedIdeaAt is savedIdea for a test that chooses the planting time. The
// zero time keeps the time NewIdea gives.
func (s *base) savedIdeaAt(project *repositories.Project, planter *repositories.User, text string, plantedAt time.Time) *repositories.Idea {
	idea, err := repositories.NewIdea(project.ID(), planter.ID(), text)
	s.Require().NoError(err)

	if !plantedAt.IsZero() {
		idea = repositories.NewIdeaFromDB(idea.Base, project.ID(), planter.ID(), text, plantedAt)
	}

	unit := s.database.NewContext()
	unit.Ideas().Insert(idea)
	s.save(unit)

	return idea
}
