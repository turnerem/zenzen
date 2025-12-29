package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/turnerem/zenzen/api"
	"github.com/turnerem/zenzen/config"
	"github.com/turnerem/zenzen/core"
	"github.com/turnerem/zenzen/logger"
	"github.com/turnerem/zenzen/service"
	"github.com/turnerem/zenzen/storage"
)

func main() {
	// Check for commands first (before flag parsing)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setup":
			logger.SetupLogger("setup")
			if err := createTestData(); err != nil {
				log.Fatal("Error creating test data:", err)
			}
			return
		case "sync-now":
			logger.SetupLogger("sync")
			if err := runSyncNow(); err != nil {
				log.Fatal("Error running sync:", err)
			}
			return
		case "api":
			logger.SetupLogger("api")
			if err := runAPIServer(); err != nil {
				log.Fatal("Error running API server:", err)
			}
			return
		}
	}

	flag.Parse()

	// Setup logging for TUI mode (logs to file)
	logFile, err := logger.SetupLogger("tui")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not setup log file: %v\n", err)
		fmt.Fprintf(os.Stderr, "Continuing without logging...\n")
		logger.Disable()
	}
	if logFile != nil {
		defer logFile.Close()
	}

	ctx := context.Background()

	// Load full configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	// Get local connection string (with fallback to legacy format)
	localConnString := cfg.Database.LocalConnection
	if localConnString == "" {
		localConnString = cfg.Database.ConnectionString
	}
	if localConnString == "" {
		log.Fatal("No local database connection configured. Set local_connection in config.yaml")
	}

	// Initialize local SQL storage
	localStore, err := storage.NewSQLStorage(ctx, localConnString)
	if err != nil {
		log.Fatalf("Error connecting to local database: %v", err)
	}
	defer localStore.Close(ctx)

	// Initialize cloud storage and sync service if configured
	var syncService *service.SyncService
	if cfg.Sync.Enabled && cfg.Database.CloudConnection != "" {
		log.Println("Cloud sync enabled, initializing cloud storage...")

		cloudStore, err := storage.NewSQLStorage(ctx, cfg.Database.CloudConnection)
		if err != nil {
			log.Printf("Warning: Could not connect to cloud database: %v", err)
			log.Println("Continuing with local-only mode")
		} else {
			defer cloudStore.Close(ctx)

			// Get sync interval
			interval, err := cfg.GetSyncInterval()
			if err != nil {
				log.Printf("Warning: Invalid sync interval '%s', using default 60s: %v", cfg.Sync.Interval, err)
				interval = 60 * 1000000000 // 60 seconds in nanoseconds
			}

			// Create and start sync service
			syncService = service.NewSyncService(localStore, cloudStore, interval)
			syncService.Start()
			defer syncService.Stop()

			log.Printf("Sync service started with interval: %v", interval)
		}
	}

	// Initialize notes service (using local storage)
	notes := service.NewNotes(localStore)

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

// runSyncNow performs an immediate one-time sync between local and cloud databases
func runSyncNow() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Check if sync is configured
	if cfg.Database.CloudConnection == "" {
		log.Println("No cloud database configured. Please set cloud_connection in config.yaml")
		log.Println("See CLOUD_SETUP.md for instructions")
		return nil
	}

	// Get local connection
	localConnString := cfg.Database.LocalConnection
	if localConnString == "" {
		localConnString = cfg.Database.ConnectionString
	}
	if localConnString == "" {
		return fmt.Errorf("no local database connection configured")
	}

	log.Println("Connecting to local database...")
	localStore, err := storage.NewSQLStorage(ctx, localConnString)
	if err != nil {
		return fmt.Errorf("error connecting to local database: %w", err)
	}
	defer localStore.Close(ctx)

	log.Println("Connecting to cloud database...")
	cloudStore, err := storage.NewSQLStorage(ctx, cfg.Database.CloudConnection)
	if err != nil {
		return fmt.Errorf("error connecting to cloud database: %w", err)
	}
	defer cloudStore.Close(ctx)

	// Create sync service
	syncService := service.NewSyncService(localStore, cloudStore, 0)

	// Perform sync
	log.Println("Starting sync...")
	syncService.SyncNow()
	log.Println("Sync completed successfully!")

	return nil
}

// runAPIServer starts the HTTP API server
func runAPIServer() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Determine which database to use (prefer cloud for API)
	connString := cfg.Database.CloudConnection
	dbType := "cloud"

	if connString == "" {
		connString = cfg.Database.LocalConnection
		dbType = "local"
		if connString == "" {
			connString = cfg.Database.ConnectionString
		}
	}

	if connString == "" {
		return fmt.Errorf("no database connection configured")
	}

	log.Printf("API server will use %s database", dbType)

	// Connect to database
	store, err := storage.NewSQLStorage(ctx, connString)
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer store.Close(ctx)

	// Get API key from environment or generate a warning
	apiKey := os.Getenv("ZENZEN_API_KEY")
	if apiKey == "" {
		apiKey = "dev-key-change-in-production"
		log.Println("⚠️  WARNING: Using default API key. Set ZENZEN_API_KEY environment variable for production!")
	}

	// Get port from environment or use default
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	// Create API server
	apiServer := api.NewServer(store, apiKey)

	// Configure Cognito if environment variables are set
	cognitoRegion := os.Getenv("COGNITO_REGION")
	cognitoUserPoolID := os.Getenv("COGNITO_USER_POOL_ID")
	cognitoClientID := os.Getenv("COGNITO_CLIENT_ID")

	if cognitoRegion != "" && cognitoUserPoolID != "" && cognitoClientID != "" {
		log.Println("Cognito authentication enabled")
		cognito, err := api.NewCognitoConfig(cognitoRegion, cognitoUserPoolID, cognitoClientID)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to initialize Cognito: %v", err)
			log.Println("Falling back to API key authentication only")
		} else {
			apiServer.SetCognitoConfig(cognito)
			log.Printf("✓ Cognito configured: Region=%s, UserPoolID=%s", cognitoRegion, cognitoUserPoolID)
			log.Println("API accepts both:")
			log.Println("  - API Key: X-API-Key header")
			log.Println("  - Cognito: Authorization: Bearer <token>")
		}
	} else {
		log.Println("Cognito not configured (using API key only)")
		log.Println("To enable Cognito, set: COGNITO_REGION, COGNITO_USER_POOL_ID, COGNITO_CLIENT_ID")
	}

	log.Printf("API Key: %s", apiKey)
	log.Printf("Example (API Key): curl -H 'X-API-Key: %s' http://localhost:%d/api/v1/entries", apiKey, port)

	return apiServer.Start(port)
}
