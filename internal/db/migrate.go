package db

import (
	"database/sql"
	"fmt"

	"go-sigil/internal/constants"
)

// Run applies any pending schema migrations.
// Uses PRAGMA user_version to track the applied version.
func Run(db *sql.DB) error {
	if err := checkFTS5(db); err != nil {
		return err
	}

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if version == constants.SchemaVersion {
		return nil // already up to date
	}

	if version > constants.SchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported %d", version, constants.SchemaVersion)
	}

	// version == 0: fresh database — apply V1DDL
	if _, err := db.Exec(V1DDL); err != nil {
		return fmt.Errorf("apply schema v1: %w", err)
	}

	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", constants.SchemaVersion)); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}

	return nil
}

// checkFTS5 verifies that FTS5 is available in this SQLite build.
// modernc.org/sqlite includes FTS5 by default.
func checkFTS5(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS _fts5_probe USING fts5(x)`)
	if err != nil {
		return fmt.Errorf("fts5 not available in SQLite build: %w", err)
	}
	_, _ = db.Exec(`DROP TABLE IF EXISTS _fts5_probe`)
	return nil
}
