package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "modernc.org/sqlite"

	"github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/migrations"
)

func RunMigrations(cfg config.DBConfig) error {
	dbPath := cfg.DBName
	if dbPath == "" {
		dbPath = "multiagent.db"
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		slog.Error("Failed to open database", "error", err, "database", dbPath)
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("Failed to connect to database", "error", err, "database", dbPath)
		return err
	}

	instance, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		slog.Error("Failed to create sqlite instance", "error", err, "database", dbPath)
		return err
	}

	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		slog.Error("Failed to create migrations source", "error", err)
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", instance)
	if err != nil {
		slog.Error("Failed to create migrate instance", "error", err, "database", dbPath)
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("Failed to run migrations up", "error", err)
		return err
	}

	return nil
}
