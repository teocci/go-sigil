// Package discovery enumerates repository files for Sigil's indexing pipeline.
// It abstracts git-based and filesystem-based enumeration behind a single
// Walker interface consumed by the service layer.
//
// NewWalker returns a GitWalker when git ≥ 2.25 is available and root is a
// git repository; otherwise it returns a FilesystemWalker.
package discovery

import (
	"context"
	"time"
)

// FileEntry is a single file discovered during a walk.
type FileEntry struct {
	// Path is repo-relative (forward slashes, no leading /).
	Path string
	// BlobSHA is the git blob SHA; empty in filesystem mode.
	BlobSHA string
	// Untracked is true for files not tracked by git.
	Untracked bool
	// MTime is the last modification time; zero in git mode.
	MTime time.Time
	// Size is the file size in bytes; zero in git mode.
	Size int64
}

// Walker enumerates files under a repository root.
// Implementations must be safe for concurrent use.
type Walker interface {
	// Walk returns all indexable files under root.
	// Returned paths are relative to root, with forward slashes.
	Walk(ctx context.Context, root string) ([]FileEntry, error)
}

// NewWalker returns the best Walker for root.
// extraIgnore holds additional .glyphignore-style patterns from user config.
// If git ≥ 2.25 is present and root is a git repository, a GitWalker is
// returned; otherwise a FilesystemWalker.
func NewWalker(root string, extraIgnore []string) Walker {
	ver, err := detectGitVersion()
	if err == nil && ver.sufficient() {
		repoRoot, err := gitRepoRoot(root)
		if err == nil {
			return &GitWalker{
				root:        repoRoot,
				ver:         ver,
				extraIgnore: extraIgnore,
			}
		}
	}
	return &FilesystemWalker{
		root:        root,
		extraIgnore: extraIgnore,
	}
}
