package cli

import (
	"fmt"

	"go-sigil/internal/db"
	"go-sigil/internal/storage"
	"go-sigil/internal/store"
)

// openStoreForRepo opens the SQLite store for a repository root.
// Returns the store and the repo hash. Caller must call st.Close().
func openStoreForRepo(repoRoot string) (store.SymbolStore, string, error) {
	repoHash, err := storage.RepoHash(repoRoot)
	if err != nil {
		return nil, "", fmt.Errorf("compute repo hash: %w", err)
	}

	dbPath := storage.IndexDBPath(cfg.CacheRoot, repoHash)
	rawDB, err := db.Open(dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("open database: %w", err)
	}
	if err := db.Run(rawDB); err != nil {
		rawDB.Close()
		return nil, "", fmt.Errorf("migrate database: %w", err)
	}
	return store.New(rawDB), repoHash, nil
}
