package storage

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	data = `{"ID":"1","Title":"K8s","Tags":["learning","open-source"],"StartedAt":"0001-01-01T00:00:00Z","EndedAt":"0001-01-01T00:00:00Z","EstimatedDuration":0,"Body":"The journey has just begun."}
{"ID":"2","Title":"System Design","Tags":["interviews"],"StartedAt":"0001-01-01T00:00:00Z","EndedAt":"0001-01-01T00:00:00Z","EstimatedDuration":0,"Body":"Books combined with youtube resources were very helpful."}`
)

func TestFSFileSystem(t *testing.T) {
	t.Run("GetAll logs from filesystem", func(t *testing.T) {
		// Create temp directory (auto-cleanup)
		tmpDir := t.TempDir()
		notesFile := filepath.Join(tmpDir, "notes.json")

		// Write test data
		if err := os.WriteFile(notesFile, []byte(data), 0644); err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}

		// Setup
		storage := NewFSFileSystem(tmpDir)

		// Execute
		logs, err := storage.GetAll()

		// Verify
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(logs) != 2 {
			t.Fatalf("Expected 2 logs, got %d", len(logs))
		}

		for _, l := range logs {
			if l.Title != "K8s" && l.Title != "System Design" {
				t.Errorf("Unexpected log title: %s", l.Title)
			}
		}
	})

}
