package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/change"
	"github.com/the127/frogly/internal/repositories"
)

func TestAChangeNoRepositoryMakesIsRefused(t *testing.T) {
	user, err := repositories.NewUser("Karo", "")
	require.NoError(t, err)

	membership, err := repositories.NewMembership(user.ID(), user.ID())
	require.NoError(t, err)

	request, err := repositories.NewJoinRequest(user.ID(), user.ID())
	require.NoError(t, err)

	for _, test := range []struct {
		name  string
		entry *change.Entry
	}{
		{name: "a deleted user", entry: change.NewEntry(change.Deleted, user)},
		{name: "an updated membership", entry: change.NewEntry(change.Updated, membership)},
		{name: "an updated request", entry: change.NewEntry(change.Updated, request)},
		{name: "an unknown change", entry: change.NewEntry(change.Type(99), user)},
	} {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			unit := &Context{}

			// act
			err := unit.apply(t.Context(), nil, test.entry)

			// assert
			require.ErrorIs(t, err, errUnsupportedChange)
		})
	}
}
