package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/the127/frogly/internal/version"
)

func TestGetNamesAVersionEvenWithoutATag(t *testing.T) {
	t.Parallel()

	// act
	got := version.Get()

	// assert
	assert.NotEmpty(t, got)
}
