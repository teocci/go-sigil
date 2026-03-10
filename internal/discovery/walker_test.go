package discovery

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ---- helpers ----------------------------------------------------------------

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func sortPaths(entries []FileEntry) []string {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	sort.Strings(paths)
	return paths
}

func hasPath(entries []FileEntry, want string) bool {
	for _, e := range entries {
		if e.Path == want {
			return true
		}
	}
	return false
}

// ---- FilesystemWalker -------------------------------------------------------

func TestFilesystemWalker_Basic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "src/util.go", "package src")
	writeFile(t, dir, "README.md", "# readme")

	w := &FilesystemWalker{root: dir}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	paths := sortPaths(entries)
	if len(paths) != 3 {
		t.Errorf("want 3 files, got %d: %v", len(paths), paths)
	}
	if !hasPath(entries, "main.go") {
		t.Error("missing main.go")
	}
	if !hasPath(entries, "src/util.go") {
		t.Error("missing src/util.go")
	}
}

func TestFilesystemWalker_BuiltinExclusions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "node_modules/lib.js", "// lib")
	writeFile(t, dir, "vendor/pkg.go", "package pkg")
	writeFile(t, dir, "dist/bundle.min.js", "// min")
	writeFile(t, dir, "app.min.js", "// min")

	w := &FilesystemWalker{root: dir}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for _, e := range entries {
		if strings.Contains(e.Path, "node_modules") {
			t.Errorf("node_modules should be excluded, got %s", e.Path)
		}
		if strings.Contains(e.Path, "vendor") {
			t.Errorf("vendor/ should be excluded, got %s", e.Path)
		}
		if strings.HasSuffix(e.Path, ".min.js") {
			t.Errorf("*.min.js should be excluded, got %s", e.Path)
		}
	}
	if !hasPath(entries, "main.go") {
		t.Error("main.go should be included")
	}
}

func TestFilesystemWalker_Gitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.log\nbuild/\n")
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "error.log", "logs")
	writeFile(t, dir, "src/debug.log", "logs")
	writeFile(t, dir, "build/output.js", "// out")
	writeFile(t, dir, "src/app.go", "package src")

	w := &FilesystemWalker{root: dir}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Path, ".log") {
			t.Errorf("*.log should be ignored, got %s", e.Path)
		}
		if strings.HasPrefix(e.Path, "build/") {
			t.Errorf("build/ should be ignored, got %s", e.Path)
		}
	}
	if !hasPath(entries, "main.go") {
		t.Error("main.go should be present")
	}
	if !hasPath(entries, "src/app.go") {
		t.Error("src/app.go should be present")
	}
}

func TestFilesystemWalker_NestedGitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.tmp\n")
	writeFile(t, dir, "sub/.gitignore", "secret.txt\n")
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "cache.tmp", "tmp")
	writeFile(t, dir, "sub/code.go", "package sub")
	writeFile(t, dir, "sub/secret.txt", "secret")
	writeFile(t, dir, "sub/data.tmp", "tmp")

	w := &FilesystemWalker{root: dir}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	if hasPath(entries, "cache.tmp") {
		t.Error("cache.tmp should be ignored (root *.tmp)")
	}
	if hasPath(entries, "sub/data.tmp") {
		t.Error("sub/data.tmp should be ignored (root *.tmp)")
	}
	if hasPath(entries, "sub/secret.txt") {
		t.Error("sub/secret.txt should be ignored (sub .gitignore)")
	}
	if !hasPath(entries, "sub/code.go") {
		t.Error("sub/code.go should be present")
	}
}

func TestFilesystemWalker_ExtraIgnore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "config.yaml", "key: val")
	writeFile(t, dir, "secrets.yaml", "pw: x")

	w := &FilesystemWalker{root: dir, extraIgnore: []string{"secrets.yaml"}}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if hasPath(entries, "secrets.yaml") {
		t.Error("secrets.yaml should be extra-ignored")
	}
	if !hasPath(entries, "main.go") {
		t.Error("main.go should be present")
	}
}

func TestFilesystemWalker_FileMeta(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")

	w := &FilesystemWalker{root: dir}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.MTime.IsZero() {
		t.Error("MTime should be set in filesystem mode")
	}
	if e.Size == 0 {
		t.Error("Size should be set in filesystem mode")
	}
	if e.BlobSHA != "" {
		t.Error("BlobSHA should be empty in filesystem mode")
	}
}

// ---- GitWalker --------------------------------------------------------------

// gitAvailable reports whether git is available and sufficient.
func gitAvailable() bool {
	ver, err := detectGitVersion()
	return err == nil && ver.sufficient()
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
}

func gitAdd(t *testing.T, dir string, files ...string) {
	t.Helper()
	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
}

func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func TestGitWalker_Basic(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available or too old")
	}

	dir := t.TempDir()
	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "src/util.go", "package src")
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "init")

	w := &GitWalker{root: dir, ver: mustDetectGitVersion(t)}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	if !hasPath(entries, "main.go") {
		t.Error("main.go should be tracked")
	}
	if !hasPath(entries, "src/util.go") {
		t.Error("src/util.go should be tracked")
	}

	for _, e := range entries {
		if e.BlobSHA == "" {
			t.Errorf("tracked file %s should have BlobSHA", e.Path)
		}
	}
}

