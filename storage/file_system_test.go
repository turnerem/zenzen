package storage

import (
	"testing"
	"testing/fstest"
)

var (
	DataK8s = `{
	"Title": "K8s",
	"Tags": ["learning", "open-source"],
	"Predicted Duration": "6d",
	"Duration": "",
	"Body": "The journey has just begun."
}`
	DataSystemDesign = `{
	"Title": "System Design",
	"Tags": ["interviews"],
	"Predicted Duration": "7d",
	"Duration": "8d",
	"Body": "Books combined with youtube resources were very helpful."
}`
)

func TestOSFileSystem(t *testing.T) {
	t.Run("GetAll logs from filesystem", func(t *testing.T) {
		fs := fstest.MapFS{
			"notes/K8s.json":          {Data: []byte(DataK8s)},
			"notes/SystemDesign.json": {Data: []byte(DataSystemDesign)},
		}

		// Setup
		storage := NewFSFileSystem("notes", fs)

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
