package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-sigil/internal/storage"
)

// RepoCacheInfo describes a repo's cache entry.
type RepoCacheInfo struct {
	Hash      string `json:"hash"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
}

// CacheStatus describes the current cache state.
type CacheStatus struct {
	CacheRoot string          `json:"cache_root"`
	Repos     []RepoCacheInfo `json:"repos"`
	TotalSize int64           `json:"total_size"`
}

// CacheManager manages the ~/.sigil/ cache.
type CacheManager struct {
	cacheRoot string
}

// NewCacheManager creates a CacheManager.
func NewCacheManager(cacheRoot string) *CacheManager {
	return &CacheManager{cacheRoot: cacheRoot}
}

// Status returns the current cache status.
func (c *CacheManager) Status(_ context.Context) (*CacheStatus, error) {
	manifest, err := storage.ReadReposManifest(c.cacheRoot)
	if err != nil {
		return nil, fmt.Errorf("read repos manifest: %w", err)
	}

	status := &CacheStatus{CacheRoot: c.cacheRoot}
	for _, repo := range manifest.Repos {
		repoDir := filepath.Join(c.cacheRoot, repo.Hash)
		size := dirSize(repoDir)
		status.Repos = append(status.Repos, RepoCacheInfo{
			Hash:      repo.Hash,
			Path:      repo.Path,
			Name:      repo.Name,
			SizeBytes: size,
		})
		status.TotalSize += size
	}
	return status, nil
}

// Invalidate removes the index DB for a repo, forcing a full rebuild.
func (c *CacheManager) Invalidate(_ context.Context, repoRoot string) error {
	hash, err := storage.RepoHash(repoRoot)
	if err != nil {
		return err
	}
	dbPath := storage.IndexDBPath(c.cacheRoot, hash)
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove index db: %w", err)
	}
	return nil
}

// Remove completely removes a repo's cache directory.
func (c *CacheManager) Remove(_ context.Context, repoRoot string) error {
	hash, err := storage.RepoHash(repoRoot)
	if err != nil {
		return err
	}
	repoDir := filepath.Join(c.cacheRoot, hash)
	return os.RemoveAll(repoDir)
}

// ClearAll removes all cached repo data and resets the manifest.
func (c *CacheManager) ClearAll(_ context.Context) error {
	manifest, err := storage.ReadReposManifest(c.cacheRoot)
	if err != nil {
		return err
	}
	for _, repo := range manifest.Repos {
		repoDir := filepath.Join(c.cacheRoot, repo.Hash)
		_ = os.RemoveAll(repoDir)
	}
	return storage.WriteReposManifest(c.cacheRoot, &storage.ReposManifest{})
}

// Prune removes cache entries whose index DB is older than olderThan.
// Returns the number of repos pruned.
func (c *CacheManager) Prune(_ context.Context, olderThan time.Duration) (int, error) {
	manifest, err := storage.ReadReposManifest(c.cacheRoot)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-olderThan)
	var kept []storage.RepoEntry
	removed := 0
	for _, repo := range manifest.Repos {
		dbPath := storage.IndexDBPath(c.cacheRoot, repo.Hash)
		info, err := os.Stat(dbPath)
		if err != nil || info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(c.cacheRoot, repo.Hash))
			removed++
			continue
		}
		kept = append(kept, repo)
	}
	manifest.Repos = kept
	return removed, storage.WriteReposManifest(c.cacheRoot, manifest)
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
