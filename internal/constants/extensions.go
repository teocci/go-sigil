// Package constants — per-language source file extension allow-lists.
package constants

// LanguageExtensions maps file extensions to their language name.
// Only files with these extensions are parsed for symbols.
var LanguageExtensions = map[string]string{
	// Phase 1
	".go":  "go",
	".ts":  "typescript",
	".tsx": "typescript",
	".js":  "javascript",
	".jsx": "javascript",
	".mjs": "javascript",
	".cjs": "javascript",
	".py":  "python",
	".pyi": "python",
}

// SupportedLanguages returns the set of languages supported in Phase 1.
func SupportedLanguages() []string {
	return []string{"go", "typescript", "javascript", "python"}
}

// PackageManifests are filenames that define package boundaries.
// When found in a directory, all symbols in that directory tree
// have their package_root set to the directory path.
var PackageManifests = []string{
	"go.mod",
	"package.json",
	"pyproject.toml",
	"Cargo.toml",
	"pom.xml",
	"build.gradle",
	"setup.py",
	"setup.cfg",
}
