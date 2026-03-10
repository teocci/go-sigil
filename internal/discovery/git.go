package discovery

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// gitVersion holds a parsed major.minor git version.
type gitVersion struct {
	major, minor int
}

// sufficient reports whether git is new enough for Sigil (≥ 2.25).
func (v gitVersion) sufficient() bool {
	return v.major > 2 || (v.major == 2 && v.minor >= 25)
}

// supportsFormat reports whether --format is available (git ≥ 2.36).
func (v gitVersion) supportsFormat() bool {
	return v.major > 2 || (v.major == 2 && v.minor >= 36)
}

// detectGitVersion runs "git --version" and parses the result.
func detectGitVersion() (gitVersion, error) {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return gitVersion{}, fmt.Errorf("git not found: %w", err)
	}
	// Output: "git version 2.39.1" (possibly with extra info after the version)
	fields := strings.Fields(string(out))
	if len(fields) < 3 {
		return gitVersion{}, fmt.Errorf("unexpected git version output: %q", out)
	}
	parts := strings.SplitN(fields[2], ".", 3)
	if len(parts) < 2 {
		return gitVersion{}, fmt.Errorf("cannot parse git version %q", fields[2])
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return gitVersion{}, fmt.Errorf("parse git major: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return gitVersion{}, fmt.Errorf("parse git minor: %w", err)
	}
	return gitVersion{major: major, minor: minor}, nil
}

// gitRepoRoot returns the absolute path of the git repository root for dir.
func gitRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repo: %w", err)
	}
	root := strings.TrimSpace(string(out))
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("abs repo root: %w", err)
	}
	return abs, nil
}

// GitWalker enumerates files via git ls-files.
type GitWalker struct {
	root        string
	ver         gitVersion
	extraIgnore []string
}

// Walk returns all indexable files under root using git ls-files.
// It combines tracked files (with blob SHAs) and untracked files.
func (w *GitWalker) Walk(ctx context.Context, root string) ([]FileEntry, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("abs root: %w", err)
	}

	// .glyphignore for user-defined extra exclusions
	glyphPats := loadGlyphIgnore(abs)
	ig := NewIgnorer(append(glyphPats, w.extraIgnore...))

	tracked, err := w.lsFiles(ctx, abs)
	if err != nil {
		return nil, err
	}

	// Untracked enumeration is best-effort; failure is non-fatal.
	untracked, _ := w.lsUntracked(ctx, abs)

	all := make([]FileEntry, 0, len(tracked)+len(untracked))
	for _, e := range tracked {
		if isExcluded(e.Path, w.extraIgnore) || ig.Ignore(e.Path, false) {
			continue
		}
		all = append(all, e)
	}
	for _, e := range untracked {
		if isExcluded(e.Path, w.extraIgnore) || ig.Ignore(e.Path, false) {
			continue
		}
		all = append(all, e)
	}
	return all, nil
}

// lsFiles runs git ls-files and returns tracked entries with blob SHAs.
func (w *GitWalker) lsFiles(ctx context.Context, root string) ([]FileEntry, error) {
	var args []string
	if w.ver.supportsFormat() {
		args = []string{"ls-files", "--format=%(objectname) %(path)"}
	} else {
		args = []string{"ls-files", "-s"}
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	if w.ver.supportsFormat() {
		return parseLsFilesFormat(out), nil
	}
	return parseLsFilesStage(out), nil
}

// parseLsFilesFormat parses "%(objectname) %(path)" output (git ≥ 2.36).
func parseLsFilesFormat(out []byte) []FileEntry {
	var entries []FileEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		sp := strings.IndexByte(line, ' ')
		if sp < 0 || sp == len(line)-1 {
			continue
		}
		sha := line[:sp]
		path := filepath.ToSlash(line[sp+1:])
		entries = append(entries, FileEntry{Path: path, BlobSHA: sha})
	}
	return entries
}

// parseLsFilesStage parses "git ls-files -s" output (git 2.25–2.35).
// Format: "100644 <sha> 0\t<path>"
func parseLsFilesStage(out []byte) []FileEntry {
	var entries []FileEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		path := filepath.ToSlash(line[tab+1:])
		fields := strings.Fields(line[:tab])
		if len(fields) < 2 {
			continue
		}
		sha := fields[1]
		entries = append(entries, FileEntry{Path: path, BlobSHA: sha})
	}
	return entries
}

// lsUntracked runs "git ls-files --others --exclude-standard".
func (w *GitWalker) lsUntracked(ctx context.Context, root string) ([]FileEntry, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files --others: %w", err)
	}
	var entries []FileEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		path := filepath.ToSlash(strings.TrimSpace(sc.Text()))
		if path == "" {
			continue
		}
		entries = append(entries, FileEntry{Path: path, Untracked: true})
	}
	return entries, nil
}

// loadGlyphIgnore reads .glyphignore from dir. Returns nil if absent.
func loadGlyphIgnore(dir string) []string {
	pats, _ := LoadIgnoreFile(filepath.Join(dir, ".glyphignore"))
	return pats
}
