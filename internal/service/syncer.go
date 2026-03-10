package service

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// Syncer performs incremental updates by running Index with Force=false.
// Intended for use by the git pre-commit hook (sigil sync).
type Syncer struct {
	idx *Indexer
}

// NewSyncer creates a Syncer backed by the given Indexer.
func NewSyncer(idx *Indexer) *Syncer {
	return &Syncer{idx: idx}
}

// Sync performs an incremental update: only changed or new files are re-parsed.
// If the repository is in git mode, staged files are detected and prioritised.
// Returns the index result from the incremental pass.
func (s *Syncer) Sync(ctx context.Context) (*IndexResult, error) {
	result, err := s.idx.Index(ctx, IndexOptions{Force: false})
	if err != nil {
		return nil, fmt.Errorf("sync: %w", err)
	}

	slog.Info("sync complete",
		"indexed", result.FilesIndexed,
		"skipped", result.FilesSkipped,
		"deleted", result.FilesDeleted,
		"symbols", result.SymbolsTotal,
		"duration", result.Duration,
	)
	return result, nil
}

// StagedFiles returns the list of staged file paths in the repository at root.
// Returns an empty slice if git is not available or there are no staged files.
func StagedFiles(root string) []string {
	out, err := exec.Command("git", "-C", root, "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}
