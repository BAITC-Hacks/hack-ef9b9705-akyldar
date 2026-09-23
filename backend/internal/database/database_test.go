package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "nested", "app.db")

	db, err := Open(databasePath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(databasePath); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign keys to be enabled, got %d", foreignKeys)
	}
}
