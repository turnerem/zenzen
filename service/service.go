package service

import (
	"time"

	"github.com/turnerem/zenzen/core"
)

type Store interface {
	GetAll() (map[string]core.Entry, error)
	SaveEntry(entry core.Entry) error
	DeleteEntry(id string) error
}

type Notes struct {
	store   Store
	Entries map[string]core.Entry
}

type Opts struct {
	// Page     int
	// PageSize int
	SortBy string // e.g., "Title", "Date"
	// Filter   string // e.g., "Tag:learning"
}

func NewNotes(store Store) *Notes {
	return &Notes{store: store}
}

func (l *Notes) LoadAll() error {
	// read in all logs and store in l.Entries
	logs, err := l.store.GetAll()
	if err != nil {
		return err
	}
	l.Entries = logs

	return nil
}

func (l *Notes) Delete(ID string) error {
	delete(l.Entries, ID)
	return l.store.DeleteEntry(ID)
}

// SaveEntry persists a single entry to storage
// Sets LastModifiedTimestamp to current time before saving
func (l *Notes) SaveEntry(entry core.Entry) error {
	// Set last modified timestamp for user edits
	entry.LastModifiedTimestamp = time.Now()

	l.Entries[entry.ID] = entry
	return l.store.SaveEntry(entry)
}

// returns logs for page size, filtered and sorted
// func (l *Notes) ListLogsSorted(opts Opts) ([]core.Entry, error) {

// }
