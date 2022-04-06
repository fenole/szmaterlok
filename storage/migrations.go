package storage

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "modernc.org/sqlite"
)

const currentVersion = 1

//go:embed sqlite_migrations
var sqliteMigrations embed.FS

func migrateSQLite(db *sql.DB) error {
	sourceInstance, err := iofs.New(sqliteMigrations, "sqlite_migrations")
	if err != nil {
		return fmt.Errorf("invalid source instance, %w", err)
	}

	targetInstance, err := sqlite.WithInstance(db, new(sqlite.Config))
	if err != nil {
		return fmt.Errorf("invalid target sqlite instance, %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs", sourceInstance, "sqlite", targetInstance)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance, %w", err)
	}

	err = m.Migrate(currentVersion)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return sourceInstance.Close()
}
