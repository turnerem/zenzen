package logger

import (
	"io"
	"log/slog"
	"os"
)

var (
	// Logger is the global structured logger
	Logger *slog.Logger
)

// SetupLogger configures logging based on the mode
func SetupLogger(mode string) (*os.File, error) {
	var writer io.Writer
	var logFile *os.File
	var err error

	switch mode {
	case "tui":
		// TUI mode: Log to file to avoid interfering with display
		logFile, err = os.OpenFile("zenzen.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writer = logFile

		// Structured JSON logging for file (easier to parse/query)
		Logger = slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		Logger = Logger.With("mode", "tui")

		return logFile, nil

	case "api":
		// API mode: Log to stdout (production-ready JSON)
		writer = os.Stdout

		Logger = slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		Logger = Logger.With("mode", "api")

		return nil, nil

	case "sync":
		// Sync mode: Log to stdout (human-readable for CLI)
		writer = os.Stdout

		Logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		Logger = Logger.With("mode", "sync")

		return nil, nil

	case "setup":
		// Setup mode: Log to stdout (human-readable for CLI)
		writer = os.Stdout

		Logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		Logger = Logger.With("mode", "setup")

		return nil, nil

	default:
		// Default: stdout with text format
		writer = os.Stdout

		Logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		return nil, nil
	}
}

// Disable disables all logging (writes to /dev/null)
func Disable() {
	Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Enable re-enables logging to stdout
func Enable() {
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Helper functions for common log patterns

// Info logs an informational message with structured fields
func Info(msg string, args ...any) {
	if Logger != nil {
		Logger.Info(msg, args...)
	}
}

// Error logs an error message with structured fields
func Error(msg string, args ...any) {
	if Logger != nil {
		Logger.Error(msg, args...)
	}
}

// Warn logs a warning message with structured fields
func Warn(msg string, args ...any) {
	if Logger != nil {
		Logger.Warn(msg, args...)
	}
}

// Debug logs a debug message with structured fields
func Debug(msg string, args ...any) {
	if Logger != nil {
		Logger.Debug(msg, args...)
	}
}
