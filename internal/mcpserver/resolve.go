// Package mcpserver implements the Sigil MCP stdio server.
package mcpserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-sigil/internal/db"
	"go-sigil/internal/storage"
	"go-sigil/internal/store"
)

// resolveRepoRoot finds the git root from path.
// If path is empty, serverCWD is used as the starting point.
func resolveRepoRoot(path, serverCWD string) (string, error) {
	base := path
	if base == "" {
		base = serverCWD
	}

	info, err := os.Stat(base)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", base, err)
	}
	dir := base
	if !info.IsDir() {
		dir = filepath.Dir(base)
	}

	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%q is not inside a git repository", dir)
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

// openRepo opens the SQLite store and reads metadata for repoRoot.
// Returns a descriptive error if the index database does not exist.
func openRepo(repoRoot, cacheRoot string) (store.SymbolStore, *storage.RepoMeta, error) {
	repoHash, err := storage.RepoHash(repoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("compute repo hash: %w", err)
	}

	dbPath := storage.IndexDBPath(cacheRoot, repoHash)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("repo %q has not been indexed — run: sigil index .", repoRoot)
	}

	rawDB, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Run(rawDB); err != nil {
		rawDB.Close()
		return nil, nil, fmt.Errorf("migrate database: %w", err)
	}

	meta, _ := storage.ReadMeta(cacheRoot, repoHash)
	return store.New(rawDB), meta, nil
}
