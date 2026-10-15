// Package change records what a unit of work is going to write. The
// records are plain data, so they can be inspected, logged and tested.
package change

// Type says what happens to an item.
type Type int

const (
	// Added is an item the database doesn't have yet.
	Added Type = iota
	// Updated is an item the database has, written again as a whole.
	Updated
)

// Entry is one item and what happens to it. The item's Go type says which
// repository writes it.
type Entry struct {
	changeType Type
	item       any
}

// NewEntry records that item gets the change.
func NewEntry(changeType Type, item any) *Entry {
	return &Entry{
		changeType: changeType,
		item:       item,
	}
}

// Type says what happens.
func (entry *Entry) Type() Type {
	return entry.changeType
}

// Item is the model the entry is about.
func (entry *Entry) Item() any {
	return entry.item
}

// Tracker collects the entries of one unit of work.
type Tracker struct {
	entries []*Entry
}

// NewTracker creates a tracker without entries.
func NewTracker() *Tracker {
	return &Tracker{}
}

// Add records an entry. Entries are written in the order they were added,
// so a project comes before the memberships that point to it.
func (tracker *Tracker) Add(entry *Entry) {
	tracker.entries = append(tracker.entries, entry)
}

// Changes is every entry, in the order they were added.
func (tracker *Tracker) Changes() []*Entry {
	return tracker.entries
}

// Clear forgets every entry, once they are written.
func (tracker *Tracker) Clear() {
	tracker.entries = nil
}
