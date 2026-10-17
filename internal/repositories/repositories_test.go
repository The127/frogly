package repositories_test

import (
	"strings"
	"testing"
	"time"

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
	assert.Equal(t, uuid.Version(7), project.ID().Version())
	assert.Zero(t, project.Version())
}

func TestNewUserTakesATrimmedNameAndEmail(t *testing.T) {
	// act
	user, err := repositories.NewUser("  Karo ", " karo@frogly.local ")

	// assert
	require.NoError(t, err)
	assert.Equal(t, "Karo", user.Name())
	assert.Equal(t, "karo@frogly.local", user.Email())
}

func TestNewUserRefusesANameLongerThan100Characters(t *testing.T) {
	// act
	_, err := repositories.NewUser(strings.Repeat("蛙", 101), "")

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))
}

func TestNewIdentityBelongsToItsUser(t *testing.T) {
	// arrange
	userID := uuid.New()

	// act
	identity, err := repositories.NewIdentity(userID, "http://localhost:8082/oidc/frogly", "karo-id")

	// assert
	require.NoError(t, err)
	assert.Equal(t, userID, identity.UserID())
	assert.Equal(t, "http://localhost:8082/oidc/frogly", identity.Issuer())
	assert.Equal(t, "karo-id", identity.Subject())
}

func TestNewMembershipMakesTheUserAnOwner(t *testing.T) {
	// arrange
	projectID, userID := uuid.New(), uuid.New()

	// act
	membership, err := repositories.NewMembership(projectID, userID, repositories.RoleOwner)

	// assert
	require.NoError(t, err)
	assert.Equal(t, projectID, membership.ProjectID())
	assert.Equal(t, userID, membership.UserID())
	assert.Equal(t, repositories.RoleOwner, membership.Role())
}

func TestModelsRebuiltFromStorageKeepWhatWasStored(t *testing.T) {
	// arrange
	id := uuid.New()
	base := repositories.NewBaseFromDB(id, 3)

	// act
	user := repositories.NewUserFromDB(base, "", "not an email")
	project := repositories.NewProjectFromDB(base, "")
	identity := repositories.NewIdentityFromDB(base, uuid.Nil, "", "")

	// assert
	assert.Equal(t, id, user.ID())
	assert.Equal(t, 3, user.Version())
	assert.Equal(t, "not an email", user.Email())
	assert.Empty(t, project.Name())
	assert.Empty(t, identity.Issuer())
}

func TestNewUserNeedsAName(t *testing.T) {
	// act
	_, err := repositories.NewUser("  ", "karo@frogly.local")

	// assert
	require.ErrorAs(t, err, new(*repositories.InvalidError))
}

func TestRefreshTakesTheNewProfile(t *testing.T) {
	tests := []struct {
		name        string
		newName     string
		newEmail    string
		wantChanged bool
		wantName    string
		wantEmail   string
	}{
		{name: "same profile", newName: "Karo", newEmail: "karo@frogly.local", wantChanged: false, wantName: "Karo", wantEmail: "karo@frogly.local"},
		{name: "same profile with spaces", newName: " Karo ", newEmail: " karo@frogly.local ", wantChanged: false, wantName: "Karo", wantEmail: "karo@frogly.local"},
		{name: "new name", newName: "Karolin", newEmail: "karo@frogly.local", wantChanged: true, wantName: "Karolin", wantEmail: "karo@frogly.local"},
		{name: "new email", newName: "Karo", newEmail: "k@frogly.local", wantChanged: true, wantName: "Karo", wantEmail: "k@frogly.local"},
		{name: "empty name keeps the old one", newName: "", newEmail: "karo@frogly.local", wantChanged: false, wantName: "Karo", wantEmail: "karo@frogly.local"},
		{name: "empty email keeps the old one", newName: "Karo", newEmail: "", wantChanged: false, wantName: "Karo", wantEmail: "karo@frogly.local"},
		{name: "too long a name keeps the old one", newName: strings.Repeat("蛙", 101), newEmail: "karo@frogly.local", wantChanged: false, wantName: "Karo", wantEmail: "karo@frogly.local"},
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
			assert.Equal(t, test.wantEmail, user.Email())
		})
	}
}

func TestNewIdentityNeedsAnIssuerAndASubject(t *testing.T) {
	tests := []struct {
		name    string
		userID  uuid.UUID
		issuer  string
		subject string
	}{
		{name: "no user", userID: uuid.Nil, issuer: "http://localhost:8082/oidc/frogly", subject: "karo-id"},
		{name: "no issuer", userID: uuid.New(), issuer: "", subject: "karo-id"},
		{name: "no subject", userID: uuid.New(), issuer: "http://localhost:8082/oidc/frogly", subject: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewIdentity(test.userID, test.issuer, test.subject)

			// assert
			require.ErrorAs(t, err, new(*repositories.InvalidError))
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
			require.ErrorAs(t, err, new(*repositories.InvalidError))
		})
	}
}

