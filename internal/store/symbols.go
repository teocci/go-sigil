package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go-sigil/internal/models"
)

const symbolColumns = `
  id, kind, name, qualified_name, language, file, package_root,
  byte_start, byte_end, line_start, line_end,
  signature, summary, tags, parent_id, children, depth, imports,
  content_hash, possible_unresolved, untracked, indexed_at`

const insertSymbolSQL = `
INSERT INTO symbols (
  id, kind, name, qualified_name, language, file, package_root,
  byte_start, byte_end, line_start, line_end,
  signature, summary, tags, parent_id, children, depth, imports,
  content_hash, possible_unresolved, untracked, indexed_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

// ReplaceFileSymbols atomically replaces all symbols for a file.
// Existing symbols are deleted (cascades to call_edges), then the new
// set is inserted, all within a single IMMEDIATE transaction.
func (s *SQLiteStore) ReplaceFileSymbols(ctx context.Context, file string, symbols []models.Symbol) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM symbols WHERE file = ?`, file); err != nil {
			return fmt.Errorf("delete symbols for %s: %w", file, err)
		}

		if len(symbols) == 0 {
			return nil
		}

		stmt, err := tx.PrepareContext(ctx, insertSymbolSQL)
		if err != nil {
			return fmt.Errorf("prepare symbol insert: %w", err)
		}
		defer stmt.Close()

		now := time.Now().UTC().Format(time.RFC3339)
		for _, sym := range symbols {
			indexedAt := sym.IndexedAt
			if indexedAt == "" {
				indexedAt = now
			}
			_, err := stmt.ExecContext(ctx,
				sym.ID, sym.Kind, sym.Name, sym.QualifiedName,
				sym.Language, sym.File,
				nullableString(sym.PackageRoot),
				nullableInt(sym.ByteStart), nullableInt(sym.ByteEnd),
				sym.LineStart, sym.LineEnd,
				nullableString(sym.Signature),
				nullableString(sym.Summary),
				nullableString(sym.Tags),
				nullableString(sym.ParentID),
				nullableString(sym.Children),
				sym.Depth,
				nullableString(sym.Imports),
				nullableString(sym.ContentHash),
				boolToInt(sym.PossibleUnresolved),
				boolToInt(sym.Untracked),
				indexedAt,
			)
			if err != nil {
				return fmt.Errorf("insert symbol %s: %w", sym.ID, err)
			}
		}
		return nil
	})
}

// GetSymbolByID retrieves a symbol by ID. Returns nil if not found.
func (s *SQLiteStore) GetSymbolByID(ctx context.Context, id string) (*models.Symbol, error) {
	q := `SELECT ` + symbolColumns + ` FROM symbols WHERE id = ?`
	row := s.db.QueryRowContext(ctx, q, id)
	sym, err := scanSymbol(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get symbol %s: %w", id, err)
	}
	return sym, nil
}

// GetSymbolsByFile returns all symbols for a file, ordered by line.
func (s *SQLiteStore) GetSymbolsByFile(ctx context.Context, file string) ([]models.Symbol, error) {
	q := `SELECT ` + symbolColumns + ` FROM symbols WHERE file = ? ORDER BY line_start`
	rows, err := s.db.QueryContext(ctx, q, file)
	if err != nil {
		return nil, fmt.Errorf("get symbols for file %s: %w", file, err)
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// GetSymbolsByIDs retrieves symbols for a set of IDs. Missing IDs are skipped.
func (s *SQLiteStore) GetSymbolsByIDs(ctx context.Context, ids []string) ([]models.Symbol, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	q := `SELECT ` + symbolColumns + ` FROM symbols WHERE id IN (` + placeholders + `)`

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("get symbols by IDs: %w", err)
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// MarkPossibleUnresolved flags all symbols in a file.
func (s *SQLiteStore) MarkPossibleUnresolved(ctx context.Context, file string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE symbols SET possible_unresolved = 1 WHERE file = ?`, file)
	if err != nil {
		return fmt.Errorf("mark possible unresolved for %s: %w", file, err)
	}
	return nil
}

// --- scan helpers ---

type symbolScanner interface {
	Scan(dest ...any) error
}

func scanSymbol(s symbolScanner) (*models.Symbol, error) {
	var sym models.Symbol
	var (
		packageRoot, signature, summary, tags sql.NullString
		parentID, children, imports           sql.NullString
		contentHash                           sql.NullString
		byteStart, byteEnd                    sql.NullInt64
		possibleUnresolved, untracked         sql.NullInt64
	)

	err := s.Scan(
		&sym.ID, &sym.Kind, &sym.Name, &sym.QualifiedName,
		&sym.Language, &sym.File,
		&packageRoot,
		&byteStart, &byteEnd,
		&sym.LineStart, &sym.LineEnd,
		&signature, &summary, &tags,
		&parentID, &children,
		&sym.Depth,
		&imports,
		&contentHash,
		&possibleUnresolved, &untracked,
		&sym.IndexedAt,
	)
	if err != nil {
		return nil, err
	}

	sym.PackageRoot = scanNullableString(packageRoot)
	sym.ByteStart = scanNullableInt(byteStart)
	sym.ByteEnd = scanNullableInt(byteEnd)
	sym.Signature = scanNullableString(signature)
	sym.Summary = scanNullableString(summary)
	sym.Tags = scanNullableString(tags)
	sym.ParentID = scanNullableString(parentID)
	sym.Children = scanNullableString(children)
	sym.Imports = scanNullableString(imports)
	sym.ContentHash = scanNullableString(contentHash)
	sym.PossibleUnresolved = possibleUnresolved.Int64 != 0
	sym.Untracked = untracked.Int64 != 0

	return &sym, nil
}

func scanSymbols(rows *sql.Rows) ([]models.Symbol, error) {
	var syms []models.Symbol
	for rows.Next() {
		sym, err := scanSymbol(rows)
		if err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		syms = append(syms, *sym)
	}
	return syms, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
