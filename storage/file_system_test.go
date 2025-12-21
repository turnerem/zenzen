package storage

import (
	"testing"
	"testing/fstest"
)

var (
	data = `{
	"Title": "K8s",
	"Tags": ["learning", "open-source"],
	"Predicted Duration": "6d",
	"Duration": "",
	"Body": "The journey has just begun."
}
{
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
			"notes.json": {Data: []byte(data)},
		}

		// Setup
		storage := NewFSFileSystem(fs)

		// Execute
		logs, err := storage.GetAll()

		// Verify
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		for _, l := range logs {
			if l.Title != "K8s" && l.Title != "System Design" {
				t.Errorf("Unexpected log title: %s", l.Title)
			}
		}
	})

}
