package people_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite"
	"github.com/the127/frogly/internal/people"
	"github.com/the127/frogly/internal/repositories"
)

func newDatabase(t *testing.T) db.Database {
	t.Helper()

	database, err := sqlite.Open(filepath.Join(t.TempDir(), "frogly.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	require.NoError(t, database.Migrate(t.Context()))

	return database
}

var karo = authn.Person{
	Issuer:  "http://issuer.example",
	Subject: "karo-id",
	Name:    "Karo",
	Email:   "karo@frogly.local",
}

func TestTheFirstRequestCreatesTheUser(t *testing.T) {
	// arrange
	database := newDatabase(t)

	// act
	user, err := people.Resolve(t.Context(), database.NewContext(), karo)

	// assert
	require.NoError(t, err)
	assert.Equal(t, "Karo", user.Name())
	assert.Equal(t, "karo@frogly.local", user.Email())

	stored, err := database.NewContext().Users().First(t.Context(), user.ID())
	require.NoError(t, err)
	assert.Equal(t, "Karo", stored.Name())
}

func TestALaterRequestFindsTheSameUser(t *testing.T) {
	// arrange
	database := newDatabase(t)
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
	database := newDatabase(t)
	first, err := people.Resolve(t.Context(), database.NewContext(), karo)
	require.NoError(t, err)

	renamed := karo
	renamed.Name = "Karolin"

	// act
	_, err = people.Resolve(t.Context(), database.NewContext(), renamed)

	// assert
	require.NoError(t, err)

	stored, err := database.NewContext().Users().First(t.Context(), first.ID())
	require.NoError(t, err)
	assert.Equal(t, "Karolin", stored.Name())
}

func TestALoginWithoutANameIsCalledBySubject(t *testing.T) {
	// arrange
	database := newDatabase(t)
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

func (identities *staleIdentities) FirstOrNil(ctx context.Context, issuer string, subject string) (*repositories.Identity, error) {
	if !identities.looked {
		identities.looked = true

		return nil, nil
	}

	return identities.IdentityRepository.FirstOrNil(ctx, issuer, subject)
}

func TestTheSlowerOfTwoFirstRequestsGetsTheSameUser(t *testing.T) {
	// arrange
	database := newDatabase(t)
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
