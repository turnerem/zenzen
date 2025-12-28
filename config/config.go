package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
	ConnectionString string `yaml:"connection_string"`
}

// GetConnectionString returns the database connection string.
// Precedence: ZENZEN_DB_CONNECTION env var > config.yaml > error
func GetConnectionString() (string, error) {
	// 1. Try environment variable first
	if connString := os.Getenv("ZENZEN_DB_CONNECTION"); connString != "" {
		return connString, nil
	}

	// 2. Try config.yaml in current directory
	configPath := "config.yaml"
	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return "", fmt.Errorf("failed to parse config.yaml: %w", err)
		}
		if cfg.Database.ConnectionString != "" {
			return cfg.Database.ConnectionString, nil
		}
	}

	// 3. Neither found
	return "", fmt.Errorf("no database connection configured. Set ZENZEN_DB_CONNECTION env var or create config.yaml (see config.example.yaml)")
}
