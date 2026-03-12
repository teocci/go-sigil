package store

import (
	"context"
	"fmt"
	"strings"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
)

// normalizeFTSQuery appends a wildcard suffix to plain word queries so that
// partial terms match (e.g. "saving" matches "savings", "savings_service").
// Queries that already contain FTS5 operators (* " - OR AND NOT) are passed through unchanged.
func normalizeFTSQuery(q string) string {
	if q == "" {
		return q
	}
	// Already contains FTS5 operators — pass through as-is.
	if strings.ContainsAny(q, `*"-`) {
		return q
	}
	upper := strings.ToUpper(q)
	if strings.Contains(upper, " OR ") || strings.Contains(upper, " AND ") || strings.Contains(upper, " NOT ") {
		return q
	}
	return q + "*"
}

// SearchSymbols performs a FTS5 full-text search with optional kind/language/file filters.
// query uses FTS5 syntax (prefix search: "Parse*", phrase: "\"parse file\"", etc.).
// Plain word queries are automatically given a wildcard suffix for partial matching.
func (s *SQLiteStore) SearchSymbols(ctx context.Context, query string, opts SearchOptions) ([]models.Symbol, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = constants.DefaultSearchLimit
	}

	ftsQuery := normalizeFTSQuery(query)

	const q = `
SELECT s.id, s.kind, s.name, s.qualified_name, s.language, s.file, s.package_root,
       s.byte_start, s.byte_end, s.line_start, s.line_end,
       s.signature, s.summary, s.tags, s.parent_id, s.children, s.depth, s.imports,
       s.content_hash, s.possible_unresolved, s.untracked, s.indexed_at
FROM symbols s
JOIN symbols_fts fts ON s.rowid = fts.rowid
WHERE symbols_fts MATCH ?
  AND (? = '' OR s.kind = ?)
  AND (? = '' OR s.language = ?)
  AND (? = '' OR s.file = ?)
ORDER BY rank
LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, q,
		ftsQuery,
		opts.Kind, opts.Kind,
		opts.Language, opts.Language,
		opts.File, opts.File,
		limit, opts.Offset,
	)
	if err != nil {
		// If FTS5 rejects the query (e.g. trailing operator), fall back to LIKE search.
		return s.searchSymbolsLike(ctx, query, opts, limit)
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// searchSymbolsLike is a fallback for when FTS5 rejects the query syntax.
// It performs a case-insensitive LIKE search on name and qualified_name.
func (s *SQLiteStore) searchSymbolsLike(ctx context.Context, query string, opts SearchOptions, limit int) ([]models.Symbol, error) {
	pattern := "%" + query + "%"
	const q = `
SELECT id, kind, name, qualified_name, language, file, package_root,
       byte_start, byte_end, line_start, line_end,
       signature, summary, tags, parent_id, children, depth, imports,
       content_hash, possible_unresolved, untracked, indexed_at
FROM symbols
WHERE (name LIKE ? OR qualified_name LIKE ?)
  AND (? = '' OR kind = ?)
  AND (? = '' OR language = ?)
  AND (? = '' OR file = ?)
ORDER BY name
LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, q,
		pattern, pattern,
		opts.Kind, opts.Kind,
		opts.Language, opts.Language,
		opts.File, opts.File,
		limit, opts.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search symbols (fallback) %q: %w", query, err)
	}
	defer rows.Close()
	return scanSymbols(rows)
}
