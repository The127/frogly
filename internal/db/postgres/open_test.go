package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/db/postgres"
)

func TestABrokenURLDoesntRevealThePassword(t *testing.T) {
	// act
	_, err := postgres.Open("postgres://frogly:s3c@cret@localhost:notaport/frogly")

	// assert
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "cret")
}
