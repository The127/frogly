// Package repositories holds the models frogly stores, and the interfaces
// of the repositories that load and save them, independent of any
// database. Reads happen right away. Writes are only recorded, and the
// unit of work writes them all at once.
package repositories

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalid is wrapped into every refusal of a model that can't be stored
// as given.
var ErrInvalid = errors.New("invalid")

// ErrNotFound is returned when nothing matches.
var ErrNotFound = errors.New("not found")

// maxName keeps names short enough for a list and a page title.
const maxName = 100

// ErrConcurrentModification is returned when saving would overwrite
// someone else's change, because a row changed since it was loaded, or a
// unique value was taken since it was looked up.
var ErrConcurrentModification = errors.New("concurrent modification")

// Base is what every stored model has: an ID and the version it was loaded
// at. The database counts the version up on every update. A model is stale
// once the unit of work that loaded it has saved: load it again before
// changing it again.
type Base struct {
	id      uuid.UUID
	version int
}

func newBase() Base {
	// a version 7 UUID starts with the time, so new rows land at the end of
	// an index instead of all over it
	return Base{id: uuid.Must(uuid.NewV7())}
}

// NewBaseFromDB rebuilds the base of a model as loaded.
func NewBaseFromDB(id uuid.UUID, version int) Base {
	return Base{id: id, version: version}
}

// ID is the model's ID.
func (base *Base) ID() uuid.UUID {
	return base.id
}

// Version is the version the model was loaded at, 0 for a model that was
// never saved.
func (base *Base) Version() int {
	return base.version
}
