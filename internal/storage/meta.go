package storage

import (
	"encoding/json"
	"fmt"
	"os"
)

// RepoMeta holds repository-level metadata persisted in meta.json.
type RepoMeta struct {
	Hash               string `json:"hash"`
	Path               string `json:"path"`
	Name               string `json:"name"`
	Mode               string `json:"mode"`                // "git" or "filesystem"
	InitializedAt      string `json:"initialized_at"`
	LastIndexedCommit  string `json:"last_indexed_commit,omitempty"`
}

// ReadMeta reads the meta.json file for a repository.
// Returns nil with no error if the file does not exist.
func ReadMeta(cacheRoot string, repoHash string) (*RepoMeta, error) {
	path := MetaPath(cacheRoot, repoHash)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read meta.json: %w", err)
	}

	var meta RepoMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse meta.json: %w", err)
	}

	return &meta, nil
}

// WriteMeta writes the meta.json file for a repository.
func WriteMeta(cacheRoot string, repoHash string, meta *RepoMeta) error {
	path := MetaPath(cacheRoot, repoHash)

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta.json: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write meta.json: %w", err)
	}

	return nil
}
