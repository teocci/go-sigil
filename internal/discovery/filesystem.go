package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// FilesystemWalker uses filepath.WalkDir with gitignore-style filtering.
// Used as a fallback when git is unavailable or too old.
type FilesystemWalker struct {
	root        string
	extraIgnore []string
}

// Walk returns all indexable files under root, respecting .gitignore and
// .sigilignore files found at each directory level, plus built-in exclusions.
func (w *FilesystemWalker) Walk(ctx context.Context, root string) ([]FileEntry, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("abs root: %w", err)
	}

	// dirIgnorers maps absolute directory path → its compiled Ignorer.
	dirIgnorers := make(map[string]*Ignorer)
	loadDirIgnorers(abs, dirIgnorers)

	var entries []FileEntry

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, ferr error) error {
		if ferr != nil {
			return nil // skip unreadable paths silently
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rel, _ := filepath.Rel(abs, path)
		if rel == "." {
			return nil
		}
		relSlash := filepath.ToSlash(rel)

		if d.IsDir() {
			// Check exclusions before descending.
			if isExcluded(relSlash+"/", w.extraIgnore) {
				return filepath.SkipDir
			}
			if isIgnoredByStack(dirIgnorers, abs, path, true) {
				return filepath.SkipDir
			}
			// Load this directory's ignore files for deeper traversal.
			loadDirIgnorers(path, dirIgnorers)
			return nil
		}

		if isExcluded(relSlash, w.extraIgnore) {
			return nil
		}
		if isIgnoredByStack(dirIgnorers, abs, path, false) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		entries = append(entries, FileEntry{
			Path:  relSlash,
			MTime: info.ModTime(),
			Size:  info.Size(),
		})
		return nil
	})

	return entries, err
}

// loadDirIgnorers reads .gitignore and .sigilignore from dir and stores them.
func loadDirIgnorers(dir string, ignorers map[string]*Ignorer) {
	var pats []string
	for _, name := range []string{".gitignore", ".sigilignore"} {
		p, _ := LoadIgnoreFile(filepath.Join(dir, name))
		pats = append(pats, p...)
	}
	if len(pats) > 0 {
		ignorers[dir] = NewIgnorer(pats)
	}
}

// isIgnoredByStack checks whether filePath is ignored by any .gitignore in
// the ancestor chain from repoRoot to filePath's parent directory.
//
// For each ancestor directory D that has an Ignorer, the path is checked
// relative to D. The last matching rule (across all levels) wins, which
// allows deeper .gitignore files to negate patterns from parent directories.
func isIgnoredByStack(ignorers map[string]*Ignorer, repoRoot, filePath string, isDir bool) bool {
	rel, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return false
	}
	relSlash := filepath.ToSlash(rel)
	parts := strings.Split(relSlash, "/")

	ignored := false

	// Check root-level ignorer with the full relative path.
	if ig, ok := ignorers[repoRoot]; ok {
		result, matched := ig.Check(relSlash, isDir)
		if matched {
			ignored = result
		}
	}

	// Check each intermediate directory with progressively shorter paths.
	cur := repoRoot
	for i := 0; i < len(parts)-1; i++ {
		cur = filepath.Join(cur, parts[i])
		subRel := strings.Join(parts[i+1:], "/")
		if ig, ok := ignorers[cur]; ok {
			result, matched := ig.Check(subRel, isDir)
			if matched {
				ignored = result
			}
		}
	}

	return ignored
}
