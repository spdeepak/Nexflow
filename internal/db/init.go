package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "modernc.org/sqlite"

	"github.com/spdeepak/nexflow/internal/config"
)

func Connect(dbCfg config.DBConfig) *sql.DB {
	dbPath := dbCfg.DBName
	if dbPath == "" {
		dbPath = "multiagent.db"
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		slog.Error("error opening database", "error", err)
		os.Exit(1)
		return nil
	}

	db.SetMaxOpenConns(dbCfg.MaxOpenConns)
	db.SetMaxIdleConns(dbCfg.MaxIdleConns)
	db.SetConnMaxLifetime(dbCfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(dbCfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		slog.Error("error connecting to database", "error", err)
		os.Exit(1)
		return nil
	}

	return db
}
