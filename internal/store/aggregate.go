package store

import (
	"context"
	"fmt"

	"go-sigil/internal/models"
)

// CountSymbols returns the total number of symbols indexed.
func (s *SQLiteStore) CountSymbols(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbols").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count symbols: %w", err)
	}
	return n, nil
}

// CountFiles returns the total number of tracked files.
func (s *SQLiteStore) CountFiles(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count files: %w", err)
	}
	return n, nil
}

// CountPossibleUnresolved returns symbols flagged as possibly unresolved.
func (s *SQLiteStore) CountPossibleUnresolved(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbols WHERE possible_unresolved = 1").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count possible unresolved: %w", err)
	}
	return n, nil
}

// GetLanguageStats returns aggregate symbol/file counts per language.
func (s *SQLiteStore) GetLanguageStats(ctx context.Context) ([]models.LanguageStat, error) {
	const q = `
SELECT s.language, COUNT(DISTINCT s.file), COUNT(*)
FROM symbols s
GROUP BY s.language
ORDER BY COUNT(*) DESC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query language stats: %w", err)
	}
	defer rows.Close()

	var stats []models.LanguageStat
	for rows.Next() {
		var st models.LanguageStat
		if err := rows.Scan(&st.Language, &st.Files, &st.Symbols); err != nil {
			return nil, fmt.Errorf("scan language stat: %w", err)
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

// GetPackageStats returns aggregate symbol counts per package_root.
func (s *SQLiteStore) GetPackageStats(ctx context.Context) ([]models.PackageStat, error) {
	const q = `
SELECT COALESCE(package_root, ''), COUNT(*)
FROM symbols
WHERE package_root IS NOT NULL AND package_root != ''
GROUP BY package_root
ORDER BY COUNT(*) DESC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query package stats: %w", err)
	}
	defer rows.Close()

	var stats []models.PackageStat
	for rows.Next() {
		var st models.PackageStat
		if err := rows.Scan(&st.Root, &st.Symbols); err != nil {
			return nil, fmt.Errorf("scan package stat: %w", err)
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

// ListSessions returns all unique session IDs with aggregated savings.
func (s *SQLiteStore) ListSessions(ctx context.Context) ([]models.SavingsSession, error) {
	const q = `
SELECT session_id, SUM(tokens_saved), COUNT(*), MIN(call_at), MAX(call_at)
FROM savings_log
GROUP BY session_id
ORDER BY MAX(call_at) DESC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.SavingsSession
	for rows.Next() {
		var ss models.SavingsSession
		if err := rows.Scan(&ss.SessionID, &ss.TokensSaved, &ss.CallCount, &ss.FirstCall, &ss.LastCall); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, ss)
	}
	return sessions, rows.Err()
}

// GetTopSessions returns the top N sessions by tokens_saved.
func (s *SQLiteStore) GetTopSessions(ctx context.Context, n int) ([]models.SavingsSession, error) {
	const q = `
SELECT session_id, SUM(tokens_saved), COUNT(*), MIN(call_at), MAX(call_at)
FROM savings_log
GROUP BY session_id
ORDER BY SUM(tokens_saved) DESC
LIMIT ?`

	rows, err := s.db.QueryContext(ctx, q, n)
	if err != nil {
		return nil, fmt.Errorf("top sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.SavingsSession
	for rows.Next() {
		var ss models.SavingsSession
		if err := rows.Scan(&ss.SessionID, &ss.TokensSaved, &ss.CallCount, &ss.FirstCall, &ss.LastCall); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, ss)
	}
	return sessions, rows.Err()
}
