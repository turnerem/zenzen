package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Sync     SyncConfig     `yaml:"sync"`
}

type DatabaseConfig struct {
	ConnectionString  string `yaml:"connection_string"`   // Legacy: local connection
	LocalConnection   string `yaml:"local_connection"`    // Local Postgres
	CloudConnection   string `yaml:"cloud_connection"`    // Cloud Postgres (RDS/Neon)
}

type SyncConfig struct {
	Enabled  bool   `yaml:"enabled"`  // Enable background sync
	Interval string `yaml:"interval"` // Sync interval (e.g. "60s", "5m")
}

// LoadConfig loads the full configuration from file or environment
func LoadConfig() (*Config, error) {
	configPath := "config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	// Apply environment variable overrides
	if localConn := os.Getenv("ZENZEN_LOCAL_DB_CONNECTION"); localConn != "" {
		cfg.Database.LocalConnection = localConn
	}
	if cloudConn := os.Getenv("ZENZEN_CLOUD_DB_CONNECTION"); cloudConn != "" {
		cfg.Database.CloudConnection = cloudConn
	}
	if syncEnabled := os.Getenv("ZENZEN_SYNC_ENABLED"); syncEnabled != "" {
		cfg.Sync.Enabled = syncEnabled == "true"
	}

	return &cfg, nil
}

// GetConnectionString returns the local database connection string.
// Precedence: ZENZEN_DB_CONNECTION env var > config.yaml > error
func GetConnectionString() (string, error) {
	// 1. Try environment variable first
	if connString := os.Getenv("ZENZEN_DB_CONNECTION"); connString != "" {
		return connString, nil
	}

	// 2. Try config.yaml in current directory
	cfg, err := LoadConfig()
	if err == nil {
		// Try new format first
		if cfg.Database.LocalConnection != "" {
			return cfg.Database.LocalConnection, nil
		}
		// Fall back to legacy format
		if cfg.Database.ConnectionString != "" {
			return cfg.Database.ConnectionString, nil
		}
	}

	// 3. Neither found
	return "", fmt.Errorf("no database connection configured. Set ZENZEN_DB_CONNECTION env var or create config.yaml (see config.example.yaml)")
}

// GetSyncInterval returns the sync interval as a time.Duration
func (c *Config) GetSyncInterval() (time.Duration, error) {
	if c.Sync.Interval == "" {
		return 60 * time.Second, nil // Default to 60 seconds
	}
	return time.ParseDuration(c.Sync.Interval)
}
