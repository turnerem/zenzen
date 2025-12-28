package service

import (
	"reflect"
	"testing"
	"time"

	"github.com/turnerem/zenzen/core"
)

const (
	DataK8s = `{
	"Title": "K8s",
	"Tags": ["learning", "open-source"],
	"Predicted Duration": "6d",
	"Duration": "",
	"Body": "The journey has just begun."
}`
	DataSystemDesign = `{
	"Title": "System Design",
	"Tags": ["interviews"],
	"Predicted Duration": "7d",
	"Duration": "8d",
	"Body": "Books combined with youtube resources were very helpful."
}`
)

type MockStore struct {
}

var (
	k8sLog = core.Entry{
		ID:                "1",
		Title:             "K8s",
		Tags:              []string{"learning", "open-source"},
		EstimatedDuration: time.Hour * 3,
		StartedAt:         time.Date(2025, 12, 20, 10, 0, 0, 0, time.UTC),
		Body:              "The journey has just begun.",
	}
	systemDesignLog = core.Entry{
		ID:                "2",
		Title:             "System Design",
		Tags:              []string{"interviews"},
		EstimatedDuration: time.Hour * 4,
		StartedAt:         time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
		EndedAt:           time.Date(2026, 1, 10, 17, 4, 5, 0, time.UTC),
		Body:              "Books combined with youtube resources were very helpful.",
	}
)

func (m *MockStore) GetAll() (map[string]core.Entry, error) {
	return map[string]core.Entry{
		"1": k8sLog,
		"2": systemDesignLog,
	}, nil
}

func (m *MockStore) SaveEntry(entry core.Entry) error {
	// Mock save - does nothing
	return nil
}

func (m *MockStore) DeleteEntry(id string) error {
	// Mock delete - does nothing
	return nil
}

func TestLoadAll(t *testing.T) {
	t.Run("get list", func(t *testing.T) {
		notes := NewNotes(&MockStore{})
		err := notes.LoadAll()

		assertNilError(t, err)

		want := map[string]core.Entry{
			"1": k8sLog,
			"2": systemDesignLog,
		}

		assertEquality(t, notes.Entries, want)
	})
}

func TestDelete(t *testing.T) {
	t.Run("delete existing log", func(t *testing.T) {
		notes := NewNotes(&MockStore{})
		err := notes.LoadAll()
		assertNilError(t, err)

		notes.Delete("1")

		want := map[string]core.Entry{
			"2": systemDesignLog,
		}

		assertEquality(t, notes.Entries, want)

	})

	t.Run("delete non-existing log", func(t *testing.T) {
		notes := NewNotes(&MockStore{})
		err := notes.LoadAll()
		assertNilError(t, err)

		notes.Delete("non-existing-id")

		want := map[string]core.Entry{
			"1": k8sLog,
			"2": systemDesignLog,
		}

		assertEquality(t, notes.Entries, want)
	})
}

func assertEquality[V any](t *testing.T, got, want V) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v but got %v", want, got)
	}
}

func assertNilError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
	}
}
