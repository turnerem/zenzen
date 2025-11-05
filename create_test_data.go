package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)


func createTestData() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(home, ".gologit", "logs")

	// Create directory
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	testLogs := []struct {
		title      string
		tags       []string
		predicted  string
		actual     string
		body       string
		startedAt  time.Time
	}{
		{
			title:     "K8s Migration",
			tags:      []string{"DevOps", "Learning", "Infrastructure"},
			predicted: "5d",
			actual:    "6d",
			body:      "Successfully migrated our microservices from Docker Swarm to Kubernetes. Learned about helm charts, persistent volumes, and ingress controllers. Next time, pre-allocate more time for testing in staging environment.",
			startedAt: time.Now().AddDate(0, 0, -15),
		},
		{
			title:     "System Design Interview",
			tags:      []string{"Interviews", "Learning"},
			predicted: "7d",
			actual:    "8d",
			body:      "Books combined with youtube resources were very helpful. Studied distributed systems, caching, and database design. Need to practice more on real-time systems.",
			startedAt: time.Now().AddDate(0, 0, -10),
		},
		{
			title:     "Bug Fix: Memory Leak",
			tags:      []string{"Bug Fix", "Performance"},
			predicted: "4h",
			actual:    "2h",
			body:      "Found and fixed memory leak in request handler. The issue was goroutines not being properly cleaned up. Added profiling to CI/CD pipeline.",
			startedAt: time.Now().AddDate(0, 0, -5),
		},
		{
			title:     "Refactor Database Layer",
			tags:      []string{"Refactoring", "Database"},
			predicted: "3d",
			actual:    "4d",
			body:      "Extracted database logic into separate package. Improved testability and reduced coupling. Added connection pooling for better performance.",
			startedAt: time.Now().AddDate(0, 0, -3),
		},
		{
			title:     "Write API Documentation",
			tags:      []string{"Documentation", "API"},
			predicted: "2d",
			actual:    "",
			body:      "Started writing comprehensive API documentation. Documenting all endpoints, request/response formats, and error codes. Using OpenAPI 3.0 spec.",
			startedAt: time.Now().AddDate(0, 0, -1),
		},
		{
			title:     "Code Review Session",
			tags:      []string{"Code Review", "Team"},
			predicted: "1d",
			actual:    "1d",
			body:      "Reviewed pull requests from team members. Provided feedback on architecture, testing, and code style. Great learning opportunity.",
			startedAt: time.Now(),
		},
	}

	for _, log := range testLogs {
		logData := Log{
			Title:         log.title,
			Tags:          log.tags,
			TTCPrediction: log.predicted,
			TTCActual:     log.actual,
			Body:          log.body,
			Start:         log.startedAt.Format(time.RFC3339),
		}

		// Create filename from title
		filename := log.title + ".json"
		filePath := filepath.Join(logsDir, filename)

		// Marshal to JSON
		data, err := json.MarshalIndent(logData, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling log %s: %w", log.title, err)
		}

		// Write to file
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("error writing log file %s: %w", filePath, err)
		}

		fmt.Printf("✓ Created %s\n", filename)
	}

	fmt.Printf("\nTest data created in %s\n", logsDir)
	return nil
}
