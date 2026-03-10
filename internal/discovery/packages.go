package discovery

import (
	"os"
	"path/filepath"

	"go-sigil/internal/constants"
)

// FindPackageRoot returns the nearest ancestor directory (from filePath up to
// repoRoot) that contains a package manifest file (go.mod, package.json, etc.).
// Returns "" if no manifest is found.
func FindPackageRoot(repoRoot, filePath string) string {
	dir := filepath.Dir(filePath)
	for {
		if containsManifest(dir) {
			rel, err := filepath.Rel(repoRoot, dir)
			if err != nil {
				return ""
			}
			return filepath.ToSlash(rel)
		}
		if dir == repoRoot {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// containsManifest reports whether dir contains any package manifest file.
func containsManifest(dir string) bool {
	for _, name := range constants.PackageManifests {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
