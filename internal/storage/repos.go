package storage

import (
	"encoding/json"
	"fmt"
	"os"
)

// RepoEntry is a single entry in the global repos.json manifest.
type RepoEntry struct {
	Hash          string `json:"hash"`
	Path          string `json:"path"`
	Name          string `json:"name"`
	InitializedAt string `json:"initialized_at"`
	LastSync      string `json:"last_sync"`
}

// ReposManifest is the top-level structure of repos.json.
type ReposManifest struct {
	Repos []RepoEntry `json:"repos"`
}

// ReadReposManifest reads the global repos.json manifest.
// Returns an empty manifest if the file does not exist.
func ReadReposManifest(cacheRoot string) (*ReposManifest, error) {
	path := ReposManifestPath(cacheRoot)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ReposManifest{}, nil
		}
		return nil, fmt.Errorf("read repos.json: %w", err)
	}

	var manifest ReposManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse repos.json: %w", err)
	}

	return &manifest, nil
}

// WriteReposManifest writes the global repos.json manifest.
func WriteReposManifest(cacheRoot string, manifest *ReposManifest) error {
	path := ReposManifestPath(cacheRoot)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal repos.json: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write repos.json: %w", err)
	}

	return nil
}

// UpsertRepo adds or updates a repository entry in the manifest.
func (m *ReposManifest) UpsertRepo(entry RepoEntry) {
	for i, r := range m.Repos {
		if r.Hash == entry.Hash {
			m.Repos[i] = entry
			return
		}
	}
	m.Repos = append(m.Repos, entry)
}
