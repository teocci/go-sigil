package service

import (
	"context"
	"errors"
	"testing"

	"go-sigil/internal/models"
	"go-sigil/internal/store"
)

func TestSearcher_Search(t *testing.T) {
	ctx := context.Background()

	sym1 := models.Symbol{ID: "aabbccdd", Kind: "function", Name: "Foo", Language: "go", File: "pkg/foo.go"}
	sym2 := models.Symbol{ID: "11223344", Kind: "method", Name: "Bar", Language: "go", File: "pkg/bar.go"}

	tests := []struct {
		name        string
		query       string
		opts        store.SearchOptions
		storeFn     func(ctx context.Context, q string, opts store.SearchOptions) ([]models.Symbol, error)
		wantTotal   int
		wantErr     bool
	}{
		{
			name:  "returns matching symbols",
			query: "Foo",
			storeFn: func(_ context.Context, q string, _ store.SearchOptions) ([]models.Symbol, error) {
				if q == "Foo" {
					return []models.Symbol{sym1}, nil
				}
				return nil, nil
			},
			wantTotal: 1,
		},
		{
			name:  "empty results",
			query: "NotFound",
			storeFn: func(_ context.Context, _ string, _ store.SearchOptions) ([]models.Symbol, error) {
				return nil, nil
			},
			wantTotal: 0,
		},
		{
			name:  "multiple results",
			query: "pkg",
			storeFn: func(_ context.Context, _ string, _ store.SearchOptions) ([]models.Symbol, error) {
				return []models.Symbol{sym1, sym2}, nil
			},
			wantTotal: 2,
		},
		{
			name:    "store error propagates",
			query:   "x",
			storeFn: func(_ context.Context, _ string, _ store.SearchOptions) ([]models.Symbol, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
		{
			name:  "kind filter passed through",
			query: "Foo",
			opts:  store.SearchOptions{Kind: "function"},
			storeFn: func(_ context.Context, _ string, opts store.SearchOptions) ([]models.Symbol, error) {
				if opts.Kind != "function" {
					t.Errorf("expected kind filter 'function', got %q", opts.Kind)
				}
				return []models.Symbol{sym1}, nil
			},
			wantTotal: 1,
		},
		{
			name:  "language filter passed through",
			query: "Bar",
			opts:  store.SearchOptions{Language: "go"},
			storeFn: func(_ context.Context, _ string, opts store.SearchOptions) ([]models.Symbol, error) {
				if opts.Language != "go" {
					t.Errorf("expected language filter 'go', got %q", opts.Language)
				}
				return []models.Symbol{sym2}, nil
			},
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &mockStore{searchSymbolsFn: tt.storeFn}
			s := NewSearcher(st)

			result, err := s.Search(ctx, tt.query, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if result.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", result.Total, tt.wantTotal)
			}
			if len(result.Symbols) != tt.wantTotal {
				t.Errorf("len(Symbols) = %d, want %d", len(result.Symbols), tt.wantTotal)
			}
		})
	}
}
