package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-sigil/internal/db"
	"go-sigil/internal/discovery"
	"go-sigil/internal/parser"
	"go-sigil/internal/parser/golang"
	"go-sigil/internal/parser/javascript"
	"go-sigil/internal/parser/python"
	"go-sigil/internal/parser/typescript"
	"go-sigil/internal/security"
	"go-sigil/internal/service"
	"go-sigil/internal/storage"
	"go-sigil/internal/store"

	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "index [path]",
		Short: "Index a repository",
		Long: `Index a repository's source code into the Sigil symbol store.

On first run, performs a full index. Subsequent runs are incremental —
only changed files are re-parsed. Use --force to trigger a full rebuild.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}

			return runIndex(cmd, root, service.IndexOptions{Force: force})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "force full rebuild, ignoring prior state")
	return cmd
}

// runIndex sets up all dependencies and runs the indexer.
func runIndex(cmd *cobra.Command, repoRoot string, opts service.IndexOptions) error {
	repoHash, err := storage.RepoHash(repoRoot)
	if err != nil {
		return fmt.Errorf("compute repo hash: %w", err)
	}

	if err := storage.EnsureLayout(cfg.CacheRoot, repoHash); err != nil {
		return fmt.Errorf("ensure storage layout: %w", err)
	}

	// Open / migrate database
	dbPath := storage.IndexDBPath(cfg.CacheRoot, repoHash)
	rawDB, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	if err := db.Run(rawDB); err != nil {
		rawDB.Close()
		return fmt.Errorf("migrate database: %w", err)
	}
	st := store.New(rawDB)
	defer st.Close()

	// Build dependencies
	walker := discovery.NewWalker(repoRoot, cfg.Security.ExtraIgnoreFilenames)

	filter, err := security.NewFilter(
		cfg.Security.ExtraIgnoreFilenames,
		cfg.Security.ExtraSecretFilenames,
		cfg.Security.ExtraSecretPatterns,
	)
	if err != nil {
		return fmt.Errorf("build security filter: %w", err)
	}

	registry := parser.NewRegistry()
	registry.Register(golang.New())
	registry.Register(typescript.New())
	registry.Register(javascript.New())
	registry.Register(python.New())

	filesDir := storage.FilesDir(cfg.CacheRoot, repoHash)
	indexer := service.NewIndexer(
		walker, filter, registry, st,
		filesDir, repoRoot,
		cfg.Indexing.MaxFiles,
	)

	// Register repo in global manifest
	repoName := filepath.Base(repoRoot)
	manifest, err := storage.ReadReposManifest(cfg.CacheRoot)
	if err != nil {
		return fmt.Errorf("read repos manifest: %w", err)
	}
	manifest.UpsertRepo(storage.RepoEntry{
		Hash:          repoHash,
		Path:          repoRoot,
		Name:          repoName,
		InitializedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err := storage.WriteReposManifest(cfg.CacheRoot, manifest); err != nil {
		return fmt.Errorf("write repos manifest: %w", err)
	}

	// Run index
	fmt.Fprintf(cmd.OutOrStdout(), "Indexing %s ...\n", repoRoot)
	result, err := indexer.Index(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("index: %w", err)
	}

	// Update meta.json
	mode := "filesystem"
	if w, ok := walker.(interface{ Mode() string }); ok {
		mode = w.Mode()
	}
	meta := &storage.RepoMeta{
		Hash:          repoHash,
		Path:          repoRoot,
		Name:          repoName,
		Mode:          mode,
		InitializedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := storage.WriteMeta(cfg.CacheRoot, repoHash, meta); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	// Report
	printIndexResult(cmd, result)
	return nil
}

func printIndexResult(cmd *cobra.Command, r *service.IndexResult) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "\nDone in %s\n", r.Duration.Round(time.Millisecond))
	fmt.Fprintf(w, "  Files indexed : %d\n", r.FilesIndexed)
	fmt.Fprintf(w, "  Files skipped : %d\n", r.FilesSkipped)
	if r.FilesDeleted > 0 {
		fmt.Fprintf(w, "  Files removed : %d\n", r.FilesDeleted)
	}
	fmt.Fprintf(w, "  Symbols total : %d\n", r.SymbolsTotal)
	fmt.Fprintf(w, "  Session ID    : %s\n", r.SessionID)

	if len(r.Errors) > 0 {
		fmt.Fprintf(w, "\n  Warnings (%d):\n", len(r.Errors))
		for _, e := range r.Errors {
			fmt.Fprintf(w, "    - %s\n", e)
		}
	}
}

// resolveRepoRoot returns the absolute path to the repo root.
// Uses the first argument if provided, otherwise falls back to CWD.
func resolveRepoRoot(args []string) (string, error) {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", target, err)
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("path %q: %w", abs, err)
	}
	return abs, nil
}
