package logger

import (
	"io"
	"log"
	"os"
)

// SetupLogger configures logging based on the mode
func SetupLogger(mode string) (*os.File, error) {
	switch mode {
	case "tui":
		// TUI mode: Log to file to avoid interfering with display
		logFile, err := os.OpenFile("zenzen.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		log.SetOutput(logFile)
		log.SetPrefix("[TUI] ")
		return logFile, nil

	case "api":
		// API mode: Log to stdout (normal server logs)
		log.SetOutput(os.Stdout)
		log.SetPrefix("[API] ")
		return nil, nil

	case "sync":
		// Sync mode: Log to stdout (want to see progress)
		log.SetOutput(os.Stdout)
		log.SetPrefix("[SYNC] ")
		return nil, nil

	case "setup":
		// Setup mode: Log to stdout
		log.SetOutput(os.Stdout)
		log.SetPrefix("[SETUP] ")
		return nil, nil

	default:
		// Default: stdout
		log.SetOutput(os.Stdout)
		return nil, nil
	}
}

// Disable disables all logging (writes to /dev/null)
func Disable() {
	log.SetOutput(io.Discard)
}

// Enable re-enables logging to stdout
func Enable() {
	log.SetOutput(os.Stdout)
}