func TestGitWalker_ExcludesBuiltins(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available or too old")
	}

	dir := t.TempDir()
	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "app.min.js", "// min")
	// Force-track the min.js by bypassing .gitignore
	cmd := exec.Command("git", "add", "-f", "app.min.js")
	cmd.Dir = dir
	cmd.Run() //nolint: ignore error
	gitAdd(t, dir, "main.go")
	gitCommit(t, dir, "init")

	w := &GitWalker{root: dir, ver: mustDetectGitVersion(t)}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Path, ".min.js") {
			t.Errorf("*.min.js should be excluded by built-in rules, got %s", e.Path)
		}
	}
}

func TestGitWalker_Untracked(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available or too old")
	}

	dir := t.TempDir()
	initGitRepo(t, dir)
	writeFile(t, dir, "main.go", "package main")
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "init")

	// Add an untracked file after commit.
	writeFile(t, dir, "new_feature.go", "package main")

	w := &GitWalker{root: dir, ver: mustDetectGitVersion(t)}
	entries, err := w.Walk(context.Background(), dir)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	var found *FileEntry
	for i := range entries {
		if entries[i].Path == "new_feature.go" {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		t.Fatal("new_feature.go (untracked) should be included")
	}
	if !found.Untracked {
		t.Error("new_feature.go should be marked Untracked=true")
	}
}

func mustDetectGitVersion(t *testing.T) gitVersion {
	t.Helper()
	ver, err := detectGitVersion()
	if err != nil {
		t.Fatalf("detectGitVersion: %v", err)
	}
	return ver
}

// ---- Ignorer ----------------------------------------------------------------

func TestIgnorer_BasicPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		isDir   bool
		want    bool
	}{
		{"glob match", "*.log", "error.log", false, true},
		{"glob miss", "*.log", "main.go", false, false},
		{"glob in subdir", "*.log", "src/error.log", false, true},
		{"dir pattern matches dir", "build/", "build", true, true},
		{"dir pattern skips file", "build/", "build", false, false},
		{"anchored pattern", "/vendor", "vendor", false, true},
		{"anchored no match subdir", "/vendor", "src/vendor", false, false},
		{"negation", "!*.go", "main.go", false, false},
		{"double star", "**/node_modules", "a/b/node_modules", false, true},
		{"double star root", "**/node_modules", "node_modules", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ig := NewIgnorer([]string{tt.pattern})
			got := ig.Ignore(tt.path, tt.isDir)
			if got != tt.want {
				t.Errorf("Ignore(%q, %v) = %v, want %v (pattern: %q)", tt.path, tt.isDir, got, tt.want, tt.pattern)
			}
		})
	}
}

func TestIgnorer_LastRuleWins(t *testing.T) {
	// Negate a pattern with a later rule.
	ig := NewIgnorer([]string{"*.go", "!main.go"})
	if ig.Ignore("main.go", false) {
		t.Error("main.go should be un-ignored by !main.go")
	}
	if !ig.Ignore("other.go", false) {
		t.Error("other.go should still be ignored by *.go")
	}
}

// ---- exclusions -------------------------------------------------------------

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		path  string
		extra []string
		want  bool
	}{
		{"node_modules/lib.js", nil, true},
		{"src/node_modules/x.js", nil, true},
		{"vendor/pkg.go", nil, true},
		{"app.min.js", nil, true},
		{"app.min.css", nil, true},
		{"go.sum", nil, false}, // not excluded
		{"Makefile", nil, false},
		{"secrets.yaml", []string{"secrets.yaml"}, true},
		{"main.go", []string{"secrets.yaml"}, false},
	}

	for _, tt := range tests {
		got := isExcluded(tt.path, tt.extra)
		if got != tt.want {
			t.Errorf("isExcluded(%q, %v) = %v, want %v", tt.path, tt.extra, got, tt.want)
		}
	}
}

// ---- FindPackageRoot --------------------------------------------------------

func TestFindPackageRoot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example\n")
	writeFile(t, dir, "cmd/main.go", "package main")
	writeFile(t, dir, "internal/pkg/util.go", "package pkg")

	tests := []struct {
		file string
		want string
	}{
		{"cmd/main.go", "."},
		{"internal/pkg/util.go", "."},
	}

	for _, tt := range tests {
		got := FindPackageRoot(dir, filepath.Join(dir, filepath.FromSlash(tt.file)))
		if got != tt.want {
			t.Errorf("FindPackageRoot(%q) = %q, want %q", tt.file, got, tt.want)
		}
	}
}

func TestFindPackageRoot_NoManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")

	got := FindPackageRoot(dir, filepath.Join(dir, "main.go"))
	if got != "" {
		t.Errorf("FindPackageRoot with no manifest = %q, want empty", got)
	}
}

func TestFindPackageRoot_Nested(t *testing.T) {
	dir := t.TempDir()
	// Root go.mod
	writeFile(t, dir, "go.mod", "module root\n")
	// Nested package.json
	writeFile(t, dir, "frontend/package.json", `{"name":"fe"}`)
	writeFile(t, dir, "frontend/src/app.ts", "// app")

	got := FindPackageRoot(dir, filepath.Join(dir, "frontend/src/app.ts"))
	if got != "frontend" {
		t.Errorf("FindPackageRoot = %q, want %q", got, "frontend")
	}
}
