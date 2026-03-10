package store

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLiteStore implements SymbolStore backed by a SQLite database.
// It is safe for concurrent use — SQLite WAL mode allows many parallel readers
// and one writer, and all writes use BEGIN IMMEDIATE transactions.
type SQLiteStore struct {
	db *sql.DB
}

// New wraps an open *sql.DB as a SymbolStore.
// The caller is responsible for running migrations before calling New.
func New(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// withTx executes fn inside a transaction.
// If fn returns an error the transaction is rolled back; otherwise it is committed.
func (s *SQLiteStore) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// nullableString converts an empty string to nil for nullable TEXT columns.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nullableInt converts a nil pointer to nil for nullable INTEGER columns.
func nullableInt(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// scanNullableString reads a nullable TEXT column into a string.
func scanNullableString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// scanNullableInt reads a nullable INTEGER column into a *int.
func scanNullableInt(ni sql.NullInt64) *int {
	if ni.Valid {
		v := int(ni.Int64)
		return &v
	}
	return nil
}
