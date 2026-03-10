package service

import (
	"context"
	"errors"
	"testing"

	"go-sigil/internal/models"
	"go-sigil/internal/storage"
)

func TestOverview_Summary(t *testing.T) {
	ctx := context.Background()

	langStats := []models.LanguageStat{
		{Language: "go", Files: 10, Symbols: 100},
		{Language: "typescript", Files: 5, Symbols: 40},
	}
	pkgStats := []models.PackageStat{
		{Root: "internal/service", Symbols: 50},
	}
	files := []models.File{
		{Path: "internal/foo.go"},
		{Path: "cmd/main.go"},
		{Path: "internal/bar.go"},
	}

	t.Run("aggregates store counts", func(t *testing.T) {
		st := &mockStore{
			countSymbolsFn:            func(_ context.Context) (int, error) { return 140, nil },
			countFilesFn:              func(_ context.Context) (int, error) { return 15, nil },
			countPossibleUnresolvedFn: func(_ context.Context) (int, error) { return 3, nil },
			getLanguageStatsFn:        func(_ context.Context) ([]models.LanguageStat, error) { return langStats, nil },
			getPackageStatsFn:         func(_ context.Context) ([]models.PackageStat, error) { return pkgStats, nil },
			listFilesFn:               func(_ context.Context) ([]models.File, error) { return files, nil },
		}
		svc := NewOverview(st, nil)
		result, err := svc.Summary(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.TotalSymbols != 140 {
			t.Errorf("TotalSymbols = %d, want 140", result.TotalSymbols)
		}
		if result.TotalFiles != 15 {
			t.Errorf("TotalFiles = %d, want 15", result.TotalFiles)
		}
		if result.PossibleUnresolvedCount != 3 {
			t.Errorf("PossibleUnresolvedCount = %d, want 3", result.PossibleUnresolvedCount)
		}
		if len(result.Languages) != 2 {
			t.Errorf("len(Languages) = %d, want 2", len(result.Languages))
		}
	})

	t.Run("top-level dirs extracted from file paths", func(t *testing.T) {
		st := &mockStore{
			listFilesFn: func(_ context.Context) ([]models.File, error) { return files, nil },
		}
		svc := NewOverview(st, nil)
		result, err := svc.Summary(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Expect ["cmd", "internal"] sorted
		if len(result.TopLevelDirs) != 2 {
			t.Errorf("TopLevelDirs = %v, want [cmd internal]", result.TopLevelDirs)
		}
	})

	t.Run("meta fields populated when provided", func(t *testing.T) {
		meta := &storage.RepoMeta{
			Name:              "myrepo",
			LastIndexedCommit: "abc123",
			InitializedAt:     "2026-01-01T00:00:00Z",
		}
		st := &mockStore{}
		svc := NewOverview(st, meta)
		result, err := svc.Summary(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Repo != "myrepo" {
			t.Errorf("Repo = %q, want %q", result.Repo, "myrepo")
		}
		if result.LastIndexedCommit != "abc123" {
			t.Errorf("LastIndexedCommit = %q, want %q", result.LastIndexedCommit, "abc123")
		}
		if result.IndexAgeSeconds <= 0 {
			t.Errorf("IndexAgeSeconds should be > 0 for a past timestamp")
		}
	})

	t.Run("nil meta handled gracefully", func(t *testing.T) {
		st := &mockStore{}
		svc := NewOverview(st, nil)
		result, err := svc.Summary(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Repo != "" {
			t.Errorf("expected empty repo name, got %q", result.Repo)
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		st := &mockStore{
			countSymbolsFn: func(_ context.Context) (int, error) {
				return 0, errors.New("db failure")
			},
		}
		svc := NewOverview(st, nil)
		_, err := svc.Summary(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
