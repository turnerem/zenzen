package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"sort"
	"time"
)

type Log struct {
	Title         string   `json:"Title"`
	Tags          []string `json:"Tags"`
	Start         string   `json:"Start"`
	TTCPrediction string   `json:"TTC Prediction"`
	TTCActual     string   `json:"TTC Actual"`
	Body          string   `json:"Body"`
}

type Logger struct {
	fileSystem FileSystem
}

func NewLogger(fileSystem FileSystem) *Logger {
	return &Logger{fileSystem: fileSystem}
}

func (l *Logger) List() ([]string, error) {
	dir, err := l.fileSystem.ReadDir(".")

	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, file := range dir {
		files = append(files, file.Name())
	}

	return files, nil
}

func (l *Logger) GetLog(filename string) (Log, error) {
	file, err := fs.ReadFile(l.fileSystem, filename)
	if err != nil {
		return Log{}, err
	}

	var log Log
	err = json.Unmarshal(file, &log)
	if err != nil {
		return Log{}, err
	}

	return log, nil
}

func (l *Logger) CreateOrUpdate(filename string, log Log) error {
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}
	return l.fileSystem.WriteFile(filename, data, 0644)
}

func (l *Logger) Delete(filename string) error {
	err := l.fileSystem.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetAllSorted returns all logs sorted by Start timestamp (newest first)
func (l *Logger) GetAllSorted() ([]Log, error) {
	filenames, err := l.List()
	if err != nil {
		return nil, err
	}

	var logs []Log
	for _, filename := range filenames {
		// Skip non-JSON files
		if len(filename) < 5 || filename[len(filename)-5:] != ".json" {
			continue
		}

		log, err := l.GetLog(filename)
		if err != nil {
			// Skip logs that can't be parsed
			continue
		}
		logs = append(logs, log)
	}

	// Sort by Start timestamp (newest first)
	sort.Slice(logs, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, logs[i].Start)
		timeJ, errJ := time.Parse(time.RFC3339, logs[j].Start)

		// If either can't be parsed, put them at the end
		if errI != nil || errJ != nil {
			return errI == nil
		}

		return timeI.After(timeJ) // Newest first
	})

	return logs, nil
}
