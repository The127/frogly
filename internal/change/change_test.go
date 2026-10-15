package change_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/the127/frogly/internal/change"
)

func TestTheTrackerKeepsTheOrderChangesWereAdded(t *testing.T) {
	// arrange
	tracker := change.NewTracker()

	// act
	tracker.Add(change.NewEntry(change.Added, "user"))
	tracker.Add(change.NewEntry(change.Updated, "project"))

	// assert
	entries := tracker.Changes()
	assert.Len(t, entries, 2)
	assert.Equal(t, change.Added, entries[0].Type())
	assert.Equal(t, "user", entries[0].Item())
	assert.Equal(t, change.Updated, entries[1].Type())
	assert.Equal(t, "project", entries[1].Item())
}

func TestClearForgetsEveryChange(t *testing.T) {
	// arrange
	tracker := change.NewTracker()
	tracker.Add(change.NewEntry(change.Added, "user"))

	// act
	tracker.Clear()

	// assert
	assert.Empty(t, tracker.Changes())
}
