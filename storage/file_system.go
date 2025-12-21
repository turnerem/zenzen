package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/turnerem/zenzen/core"
)

// get all logs from fs
// add log
// delete log

const FILENAME = "notes.json"

type FSFileSystem struct {
	baseDir string
}

func NewFSFileSystem(baseDir string) *FSFileSystem {
	return &FSFileSystem{
		baseDir: baseDir,
	}
}

func (o *FSFileSystem) GetAll() (map[string]core.Entry, error) {
	filePath := filepath.Join(o.baseDir, FILENAME)
	logFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	entries := make(map[string]core.Entry)
	decoder := json.NewDecoder(bytes.NewReader(logFile))

	for {
		var entry core.Entry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		entries[entry.ID] = entry
	}

	return entries, nil
}

// Save writes all entries back to the notes file in NDJSON format
func (o *FSFileSystem) Save(entries map[string]core.Entry) error {
	filePath := filepath.Join(o.baseDir, FILENAME)

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)

	// Write entries (order not guaranteed, map iteration order)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}

	return nil
}
