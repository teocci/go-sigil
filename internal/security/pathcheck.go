package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsPathSafe reports whether path is safely contained within root.
// It resolves symlinks on both sides and checks that the real path of
// the file does not escape the real path of root.
// Returns false (not an error) when the file does not exist.
func IsPathSafe(root, path string) (bool, error) {
	realRoot, err := evalOrAbs(root)
	if err != nil {
		return false, fmt.Errorf("resolve root: %w", err)
	}

	realPath, err := evalOrAbs(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("resolve path: %w", err)
	}

	rel, err := filepath.Rel(realRoot, realPath)
	if err != nil {
		return false, fmt.Errorf("rel path: %w", err)
	}

	// filepath.Rel returns ".." prefix when realPath is outside realRoot.
	safe := !strings.HasPrefix(rel, "..")
	return safe, nil
}

// evalOrAbs resolves symlinks; falls back to filepath.Abs if the path does
// not exist yet (so the caller can distinguish existence errors from others).
func evalOrAbs(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err // caller handles os.IsNotExist
	}
	return real, nil
}
