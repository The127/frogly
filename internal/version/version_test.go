package version_test

import (
	"testing"

	"github.com/the127/frogly/internal/version"
)

func TestGetNamesAVersionEvenWithoutATag(t *testing.T) {
	t.Parallel()

	if got := version.Get(); got == "" {
		t.Fatal("Get returned an empty version")
	}
}
