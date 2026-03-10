package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-sigil/internal/constants"
	"go-sigil/internal/discovery"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
	"go-sigil/internal/security"
	"go-sigil/internal/store"
)

// IndexOptions controls how an index run behaves.
type IndexOptions struct {
	// Force triggers a full rebuild, ignoring all previously indexed state.
	Force bool
	// MaxFiles overrides the configured limit (0 = use Indexer default).
	MaxFiles int
}

// IndexResult summarises a completed index operation.
type IndexResult struct {
	SessionID    string
	FilesIndexed int
	FilesSkipped int
	FilesDeleted int
	SymbolsTotal int
	Duration     time.Duration
	Errors       []string
}

// Indexer orchestrates the full indexing pipeline:
//
//	Walker → SecurityFilter → Parser → Store
//
// It is safe for use from a single goroutine.
type Indexer struct {
	walker   discovery.Walker
	filter   security.Filter
	registry *parser.Registry
	st       store.SymbolStore
	filesDir string // ~/.sigil/{hash}/files/ — raw file mirror
	repoRoot string // absolute repo root
	maxFiles int
}

// NewIndexer creates an Indexer with all required dependencies.
// maxFiles is the per-run file cap (use constants.DefaultMaxIndexFiles if ≤ 0).
func NewIndexer(
	walker discovery.Walker,
	filter security.Filter,
	registry *parser.Registry,
	st store.SymbolStore,
	filesDir string,
	repoRoot string,
	maxFiles int,
) *Indexer {
	if maxFiles <= 0 {
		maxFiles = constants.DefaultMaxIndexFiles
	}
	return &Indexer{
		walker:   walker,
		filter:   filter,
		registry: registry,
		st:       st,
		filesDir: filesDir,
		repoRoot: repoRoot,
		maxFiles: maxFiles,
	}
}