func TestNewMembershipRefusesWhatItCantUse(t *testing.T) {
	tests := []struct {
		name      string
		projectID uuid.UUID
		userID    uuid.UUID
		role      repositories.Role
	}{
		{name: "unknown role", projectID: uuid.New(), userID: uuid.New(), role: repositories.Role("admiral")},
		{name: "no project", projectID: uuid.Nil, userID: uuid.New(), role: repositories.RoleOwner},
		{name: "no user", projectID: uuid.New(), userID: uuid.Nil, role: repositories.RoleOwner},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewMembership(test.projectID, test.userID, test.role)

			// assert
			require.ErrorAs(t, err, new(*repositories.InvalidError))
		})
	}
}

func TestNewUserFromLoginFallsBackToTheSubject(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		subject  string
		wantName string
	}{
		{name: "a usable name", login: " Karo ", subject: "karo-id", wantName: "Karo"},
		{name: "only spaces", login: "   ", subject: "karo-id", wantName: "karo-id"},
		{name: "too long a name", login: strings.Repeat("蛙", 101), subject: "karo-id", wantName: "karo-id"},
		{name: "too long a subject", login: "", subject: strings.Repeat("k", 120), wantName: strings.Repeat("k", 100)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			user, err := repositories.NewUserFromLogin(test.login, test.subject, "karo@frogly.local")

			// assert
			require.NoError(t, err)
			assert.Equal(t, test.wantName, user.Name())
		})
	}
}

func TestNewIdeaIsPlantedNowWithATrimmedText(t *testing.T) {
	// arrange
	projectID, planterID := uuid.New(), uuid.New()

	// act
	idea, err := repositories.NewIdea(projectID, planterID, "  search should find ideas by any word  ")

	// assert
	require.NoError(t, err)
	assert.Equal(t, projectID, idea.ProjectID())
	assert.Equal(t, planterID, idea.PlanterID())
	assert.Equal(t, "search should find ideas by any word", idea.Text())
	assert.WithinDuration(t, time.Now(), idea.PlantedAt(), time.Second)
	// both databases keep microseconds
	assert.Equal(t, idea.PlantedAt().Truncate(time.Microsecond), idea.PlantedAt())
	assert.Equal(t, time.UTC, idea.PlantedAt().Location())
}

func TestNewIdeaTakesWhatFitsInOneLine(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "500 characters", text: strings.Repeat("蛙", 500)},
		{name: "500 characters and spaces around them", text: "  " + strings.Repeat("蛙", 500) + "  "},
		{name: "a tab", text: "search\tfind"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewIdea(uuid.New(), uuid.New(), test.text)

			// assert
			require.NoError(t, err)
		})
	}
}

func TestNewIdeaRefusesATextItCantUse(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "empty", text: " \n "},
		{name: "a line feed", text: "one\ntwo"},
		{name: "a carriage return", text: "one\rtwo"},
		{name: "a line separator", text: "one\u2028two"},
		{name: "a NUL", text: "one\x00two"},
		{name: "invalid UTF-8", text: "one\xfftwo"},
		{name: "longer than 500 characters", text: strings.Repeat("蛙", 501)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewIdea(uuid.New(), uuid.New(), test.text)

			// assert
			require.ErrorAs(t, err, new(*repositories.InvalidError))
		})
	}
}

func TestNewIdeaNeedsAProjectAndAPlanter(t *testing.T) {
	tests := []struct {
		name      string
		projectID uuid.UUID
		planterID uuid.UUID
	}{
		{name: "no project", projectID: uuid.Nil, planterID: uuid.New()},
		{name: "no planter", projectID: uuid.New(), planterID: uuid.Nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := repositories.NewIdea(test.projectID, test.planterID, "an idea")

			// assert
			require.ErrorAs(t, err, new(*repositories.InvalidError))
		})
	}
}

func TestNewIdeaFromDBKeepsTheTimeInUTC(t *testing.T) {
	// arrange
	plantedAt := time.Date(2026, 10, 15, 23, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

	// act
	idea := repositories.NewIdeaFromDB(repositories.NewBaseFromDB(uuid.New(), 1), uuid.New(), uuid.New(), "an idea", plantedAt)

	// assert
	assert.Equal(t, time.UTC, idea.PlantedAt().Location())
	assert.True(t, plantedAt.Equal(idea.PlantedAt()))
}
