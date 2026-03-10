package store

import (
	"context"
	"fmt"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
)

// SearchSymbols performs a FTS5 full-text search with optional kind/language/file filters.
// query uses FTS5 syntax (prefix search: "Parse*", phrase: "\"parse file\"", etc.)
func (s *SQLiteStore) SearchSymbols(ctx context.Context, query string, opts SearchOptions) ([]models.Symbol, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = constants.DefaultSearchLimit
	}

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
		query,
		opts.Kind, opts.Kind,
		opts.Language, opts.Language,
		opts.File, opts.File,
		limit, opts.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search symbols %q: %w", query, err)
	}
	defer rows.Close()
	return scanSymbols(rows)
}
