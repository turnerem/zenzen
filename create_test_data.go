package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/turnerem/zenzen/core"
)

// parseDuration converts strings like "5d", "2h", "3w" to time.Duration
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}

	var value int
	var unit string
	fmt.Sscanf(s, "%d%s", &value, &unit)

	switch unit {
	case "h":
		return time.Duration(value) * time.Hour
	case "d":
		return time.Duration(value) * core.DAY
	case "w":
		return time.Duration(value) * core.WEEK
	default:
		return 0
	}
}

func createTestData() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(home, ".zenzen", "notes")

	// Create directory
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	testLogs := []struct {
		id                string
		title             string
		tags              []string
		startedAt         time.Time
		endedAt           time.Time
		estimatedDuration time.Duration
		body              string
	}{
		{
			id:                "1",
			title:             "K8s Migration",
			tags:              []string{"DevOps", "Learning", "Infrastructure"},
			estimatedDuration: 5 * core.DAY,
			body:              "Successfully migrated our microservices from Docker Swarm to Kubernetes. Learned about helm charts, persistent volumes, and ingress controllers. Next time, pre-allocate more time for testing in staging environment.",
			startedAt:         time.Now().AddDate(0, 0, -15),
			endedAt:           time.Now().AddDate(0, 0, -15).Add(6 * core.DAY),
		},
		{
			id:                "2",
			title:             "System Design Interview",
			tags:              []string{"Interviews", "Learning"},
			estimatedDuration: 7 * core.DAY,
			body:              "Books combined with youtube resources were very helpful. Studied distributed systems, caching, and database design. Need to practice more on real-time systems.",
			startedAt:         time.Now().AddDate(0, 0, -10),
			endedAt:           time.Now().AddDate(0, 0, -10).Add(8 * core.DAY),
		},
		{
			id:                "3",
			title:             "Bug Fix: Memory Leak",
			tags:              []string{"Bug Fix", "Performance"},
			estimatedDuration: 4 * time.Hour,
			body:              "Found and fixed memory leak in request handler. The issue was goroutines not being properly cleaned up. Added profiling to CI/CD pipeline.",
			startedAt:         time.Now().AddDate(0, 0, -5),
			endedAt:           time.Now().AddDate(0, 0, -5).Add(2 * time.Hour),
		},
		{
			id:                "4",
			title:             "Refactor Database Layer",
			tags:              []string{"Refactoring", "Database"},
			estimatedDuration: 3 * core.DAY,
			body:              "Extracted database logic into separate package. Improved testability and reduced coupling. Added connection pooling for better performance.",
			startedAt:         time.Now().AddDate(0, 0, -3),
			endedAt:           time.Now().AddDate(0, 0, -3).Add(4 * core.DAY),
		},
		{
			id:                "5",
			title:             "Write API Documentation",
			tags:              []string{"Documentation", "API"},
			estimatedDuration: 2 * core.DAY,
			body:              "Started writing comprehensive API documentation. Documenting all endpoints, request/response formats, and error codes. Using OpenAPI 3.0 spec.",
			startedAt:         time.Now().AddDate(0, 0, -1),
			// endedAt is zero value - still in progress
		},
		{
			id:                "6",
			title:             "Code Review Session",
			tags:              []string{"Code Review", "Team"},
			estimatedDuration: 1 * core.DAY,
			body:              "Reviewed pull requests from team members. Provided feedback on architecture, testing, and code style. Great learning opportunity.",
			startedAt:         time.Now(),
			endedAt:           time.Now().Add(1 * core.DAY),
		},
	}

	// Create single notes.json file
	notesFile := filepath.Join(logsDir, "notes.json")
	f, err := os.Create(notesFile)
	if err != nil {
		return fmt.Errorf("error creating notes file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)

	for _, log := range testLogs {
		logData := core.Entry{
			ID:                log.id,
			Title:             log.title,
			Tags:              log.tags,
			StartedAt:         log.startedAt,
			EndedAt:           log.endedAt,
			EstimatedDuration: log.estimatedDuration,
			Body:              log.body,
		}

		// Encode to file (each entry on its own line)
		if err := encoder.Encode(logData); err != nil {
			return fmt.Errorf("error encoding log %s: %w", log.title, err)
		}

		fmt.Printf("✓ Added %s\n", log.title)
	}

	fmt.Printf("\nTest data created in %s\n", notesFile)
	return nil
}
