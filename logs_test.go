package main

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

const (
	DataK8s = `{
	"Title": "K8s",
	"Tags": ["learning", "open-source"],
	"TTC Prediction": "6d",
	"TTC Actual": "",
	"Body": "The journey has just begun."
}`
	DataSystemDesign = `{
	"Title": "System Design",
	"Tags": ["interviews"],
	"TTC Prediction": "7d",
	"TTC Actual": "8d",
	"Body": "Books combined with youtube resources were very helpful."
}`
)

type MockFileSystem struct {
	fstest.MapFS
	files map[string][]byte
}

func (m *MockFileSystem) Open(name string) (fs.File, error) {
	return m.MapFS.Open(name)
}

func (m *MockFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return m.MapFS.ReadDir(name)
}

func (m *MockFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.files[name] = data
	return nil
}

func (m *MockFileSystem) Remove(name string) error {
	delete(m.files, name)
	return nil
}

func TestList(t *testing.T) {
	t.Run("get list", func(t *testing.T) {
		testFS := fstest.MapFS{
			"K8s.json":           {Data: []byte(DataK8s)},
			"System Design.json": {Data: []byte(DataSystemDesign)},
		}

		logger := NewLogger(&MockFileSystem{MapFS: testFS, files: make(map[string][]byte)})
		got, err := logger.List()

		assertNilError(t, err)

		want := []string{"K8s.json", "System Design.json"}

		assertEquality(t, got, want)
	})
}

func TestGetLog(t *testing.T) {
	t.Run("get file", func(t *testing.T) {
		testFS := fstest.MapFS{
			"K8s.json":           {Data: []byte(DataK8s)},
			"System Design.json": {Data: []byte(DataSystemDesign)},
		}

		logger := NewLogger(&MockFileSystem{MapFS: testFS, files: make(map[string][]byte)})
		got, err := logger.GetLog("System Design.json")

		want := Log{
			Title:         "System Design",
			Tags:          []string{"interviews"},
			TTCPrediction: "7d",
			TTCActual:     "8d",
			Body:          "Books combined with youtube resources were very helpful.",
		}

		assertNilError(t, err)

		assertEquality(t, got, want)
	})

	t.Run("errors for non-existent file", func(t *testing.T) {
		testFS := fstest.MapFS{
			"K8s.json": {Data: []byte(DataK8s)},
		}

		logger := NewLogger(&MockFileSystem{MapFS: testFS, files: make(map[string][]byte)})
		got, err := logger.GetLog("K7s.json")

		assertError(t, err)

		assertEquality(t, got, Log{})
	})
}

func TestCreateOrUpdate(t *testing.T) {
	t.Run("create new log", func(t *testing.T) {
		mockSystem := MockFileSystem{files: make(map[string][]byte)}
		logger := NewLogger(&mockSystem)

		filename := "Bug Fix.json"
		log := Log{Body: "Fixed a bug"}
		err := logger.CreateOrUpdate(filename, log)

		assertNilError(t, err)

		got, ok := mockSystem.files[filename]
		if !ok {
			t.Error("expected file to be created")
		}

		var gotLog Log
		err = json.Unmarshal(got, &gotLog)
		assertNilError(t, err)

		assertEquality(t, gotLog, log)
	})

	t.Run("update log", func(t *testing.T) {
		filename := "Bug Fix.json"
		log := Log{Body: "Fixed a bug"}
		data, err := json.Marshal(log)
		assertNilError(t, err)

		mockSystem := MockFileSystem{files: map[string][]byte{
			filename: data,
		}}
		logger := NewLogger(&mockSystem)

		updatedLog := Log{Body: "Fixed a concurrency bug"}
		err = logger.CreateOrUpdate(filename, updatedLog)
		assertNilError(t, err)

		got, ok := mockSystem.files[filename]
		if !ok {
			t.Error("expected file to be created")
		}

		var gotLog Log
		err = json.Unmarshal(got, &gotLog)
		assertNilError(t, err)

		assertEquality(t, gotLog, updatedLog)
	})
}

func TestDelete(t *testing.T) {
	t.Run("remove file", func(t *testing.T) {
		mockSystem := MockFileSystem{files: make(map[string][]byte)}
		logger := NewLogger(&mockSystem)

		filename := "Bug Fix.json"
		log := Log{Body: "Fixed a bug"}
		err := logger.CreateOrUpdate(filename, log)

		assertNilError(t, err)

		err = logger.Delete(filename)

		assertNilError(t, err)

		_, ok := mockSystem.files[filename]
		if ok {
			t.Error("expected file to be deleted")
		}
	})

	// cannot test this with current mock setup
	// t.Run("should not error if attempted to delete file that does not exist", func(t *testing.T) {
	// })
}

func assertEquality[V any](t *testing.T, got, want V) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %v but got %v", want, got)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Error("expected error")
	}
}

func assertNilError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
	}
}
