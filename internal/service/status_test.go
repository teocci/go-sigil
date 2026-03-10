package service

import (
	"context"
	"testing"

	"go-sigil/internal/storage"
)

func TestStatus_Check(t *testing.T) {
	ctx := context.Background()

	t.Run("returns counts from store", func(t *testing.T) {
		st := &mockStore{
			countFilesFn:              func(_ context.Context) (int, error) { return 20, nil },
			countSymbolsFn:            func(_ context.Context) (int, error) { return 200, nil },
			countPossibleUnresolvedFn: func(_ context.Context) (int, error) { return 5, nil },
		}
		svc := NewStatus(st, nil, "/repo")
		result, err := svc.Check(ctx, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.TotalFiles != 20 {
			t.Errorf("TotalFiles = %d, want 20", result.TotalFiles)
		}
		if result.TotalSymbols != 200 {
			t.Errorf("TotalSymbols = %d, want 200", result.TotalSymbols)
		}
		if result.PossibleUnresolvedCount != 5 {
			t.Errorf("PossibleUnresolvedCount = %d, want 5", result.PossibleUnresolvedCount)
		}
	})

	t.Run("meta fields populated", func(t *testing.T) {
		meta := &storage.RepoMeta{
			Name:              "myrepo",
			LastIndexedCommit: "deadbeef",
			Mode:              "git",
			InitializedAt:     "2026-01-01T00:00:00Z",
		}
		st := &mockStore{}
		svc := NewStatus(st, meta, "/repo")
		result, err := svc.Check(ctx, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Repo != "myrepo" {
			t.Errorf("Repo = %q, want %q", result.Repo, "myrepo")
		}
		if result.Mode != "git" {
			t.Errorf("Mode = %q, want %q", result.Mode, "git")
		}
		if result.IndexAgeSeconds <= 0 {
			t.Error("IndexAgeSeconds should be positive for a past timestamp")
		}
	})

	t.Run("nil meta defaults mode to filesystem", func(t *testing.T) {
		st := &mockStore{}
		svc := NewStatus(st, nil, "/repo")
		result, err := svc.Check(ctx, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Mode != "filesystem" {
			t.Errorf("Mode = %q, want %q", result.Mode, "filesystem")
		}
	})

	t.Run("no verification result without verify flag", func(t *testing.T) {
		st := &mockStore{}
		svc := NewStatus(st, nil, "/repo")
		result, err := svc.Check(ctx, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Verification != nil {
			t.Error("expected nil Verification without --verify flag")
		}
	})

	t.Run("path set to repoRoot", func(t *testing.T) {
		st := &mockStore{}
		svc := NewStatus(st, nil, "/my/repo/root")
		result, err := svc.Check(ctx, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Path != "/my/repo/root" {
			t.Errorf("Path = %q, want %q", result.Path, "/my/repo/root")
		}
	})
}
