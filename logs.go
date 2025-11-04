package main

import (
	"encoding/json"
	"io/fs"
	"os"
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
