package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/turnerem/zenzen/service"
	"github.com/turnerem/zenzen/storage"
)

func main() {
	// Check for setup command first (before flag parsing)
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		if err := createTestData(); err != nil {
			log.Fatal("Error creating test data:", err)
		}
		return
	}

	dir := flag.String("dir", "", "Directory for logs (default: ~/.zenzen)")
	flag.Parse()

	// Set default directory
	if *dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Error: could not determine home directory:", err)
		}
		*dir = filepath.Join(home, ".zenzen")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatal("Error: could not create logs directory:", err)
	}

	// Initialize filesystem and notes
	fs := storage.NewFSFileSystem("notes", os.DirFS(*dir))
	notes := service.NewNotes(fs)

	// Get all logs sorted by timestamp
	err := notes.LoadAll()
	if err != nil {
		log.Fatal("Error: could not load notes:", err)
	}

	// Start interactive TUI
	if err := StartTUI(notes.Entries); err != nil {
		log.Fatal("Error starting TUI:", err)
	}
}
