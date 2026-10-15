package repositories_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/repositories"
)

func TestANewModelHasAnIDButNoVersion(t *testing.T) {
	// act
	project, err := repositories.NewProject("Garden")

	// assert
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, project.ID())
	assert.Zero(t, project.Version())
}

func TestNewUserNeedsAName(t *testing.T) {
	// act
	_, err := repositories.NewUser("  ", "karo@frogly.local")

	// assert
	require.ErrorIs(t, err, repositories.ErrInvalid)
}

func TestRefreshTakesTheNewProfile(t *testing.T) {
	tests := []struct {
		name        string
		newName     string
		newEmail    string
		wantChanged bool
		wantName    string
	}{
		{name: "same profile", newName: "Karo", newEmail: "karo@frogly.local", wantChanged: false, wantName: "Karo"},
		{name: "new name", newName: "Karolin", newEmail: "karo@frogly.local", wantChanged: true, wantName: "Karolin"},
		{name: "new email", newName: "Karo", newEmail: "k@frogly.local", wantChanged: true, wantName: "Karo"},
		{name: "empty name keeps the old one", newName: "", newEmail: "karo@frogly.local", wantChanged: false, wantName: "Karo"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			user, err := repositories.NewUser("Karo", "karo@frogly.local")
			require.NoError(t, err)

			// act
			changed := user.Refresh(test.newName, test.newEmail)

			// assert
			assert.Equal(t, test.wantChanged, changed)
			assert.Equal(t, test.wantName, user.Name())
		})
	}
}

func TestNewIdentityNeedsAnIssuerAndASubject(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		subject string
	}{
		{name: "no issuer", issuer: "", subject: "karo-id"},
		{name: "no subject", issuer: "http://localhost:8082/oidc/frogly", subject: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewIdentity(uuid.New(), test.issuer, test.subject)

			// assert
			require.ErrorIs(t, err, repositories.ErrInvalid)
		})
	}
}

func TestNewProjectTakesATrimmedName(t *testing.T) {
	// act
	project, err := repositories.NewProject("  Garden  ")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "Garden", project.Name())
}

func TestNewProjectRefusesNamesItCantUse(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
	}{
		{name: "empty", projectName: "   "},
		{name: "longer than 100 characters", projectName: strings.Repeat("蛙", 101)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewProject(test.projectName)

			// assert
			require.ErrorIs(t, err, repositories.ErrInvalid)
		})
	}
}

func TestNewMembershipNeedsAKnownRole(t *testing.T) {
	// act
	_, err := repositories.NewMembership(uuid.New(), uuid.New(), repositories.Role("admiral"))

	// assert
	require.ErrorIs(t, err, repositories.ErrInvalid)
}
