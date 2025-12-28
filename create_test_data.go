package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/turnerem/zenzen/config"
	"github.com/turnerem/zenzen/core"
	"github.com/turnerem/zenzen/storage"
)

// parseDuration converts strings like "5d", "2h", "1h30m", "2d5h" to time.Duration
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var total time.Duration
	var currentNum int
	hasDigits := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if ch >= '0' && ch <= '9' {
			currentNum = currentNum*10 + int(ch-'0')
			hasDigits = true
		} else if ch == 'd' || ch == 'h' || ch == 'm' || ch == 'w' {
			if !hasDigits {
				continue
			}

			switch ch {
			case 'm':
				total += time.Duration(currentNum) * time.Minute
			case 'h':
				total += time.Duration(currentNum) * time.Hour
			case 'd':
				total += time.Duration(currentNum) * core.DAY
			case 'w':
				total += time.Duration(currentNum) * core.WEEK
			}

			currentNum = 0
			hasDigits = false
		}
	}

	return total
}

func createTestData() error {
	ctx := context.Background()

	// Get database connection string
	connString, err := config.GetConnectionString()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Initialize SQL storage
	store, err := storage.NewSQLStorage(ctx, connString)
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer store.Close(ctx)

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

	// Save each entry to the database
	for _, log := range testLogs {
		entry := core.Entry{
			ID:                    log.id,
			Title:                 log.title,
			Tags:                  log.tags,
			StartedAtTimestamp:    log.startedAt,
			EndedAtTimestamp:      log.endedAt,
			EstimatedDuration:     log.estimatedDuration,
			Body:                  log.body,
			LastModifiedTimestamp: time.Now(),
		}

		if err := store.SaveEntry(entry); err != nil {
			return fmt.Errorf("error saving entry %s: %w", log.title, err)
		}

		fmt.Printf("✓ Added %s\n", log.title)
	}

	fmt.Printf("\nTest data created in database\n")
	return nil
}
