package service

import (
	"github.com/turnerem/zenzen/core"
)

type Store interface {
	GetAll() ([]core.Entry, error)
	// ReadDir(name string) ([]fs.DirEntry, error)
	// WriteFile(name string, data []byte, perm os.FileMode) error
	// Remove(name string) error
	// Open(name string) (fs.File, error)
}

type Notes struct {
	store   Store
	Entries []core.Entry
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
	// read in all logs and store in l.logs
	logs, err := l.store.GetAll()
	if err != nil {
		return err
	}
	l.Entries = logs

	return nil
}

// func (l *Notes) Delete(filename string) error {
// 	err := l.store.Remove(filename)
// 	if err != nil && !os.IsNotExist(err) {
// 		return err
// 	}
// 	return nil
// }

// returns logs for page size, filtered and sorted
// func (l *Notes) ListLogsSorted(opts Opts) ([]core.Entry, error) {

// }
