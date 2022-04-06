package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/fenole/szmaterlok/service"

	_ "embed"
	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	mtx *sync.Mutex
	db  *sql.DB
}

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
