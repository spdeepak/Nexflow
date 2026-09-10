package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/spdeepak/nexflow/internal/config"
)

func TestRunMigrationsPathWithSpace(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "Application Support", "nexflow.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DBConfig{
		DBName:            dbPath,
		MaxOpenConns:      5,
		MaxIdleConns:      5,
		ConnMaxLifetime:   time.Minute,
		ConnMaxIdleTime:   time.Minute,
		HealthCheckPeriod: time.Second,
	}

	if err := RunMigrations(cfg); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on", dbPath)
	verify, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer verify.Close()

	var n int
	if err := verify.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='model_credentials'").Scan(&n); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected model_credentials table, got %d", n)
	}
}