// Index runs a full or incremental index depending on opts.Force and prior state.
// Smart mode: compares stored blob SHA / mtime+size against the current walk
// and only re-parses changed or new files.
func (idx *Indexer) Index(ctx context.Context, opts IndexOptions) (*IndexResult, error) {
	start := time.Now()
	result := &IndexResult{SessionID: NewSessionID()}

	maxFiles := opts.MaxFiles
	if maxFiles <= 0 {
		maxFiles = idx.maxFiles
	}

	// Walk the repo
	entries, err := idx.walker.Walk(ctx, idx.repoRoot)
	if err != nil {
		return nil, fmt.Errorf("walk repo: %w", err)
	}

	// Load existing file records for change detection (skip in force mode)
	existingFiles := make(map[string]*models.File)
	if !opts.Force {
		tracked, err := idx.st.ListFiles(ctx)
		if err != nil {
			return nil, fmt.Errorf("list tracked files: %w", err)
		}
		for i := range tracked {
			existingFiles[tracked[i].Path] = &tracked[i]
		}
	}

	// Index files
	walkedPaths := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if result.FilesIndexed >= maxFiles {
			slog.Info("max files limit reached", "limit", maxFiles)
			break
		}

		walkedPaths[entry.Path] = true

		// Security classification — skip fully ignored files
		tier := idx.filter.ClassifyFile(entry.Path)
		if tier == security.TierIgnored {
			result.FilesSkipped++
			continue
		}

		// Smart skip: file unchanged since last index
		if !opts.Force {
			if existing, ok := existingFiles[entry.Path]; ok {
				if !fileChanged(entry, existing) {
					result.FilesSkipped++
					continue
				}
			}
		}

		n, err := idx.indexFile(ctx, entry, tier)
		if err != nil {
			slog.Warn("index file failed", "path", entry.Path, "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", entry.Path, err))
			continue
		}
		result.FilesIndexed++
		result.SymbolsTotal += n
	}

	// Detect and remove files no longer present in the walk
	for path := range existingFiles {
		if !walkedPaths[path] {
			if err := idx.st.MarkPossibleUnresolved(ctx, path); err != nil {
				slog.Warn("mark possible unresolved", "path", path, "error", err)
			}
			if err := idx.st.DeleteFile(ctx, path); err != nil {
				slog.Warn("delete stale file record", "path", path, "error", err)
			}
			result.FilesDeleted++
		}
	}

	// Resolve callee IDs across all indexed symbols
	if err := idx.resolveCallees(ctx); err != nil {
		slog.Warn("callee resolution incomplete", "error", err)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// indexFile reads, classifies, parses, and stores a single file.
// Returns the number of symbols extracted.
func (idx *Indexer) indexFile(ctx context.Context, entry discovery.FileEntry, tier security.Tier) (int, error) {
	absPath := filepath.Join(idx.repoRoot, filepath.FromSlash(entry.Path))

	src, err := os.ReadFile(absPath)
	if err != nil {
		return 0, fmt.Errorf("read file: %w", err)
	}

	// Compute blob SHA: use git blob SHA in git mode; SHA256(content) otherwise
	blobSHA := entry.BlobSHA
	if blobSHA == "" {
		h := sha256.Sum256(src)
		blobSHA = hex.EncodeToString(h[:])
	}

	// Mirror raw file content (content-addressed, deduplication via fixed path)
	if err := idx.mirrorFile(blobSHA, src); err != nil {
		slog.Warn("mirror file failed", "path", entry.Path, "error", err)
		// Non-fatal: retrieval falls back to reading the original path
	}

	// Determine language from extension
	ext := strings.ToLower(filepath.Ext(entry.Path))
	language := constants.LanguageExtensions[ext]

	// pkgPath for qualified-name building = repo-relative directory of the file.
	// e.g. "internal/parser/golang" for internal/parser/golang/golang.go.
	pkgPath := filepath.ToSlash(filepath.Dir(entry.Path))
	if pkgPath == "." {
		pkgPath = ""
	}

	// packageRoot = nearest ancestor with a package manifest (go.mod, package.json…).
	packageRoot := discovery.FindPackageRoot(idx.repoRoot, absPath)

	now := time.Now().UTC().Format(time.RFC3339)

	fileRecord := models.File{
		Path:        entry.Path,
		BlobSHA:     blobSHA,
		LastIndexed: now,
	}
	if !entry.MTime.IsZero() {
		fileRecord.Mtime = entry.MTime.UTC().Format(time.RFC3339)
		fileRecord.Size = entry.Size
	}

	var symbols []models.Symbol
	var edges []models.CallEdge

	if language != "" {
		if p, ok := idx.registry.Get(language); ok {
			res, err := p.Parse(entry.Path, pkgPath, src)
			if err != nil {
				slog.Warn("parse error", "path", entry.Path, "language", language, "error", err)
				// Continue — still record the file so incremental sync skips it
			} else {
				for _, sym := range res.Symbols {
					sym.Language = language
					sym.IndexedAt = now
					sym.PackageRoot = packageRoot

					// Redact byte offsets for sensitive files
					if tier == security.TierRedacted {
						sym.ByteStart = nil
						sym.ByteEnd = nil
					} else if sym.ByteStart != nil && sym.ByteEnd != nil {
						// Compute content hash from source slice
						s, e := *sym.ByteStart, *sym.ByteEnd
						if s >= 0 && e <= len(src) && s <= e {
							h := sha256.Sum256(src[s:e])
							sym.ContentHash = hex.EncodeToString(h[:8]) // 16 hex chars
						}
					}
					symbols = append(symbols, *sym)
				}
				for _, edge := range res.Edges {
					edges = append(edges, *edge)
				}
			}
		}
	}

	if err := idx.st.UpsertFile(ctx, fileRecord); err != nil {
		return 0, fmt.Errorf("upsert file: %w", err)
	}
	if err := idx.st.ReplaceFileSymbols(ctx, entry.Path, symbols); err != nil {
		return 0, fmt.Errorf("replace symbols: %w", err)
	}
	if err := idx.st.ReplaceFileEdges(ctx, entry.Path, edges); err != nil {
		return 0, fmt.Errorf("replace edges: %w", err)
	}

	return len(symbols), nil
}

// mirrorFile writes src to filesDir/{blobSHA} if it doesn't already exist.
// Content-addressed deduplication: same SHA means same bytes.
func (idx *Indexer) mirrorFile(blobSHA string, src []byte) error {
	if idx.filesDir == "" || blobSHA == "" {
		return nil
	}
	dest := filepath.Join(idx.filesDir, blobSHA)
	if _, err := os.Stat(dest); err == nil {
		return nil // already mirrored
	}
	return os.WriteFile(dest, src, 0600)
}

// resolveCallees performs a best-effort resolution of call edges whose callee_id
// is empty. Looks up callee names in the current symbol table by name.
// Multiple matches (overloaded names) are left unresolved.
func (idx *Indexer) resolveCallees(ctx context.Context) error {
	files, err := idx.st.ListFiles(ctx)
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	// Build name → []id index from all symbols
	nameIndex := make(map[string][]string)
	for _, f := range files {
		syms, err := idx.st.GetSymbolsByFile(ctx, f.Path)
		if err != nil {
			continue
		}
		for _, sym := range syms {
			nameIndex[sym.Name] = append(nameIndex[sym.Name], sym.ID)
		}
	}

	// For each file, resolve unresolved edges and re-insert
	for _, f := range files {
		syms, err := idx.st.GetSymbolsByFile(ctx, f.Path)
		if err != nil {
			continue
		}
		for _, sym := range syms {
			calls, err := idx.st.GetCalls(ctx, sym.ID, 1)
			if err != nil {
				continue
			}

			var resolved []models.CallEdge
			changed := false
			for _, edge := range calls {
				if edge.CalleeID != "" {
					resolved = append(resolved, edge)
					continue
				}
				// Try to resolve by raw expression (function name)
				name := edge.RawExpression
				// Strip package qualifier: "pkg.Func" → "Func"
				if dot := strings.LastIndex(name, "."); dot >= 0 {
					name = name[dot+1:]
				}
				ids := nameIndex[name]
				if len(ids) == 1 {
					edge.CalleeID = ids[0]
					changed = true
				}
				resolved = append(resolved, edge)
			}

			if !changed {
				continue
			}

			// Re-insert resolved edges for this file
			// (ReplaceFileEdges deletes by caller symbol → file join, so pass file path)
			if err := idx.st.ReplaceFileEdges(ctx, f.Path, resolved); err != nil {
				slog.Warn("replace edges after resolution", "file", f.Path, "error", err)
			}
		}
	}
	return nil
}

// fileChanged reports whether entry differs from the stored file record.
// Git mode: compare blob SHA. Filesystem mode: compare mtime + size.
func fileChanged(entry discovery.FileEntry, stored *models.File) bool {
	if entry.BlobSHA != "" {
		return entry.BlobSHA != stored.BlobSHA
	}
	// Filesystem mode
	if entry.Size != stored.Size {
		return true
	}
	if !entry.MTime.IsZero() {
		storedMtime := stored.Mtime
		entryMtime := entry.MTime.UTC().Format(time.RFC3339)
		return entryMtime != storedMtime
	}
	return false
}
