// Package storage manages the ~/.sigil/ filesystem layout.
package storage

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// RepoHash computes the 12-character hex hash used as the directory name
// for a repository's index data: SHA256(canonical_path)[:12].
func RepoHash(repoPath string) (string, error) {
	canonical, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	canonical = filepath.Clean(canonical)

	h := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", h[:6]), nil // 6 bytes = 12 hex chars
}

// EnsureLayout creates the directory structure for a repository index:
//
//	{cacheRoot}/{repoHash}/
//	{cacheRoot}/{repoHash}/files/
//
// Directories are created with mode 0700.
func EnsureLayout(cacheRoot string, repoHash string) error {
	dirs := []string{
		cacheRoot,
		filepath.Join(cacheRoot, repoHash),
		filepath.Join(cacheRoot, repoHash, "files"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}

// IndexDBPath returns the path to the SQLite index database.
func IndexDBPath(cacheRoot string, repoHash string) string {
	return filepath.Join(cacheRoot, repoHash, "index.db")
}

// MetaPath returns the path to the repo metadata file.
func MetaPath(cacheRoot string, repoHash string) string {
	return filepath.Join(cacheRoot, repoHash, "meta.json")
}

// FilesDir returns the path to the raw source file mirror directory.
func FilesDir(cacheRoot string, repoHash string) string {
	return filepath.Join(cacheRoot, repoHash, "files")
}

// ReposManifestPath returns the path to the global repos.json manifest.
func ReposManifestPath(cacheRoot string) string {
	return filepath.Join(cacheRoot, "repos.json")
}

// TokensSavedPath returns the path to the global savings rollup file.
func TokensSavedPath(cacheRoot string) string {
	return filepath.Join(cacheRoot, "tokens_saved.json")
}
