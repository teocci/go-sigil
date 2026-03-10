package service

import (
	"context"
	"fmt"

	"go-sigil/internal/models"
	"go-sigil/internal/store"
)

// SearchResult holds results from a symbol search.
type SearchResult struct {
	Symbols []models.Symbol `json:"symbols"`
	Total   int             `json:"total"`
}

// Searcher performs FTS5 symbol searches.
type Searcher struct {
	st store.SymbolStore
}

// NewSearcher creates a Searcher backed by the given store.
func NewSearcher(st store.SymbolStore) *Searcher {
	return &Searcher{st: st}
}

// Search finds symbols matching query with optional filters.
func (s *Searcher) Search(ctx context.Context, query string, opts store.SearchOptions) (*SearchResult, error) {
	symbols, err := s.st.SearchSymbols(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("search symbols: %w", err)
	}
	return &SearchResult{Symbols: symbols, Total: len(symbols)}, nil
}
