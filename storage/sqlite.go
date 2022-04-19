package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/fenole/szmaterlok/service"

	_ "embed"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	mtx *sync.Mutex
	db  *sql.DB
}

// NewSQLiteStorage opens and migrates storage from given path.
func NewSQLiteStorage(ctx context.Context, path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if err := migrateSQLite(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("failed to enable wal mode: %w", err)
	}

	return &SQLiteStorage{
		db:  db,
		mtx: &sync.Mutex{},
	}, nil
}

//go:embed sqlite_store_event.sql
var storeEventQuery string

// StoreEvent stores given bridge event in sqlite event storage.
func (s *SQLiteStorage) StoreEvent(ctx context.Context, evt service.BridgeEvent) error {
	headers, err := json.Marshal(evt.Headers)
	if err != nil {
		return fmt.Errorf("failed to encode headers as json: %w", err)
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, err = s.db.ExecContext(
		ctx,
		storeEventQuery,
		sql.Named("id", evt.ID),
		sql.Named("type", evt.Name),
		sql.Named("headers", headers),
		sql.Named("createdat", evt.CreatedAt),
		sql.Named("data", evt.Data),
	)
	if err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	return nil
}

//go:embed sqlite_events.sql
var eventsQuery string

// Events sends all events from state archive through given channels
// grouped by their creation date.
func (s *SQLiteStorage) Events(ctx context.Context, c chan<- service.BridgeEvent) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	rows, err := s.db.QueryContext(ctx, eventsQuery)
	if err != nil {
		return fmt.Errorf("failed to create query: %w", err)
	}
	defer rows.Close()

	var rawEvent struct {
		name      string
		id        string
		headers   []byte
		data      []byte
		createdAt int64
	}

	for rows.Next() {
		if err := rows.Scan(
			&rawEvent.id,
			&rawEvent.name,
			&rawEvent.createdAt,
			&rawEvent.headers,
			&rawEvent.data,
		); err != nil {
			return fmt.Errorf("failed to scan event: %w", err)
		}

		headers := service.BridgeHeaders{}
		if err := json.Unmarshal(rawEvent.headers, &headers); err != nil {
			return fmt.Errorf("failed to parse event headers: %w", err)
		}

		c <- service.BridgeEvent{
			Name:      service.BridgeEventType(rawEvent.name),
			ID:        rawEvent.id,
			Headers:   headers,
			CreatedAt: rawEvent.createdAt,
			Data:      slices.Clone(rawEvent.data),
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration failure: %w", err)
	}

	return nil
}
