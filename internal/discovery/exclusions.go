package discovery

import (
	"path/filepath"
	"strings"

	"go-sigil/internal/constants"
)

// isExcluded reports whether relPath (forward slashes) matches any built-in
// exclusion or user-supplied extra pattern.
func isExcluded(relPath string, extra []string) bool {
	name := filepath.Base(relPath)
	for _, pat := range constants.BuiltinExclusions {
		if exclusionMatches(pat, relPath, name) {
			return true
		}
	}
	for _, pat := range extra {
		if exclusionMatches(pat, relPath, name) {
			return true
		}
	}
	return false
}

// exclusionMatches checks a single exclusion pattern against relPath/name.
//
// Pattern types:
//   - "node_modules/"  → directory component match
//   - "*.min.js"       → glob on basename
//   - "Makefile"       → exact basename match
func exclusionMatches(pat, relSlash, name string) bool {
	if strings.HasSuffix(pat, "/") {
		dir := pat[:len(pat)-1]
		for _, seg := range strings.Split(relSlash, "/") {
			if seg == dir {
				return true
			}
		}
		return false
	}
	if strings.ContainsAny(pat, "*?[") {
		ok, _ := filepath.Match(pat, name)
		return ok
	}
	return name == pat
}
