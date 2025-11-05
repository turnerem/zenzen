package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Check for setup command first (before flag parsing)
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		if err := createTestData(); err != nil {
			log.Fatal("Error creating test data:", err)
		}
		return
	}

	dir := flag.String("dir", "", "Directory for logs (default: ~/.gologit/logs)")
	flag.Parse()

	// Set default directory
	if *dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Error: could not determine home directory:", err)
		}
		*dir = filepath.Join(home, ".gologit", "logs")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatal("Error: could not create logs directory:", err)
	}

	// Initialize filesystem and logger
	fs := NewOSFileSystem(*dir)
	logger := NewLogger(fs)

	// Get all logs sorted by timestamp
	logs, err := logger.GetAllSorted()
	if err != nil {
		log.Fatal("Error: could not load logs:", err)
	}

	// Start interactive TUI
	if err := StartTUI(logs); err != nil {
		log.Fatal("Error starting TUI:", err)
	}
}
