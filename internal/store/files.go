package store

import (
	"context"
	"database/sql"
	"fmt"

	"go-sigil/internal/models"
)

// UpsertFile inserts or replaces a file record.
func (s *SQLiteStore) UpsertFile(ctx context.Context, f models.File) error {
	const q = `
INSERT INTO files (path, blob_sha, mtime, size, last_indexed)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
  blob_sha     = excluded.blob_sha,
  mtime        = excluded.mtime,
  size         = excluded.size,
  last_indexed = excluded.last_indexed`

	_, err := s.db.ExecContext(ctx, q,
		f.Path,
		nullableString(f.BlobSHA),
		nullableString(f.Mtime),
		f.Size,
		f.LastIndexed,
	)
	if err != nil {
		return fmt.Errorf("upsert file %s: %w", f.Path, err)
	}
	return nil
}

// GetFile returns the file record for path, or nil if not found.
func (s *SQLiteStore) GetFile(ctx context.Context, path string) (*models.File, error) {
	const q = `SELECT path, blob_sha, mtime, size, last_indexed FROM files WHERE path = ?`

	row := s.db.QueryRowContext(ctx, q, path)
	f, err := scanFile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get file %s: %w", path, err)
	}
	return f, nil
}

// DeleteFile removes a file record. Symbol cascade is handled by the DB FK constraint.
func (s *SQLiteStore) DeleteFile(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM files WHERE path = ?`, path)
	if err != nil {
		return fmt.Errorf("delete file %s: %w", path, err)
	}
	return nil
}

// ListFiles returns all tracked files.
func (s *SQLiteStore) ListFiles(ctx context.Context) ([]models.File, error) {
	const q = `SELECT path, blob_sha, mtime, size, last_indexed FROM files ORDER BY path`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, *f)
	}
	return files, rows.Err()
}

type fileScanner interface {
	Scan(dest ...any) error
}

func scanFile(s fileScanner) (*models.File, error) {
	var f models.File
	var blobSHA, mtime sql.NullString
	var size sql.NullInt64

	if err := s.Scan(&f.Path, &blobSHA, &mtime, &size, &f.LastIndexed); err != nil {
		return nil, err
	}

	f.BlobSHA = scanNullableString(blobSHA)
	f.Mtime = scanNullableString(mtime)
	if size.Valid {
		f.Size = size.Int64
	}

	return &f, nil
}
