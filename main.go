package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/turnerem/zenzen/config"
	"github.com/turnerem/zenzen/core"
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

	flag.Parse()

	ctx := context.Background()

	// Get database connection string from env var or config file
	connString, err := config.GetConnectionString()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	// Initialize SQL storage
	store, err := storage.NewSQLStorage(ctx, connString)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer store.Close(ctx)

	// Initialize notes service
	notes := service.NewNotes(store)

	// Load all notes
	if err := notes.LoadAll(); err != nil {
		log.Fatalf("Error loading notes: %v", err)
	}

	// Create callbacks for TUI
	saveEntryFn := func(entry core.Entry) error {
		return notes.SaveEntry(entry)
	}

	deleteEntryFn := func(id string) error {
		return notes.Delete(id)
	}

	// Start interactive TUI
	if err := StartTUI(notes.Entries, saveEntryFn, deleteEntryFn); err != nil {
		log.Fatalf("Error starting TUI: %v", err)
	}
}
