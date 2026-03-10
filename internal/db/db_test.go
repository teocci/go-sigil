package db_test

import (
	"testing"

	"go-sigil/internal/db"
)

func TestOpen_InMemory(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Verify journal mode: WAL for file DBs, "memory" for :memory: (SQLite ignores WAL in-memory)
	var mode string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" && mode != "memory" {
		t.Errorf("journal_mode = %q, want wal or memory", mode)
	}

	// Verify foreign keys enabled
	var fk int
	if err := d.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestMigrate(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// First run: apply schema
	if err := db.Run(d); err != nil {
		t.Fatalf("Run (first): %v", err)
	}

	var version int
	if err := d.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if version != 1 {
		t.Errorf("user_version = %d, want 1", version)
	}

	// Second run: idempotent
	if err := db.Run(d); err != nil {
		t.Fatalf("Run (second): %v", err)
	}

	// Verify tables exist
	tables := []string{"files", "symbols", "call_edges", "savings_log"}
	for _, tbl := range tables {
		var name string
		err := d.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}

	// Verify FTS5 virtual table
	var vtbl string
	err = d.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='symbols_fts'`).Scan(&vtbl)
	if err != nil {
		t.Errorf("FTS5 table symbols_fts not found: %v", err)
	}
}
