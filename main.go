package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

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
				logger.Error("setup_failed", "error", err.Error())
				os.Exit(1)
			}
			return
		case "sync-now":
			logger.SetupLogger("sync")
			if err := runSyncNow(); err != nil {
				logger.Error("sync_command_failed", "error", err.Error())
				os.Exit(1)
			}
			return
		case "api":
			logger.SetupLogger("api")
			if err := runAPIServer(); err != nil {
				logger.Error("api_server_failed", "error", err.Error())
				os.Exit(1)
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
		logger.Error("config_load_failed", "error", err.Error())
		os.Exit(1)
	}

	// Get local connection string (with fallback to legacy format)
	localConnString := cfg.Database.LocalConnection
	if localConnString == "" {
		localConnString = cfg.Database.ConnectionString
	}
	if localConnString == "" {
		logger.Error("no_database_configured", "message", "Set local_connection in config.yaml")
		os.Exit(1)
	}

	// Initialize local SQL storage
	localStore, err := storage.NewSQLStorage(ctx, localConnString)
	if err != nil {
		logger.Error("local_database_connection_failed", "error", err.Error())
		os.Exit(1)
	}
	defer localStore.Close(ctx)

	// Initialize cloud storage and sync service if configured
	var syncService *service.SyncService
	if cfg.Sync.Enabled && cfg.Database.CloudConnection != "" {
		logger.Info("cloud_sync_enabled", "initializing", "cloud_storage")

		// Use a timeout context for cloud connection to fail fast if unreachable
		cloudCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		cloudStore, err := storage.NewSQLStorage(cloudCtx, cfg.Database.CloudConnection)
		if err != nil {
			logger.Warn("cloud_database_connection_failed", "error", err.Error(), "mode", "local_only")
		} else {
			defer cloudStore.Close(ctx)

			// Get sync interval
			interval, err := cfg.GetSyncInterval()
			if err != nil {
				logger.Warn("invalid_sync_interval", "interval", cfg.Sync.Interval, "error", err.Error(), "default", "60s")
				interval = 60 * 1000000000 // 60 seconds in nanoseconds
			}

			// Create and start sync service
			syncService = service.NewSyncService(localStore, cloudStore, interval)
			syncService.Start()
			defer syncService.Stop()
		}
	}

	// Initialize notes service (using local storage)
	notes := service.NewNotes(localStore)

	// Load all notes
	if err := notes.LoadAll(); err != nil {
		logger.Error("notes_load_failed", "error", err.Error())
		os.Exit(1)
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
		logger.Error("tui_start_failed", "error", err.Error())
		os.Exit(1)
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
		logger.Warn("no_cloud_database_configured", "message", "Please set cloud_connection in config.yaml")
		logger.Info("see_documentation", "file", "CLOUD_SETUP.md")
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

	logger.Info("connecting_to_database", "type", "local")
	localStore, err := storage.NewSQLStorage(ctx, localConnString)
	if err != nil {
		logger.Error("local_connection_failed", "error", err.Error())
		return fmt.Errorf("error connecting to local database: %w", err)
	}
	defer localStore.Close(ctx)

	logger.Info("connecting_to_database", "type", "cloud")
	cloudStore, err := storage.NewSQLStorage(ctx, cfg.Database.CloudConnection)
	if err != nil {
		logger.Error("cloud_connection_failed", "error", err.Error())
		return fmt.Errorf("error connecting to cloud database: %w", err)
	}
	defer cloudStore.Close(ctx)

	// Create sync service
	syncService := service.NewSyncService(localStore, cloudStore, 0)

	// Perform sync
	syncService.SyncNow()
	logger.Info("manual_sync_completed")

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

	logger.Info("api_database_selected", "type", dbType)

	// Connect to database
	store, err := storage.NewSQLStorage(ctx, connString)
	if err != nil {
		logger.Error("api_database_connection_failed", "error", err.Error())
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer store.Close(ctx)

	// Get API key from environment or generate a warning
	apiKey := os.Getenv("ZENZEN_API_KEY")
	if apiKey == "" {
		apiKey = "dev-key-change-in-production"
		logger.Warn("using_default_api_key", "message", "Set ZENZEN_API_KEY environment variable for production")
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
		logger.Info("cognito_authentication_enabled")
		cognito, err := api.NewCognitoConfig(cognitoRegion, cognitoUserPoolID, cognitoClientID)
		if err != nil {
			logger.Warn("cognito_init_failed", "error", err.Error(), "fallback", "api_key_only")
		} else {
			apiServer.SetCognitoConfig(cognito)
			logger.Info("cognito_configured", "region", cognitoRegion, "user_pool_id", cognitoUserPoolID)
			logger.Info("api_auth_methods", "methods", "api_key,cognito")
		}
	} else {
		logger.Info("cognito_not_configured", "auth_method", "api_key_only")
	}

	logger.Info("api_server_starting", "port", port, "api_key", apiKey)

	return apiServer.Start(port)
}
