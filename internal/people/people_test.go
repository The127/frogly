package people_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
	"github.com/the127/frogly/internal/people"
	"github.com/the127/frogly/internal/repositories"
)

var karo = authn.Person{
	Issuer:  "http://issuer.example",
	Subject: "karo-id",
	Name:    "Karo",
	Email:   "karo@frogly.local",
}

func TestTheFirstRequestCreatesTheUser(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)

	// act
	user, err := people.Resolve(t.Context(), database.NewContext(), karo)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "Karo", user.Name())
	assert.Equal(t, "karo@frogly.local", user.Email())

	stored, err := database.NewContext().Users().ByID(t.Context(), user.ID())
	require.NoError(t, err)
	assert.Equal(t, "Karo", stored.Name())
}

func TestALaterRequestFindsTheSameUser(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	first, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	// act
	again, err := people.Resolve(t.Context(), database.NewContext(), karo)

	// assert
	require.NoError(t, err)
	assert.Equal(t, first.ID(), again.ID())
}

func TestAChangedProfileIsStored(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	first, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	renamed := karo
	renamed.Name = "Karolin"

	// act
	_, err = people.Resolve(t.Context(), database.NewContext(), renamed)

	// assert
	require.NoError(t, err)

	stored, err := database.NewContext().Users().ByID(t.Context(), first.ID())
	require.NoError(t, err)
	assert.Equal(t, "Karolin", stored.Name())
}

func TestALoginWithoutANameIsCalledBySubject(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	nameless := karo
	nameless.Name = ""

	// act
	user, err := people.Resolve(t.Context(), database.NewContext(), nameless)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "karo-id", user.Name())
}

// staleIdentities misses the identity another request saved a moment
// earlier, like the slower of two first requests does.
type staleIdentities struct {
	repositories.IdentityRepository

	looked bool
}

func (identities *staleIdentities) ByLogin(ctx context.Context, issuer string, subject string) (*repositories.Identity, error) {
	if !identities.looked {
		identities.looked = true

		return nil, nil
	}

	return identities.IdentityRepository.ByLogin(ctx, issuer, subject)
}

func TestTheSlowerOfTwoFirstRequestsGetsTheSameUser(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	faster, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	unit := database.NewContext()
	slower := staleUnit{Context: unit, identities: &staleIdentities{IdentityRepository: unit.Identities()}}

	// act
	user, err := people.Resolve(t.Context(), slower, karo)

	// assert
	require.NoError(t, err)
	assert.Equal(t, faster.ID(), user.ID())
}

type staleUnit struct {
	db.Context

	identities *staleIdentities
}

func (unit staleUnit) Identities() repositories.IdentityRepository {
	return unit.identities
}

func TestALoginNameFroglyCantUseFallsBackToTheSubject(t *testing.T) {
	tests := []struct {
		name     string
		person   authn.Person
		wantName string
	}{
		{
			name:     "only spaces",
			person:   authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: "   "},
			wantName: "karo-id",
		},
		{
			name:     "longer than 100 characters",
			person:   authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: strings.Repeat("蛙", 101)},
			wantName: "karo-id",
		},
		{
			name:     "no name and a subject longer than 100 characters",
			person:   authn.Person{Issuer: "http://issuer.example", Subject: strings.Repeat("k", 120)},
			wantName: strings.Repeat("k", 100),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			database := sqlitetest.Open(t)

			// act
			user, err := people.Resolve(t.Context(), database.NewContext(), test.person)

			// assert
			require.NoError(t, err)
			assert.Equal(t, test.wantName, user.Name())
		})
	}
}

func TestALoginWithoutAnEmailKeepsTheOldOne(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	withoutEmail := karo
	withoutEmail.Email = ""

	// act
	user, err := people.Resolve(t.Context(), database.NewContext(), withoutEmail)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "karo@frogly.local", user.Email())
}

func TestTheSameSubjectAtAnotherIssuerIsAnotherUser(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	first, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	elsewhere := karo
	elsewhere.Issuer = "http://elsewhere.example"

	// act
	other, err := people.Resolve(t.Context(), database.NewContext(), elsewhere)

	// assert
	require.NoError(t, err)
	assert.NotEqual(t, first.ID(), other.ID())
}

// racingSave loses the first save to another request, like a refresh that
// raced another one.
type racingSave struct {
	db.Context

	lost bool
}

func (unit *racingSave) SaveChanges(ctx context.Context) error {
	if !unit.lost {
		unit.lost = true

		return repositories.ErrConcurrentModification
	}

	return unit.Context.SaveChanges(ctx)
}

func TestARefreshThatLostARaceAnswersWithWhatIsStored(t *testing.T) {
	// arrange
	database := sqlitetest.Open(t)
	_, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	renamed := karo
	renamed.Name = "Karolin"

	// act
	user, err := people.Resolve(t.Context(), &racingSave{Context: database.NewContext()}, renamed)

	// assert
	require.NoError(t, err)

	stored, err := database.NewContext().Users().ByID(t.Context(), user.ID())
	require.NoError(t, err)
	assert.Equal(t, stored.Name(), user.Name())
}
