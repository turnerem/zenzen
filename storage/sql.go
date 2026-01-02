package storage

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/turnerem/zenzen/core"
)

const (
	ENTRIES_TABLE = "entries"
)

// DBConn is an interface for database connections (allows mocking)
type DBConn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Close(ctx context.Context) error
}

type SQLStorage struct {
	conn DBConn
	psql sq.StatementBuilderType
}

// NewSQLStorage creates a new SQL storage and ensures the table exists
func NewSQLStorage(ctx context.Context, connString string) (*SQLStorage, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	storage := &SQLStorage{
		conn: conn,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	// Create table if it doesn't exist
	if err := storage.createTableIfNotExists(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return storage, nil
}

// createTableIfNotExists creates the entries table if it doesn't already exist
func (s *SQLStorage) createTableIfNotExists(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS entries (
			id VARCHAR(255) PRIMARY KEY,
			title TEXT NOT NULL,
			tags TEXT[],
			started_at_timestamp TIMESTAMPTZ,
			ended_at_timestamp TIMESTAMPTZ,
			last_modified_timestamp TIMESTAMPTZ,
			estimated_duration BIGINT,
			body TEXT
		)
	`
	_, err := s.conn.Exec(ctx, query)
	return err
}

// Close closes the database connection
func (s *SQLStorage) Close(ctx context.Context) error {
	return s.conn.Close(ctx)
}

// GetAll retrieves all entries from the database
func (s *SQLStorage) GetAll() (map[string]core.Entry, error) {
	ctx := context.Background()

	query, args, err := s.psql.
		Select("id", "title", "tags", "started_at_timestamp", "ended_at_timestamp", "last_modified_timestamp", "estimated_duration", "body").
		From(ENTRIES_TABLE).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	entries := make(map[string]core.Entry)

	for rows.Next() {
		var entry core.Entry
		var tags []string
		var startedAt, endedAt, lastModified pgtype.Timestamptz
		var estimatedDuration int64

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&tags,
			&startedAt,
			&endedAt,
			&lastModified,
			&estimatedDuration,
			&entry.Body,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		entry.Tags = tags
		if startedAt.Valid {
			entry.StartedAtTimestamp = startedAt.Time
		}
		if endedAt.Valid {
			entry.EndedAtTimestamp = endedAt.Time
		}
		if lastModified.Valid {
			entry.LastModifiedTimestamp = lastModified.Time
		}
		entry.EstimatedDuration = time.Duration(estimatedDuration)

		entries[entry.ID] = entry
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entries, nil
}

// SaveEntry inserts or updates a single entry
// Note: LastModifiedTimestamp should be set by the caller before calling this method
func (s *SQLStorage) SaveEntry(entry core.Entry) error {
	ctx := context.Background()

	query, args, err := s.psql.
		Insert(ENTRIES_TABLE).
		Columns("id", "title", "tags", "started_at_timestamp", "ended_at_timestamp", "last_modified_timestamp", "estimated_duration", "body").
		Values(
			entry.ID,
			entry.Title,
			entry.Tags,
			entry.StartedAtTimestamp,
			entry.EndedAtTimestamp,
			entry.LastModifiedTimestamp,
			int64(entry.EstimatedDuration),
			entry.Body,
		).
		Suffix("ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, tags = EXCLUDED.tags, started_at_timestamp = EXCLUDED.started_at_timestamp, ended_at_timestamp = EXCLUDED.ended_at_timestamp, last_modified_timestamp = EXCLUDED.last_modified_timestamp, estimated_duration = EXCLUDED.estimated_duration, body = EXCLUDED.body").
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = s.conn.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to save entry: %w", err)
	}

	return nil
}

// DeleteEntry removes an entry from the database
func (s *SQLStorage) DeleteEntry(id string) error {
	ctx := context.Background()

	query, args, err := s.psql.
		Delete(ENTRIES_TABLE).
		Where(sq.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = s.conn.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	return nil
}
