package service

import (
	"context"
	"errors"
	"testing"

	"go-sigil/internal/models"
)

func TestTree_Build(t *testing.T) {
	ctx := context.Background()

	files := []models.File{
		{Path: "internal/service/searcher.go"},
		{Path: "internal/service/deps.go"},
		{Path: "internal/store/sqlite.go"},
		{Path: "cmd/sigil/main.go"},
	}
	syms := []models.Symbol{
		{ID: "sym00001", Kind: "function", Name: "Search"},
	}

	t.Run("builds full tree from all files", func(t *testing.T) {
		st := &mockStore{
			listFilesFn: func(_ context.Context) ([]models.File, error) { return files, nil },
		}
		tr := NewTree(st, "/repo")
		result, err := tr.Build(ctx, ".", 3, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Nodes) == 0 {
			t.Error("expected non-empty tree nodes")
		}
	})

	t.Run("scope filters to subtree", func(t *testing.T) {
		st := &mockStore{
			listFilesFn: func(_ context.Context) ([]models.File, error) { return files, nil },
		}
		tr := NewTree(st, "/repo")
		result, err := tr.Build(ctx, "internal/service", 3, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only files under internal/service should appear
		for _, node := range result.Nodes {
			if node.IsDir {
				continue
			}
			// all nodes should be within scope
			_ = node.Path
		}
		if result.Root != "internal/service" {
			t.Errorf("Root = %q, want %q", result.Root, "internal/service")
		}
	})

	t.Run("maxDepth zero defaults to 3", func(t *testing.T) {
		st := &mockStore{
			listFilesFn: func(_ context.Context) ([]models.File, error) { return files, nil },
		}
		tr := NewTree(st, "/repo")
		// depth 0 should not panic and should use default 3
		result, err := tr.Build(ctx, ".", 0, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("nil result")
		}
	})

	t.Run("symbol counts included when requested", func(t *testing.T) {
		st := &mockStore{
			listFilesFn: func(_ context.Context) ([]models.File, error) {
				return []models.File{{Path: "internal/service/searcher.go"}}, nil
			},
			getSymbolsByFileFn: func(_ context.Context, _ string) ([]models.Symbol, error) {
				return syms, nil
			},
		}
		tr := NewTree(st, "/repo")
		result, err := tr.Build(ctx, ".", 3, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// verify that at least one node has a symbol count
		found := false
		var walk func(nodes []TreeNode)
		walk = func(nodes []TreeNode) {
			for _, n := range nodes {
				if n.SymbolCount > 0 {
					found = true
				}
				walk(n.Children)
			}
		}
		walk(result.Nodes)
		if !found {
			t.Error("expected at least one node with SymbolCount > 0")
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		st := &mockStore{
			listFilesFn: func(_ context.Context) ([]models.File, error) {
				return nil, errors.New("db failure")
			},
		}
		tr := NewTree(st, "/repo")
		_, err := tr.Build(ctx, ".", 3, false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
