// Package config handles application configuration loading.
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultCacheRoot returns the default path for the Sigil cache directory.
// Respects CODE_INDEX_PATH env var if set.
// Falls back to ~/.sigil/ on Unix, %APPDATA%\sigil\ on Windows.
func DefaultCacheRoot() string {
	if v := os.Getenv("CODE_INDEX_PATH"); v != "" {
		return v
	}

	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "sigil")
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".sigil")
	}
	return filepath.Join(home, ".sigil")
}
