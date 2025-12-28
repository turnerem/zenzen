package storage

import (
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/turnerem/zenzen/core"
)

func TestSQLStorage_GetAll(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(nil)

	storage := &SQLStorage{
		conn: mock,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	// Mock the query
	rows := pgxmock.NewRows([]string{"id", "title", "tags", "started_at_timestamp", "ended_at_timestamp", "last_modified_timestamp", "estimated_duration", "body"}).
		AddRow("1", "K8s", []string{"learning"}, time.Time{}, time.Time{}, time.Now(), int64(0), "Test body").
		AddRow("2", "System Design", []string{"interviews"}, time.Time{}, time.Time{}, time.Now(), int64(0), "Test body 2")

	mock.ExpectQuery(`SELECT id, title, tags, started_at_timestamp, ended_at_timestamp, last_modified_timestamp, estimated_duration, body FROM entries`).
		WillReturnRows(rows)

	// Execute
	entries, err := storage.GetAll()

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	if entries["1"].Title != "K8s" {
		t.Errorf("Expected title 'K8s', got '%s'", entries["1"].Title)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSQLStorage_SaveEntry(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(nil)

	storage := &SQLStorage{
		conn: mock,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	entry := core.Entry{
		ID:    "1",
		Title: "Test Entry",
		Tags:  []string{"test"},
		Body:  "Test body",
	}

	// Mock the insert/update query - use AnyArg() for LastModifiedTimestamp since it's set dynamically
	mock.ExpectExec(`INSERT INTO entries`).
		WithArgs("1", "Test Entry", []string{"test"}, entry.StartedAtTimestamp, entry.EndedAtTimestamp, pgxmock.AnyArg(), int64(0), "Test body").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Execute
	err = storage.SaveEntry(entry)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSQLStorage_DeleteEntry(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(nil)

	storage := &SQLStorage{
		conn: mock,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	// Mock the delete query
	mock.ExpectExec(`DELETE FROM entries WHERE id = \$1`).
		WithArgs("1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// Execute
	err = storage.DeleteEntry("1")

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
