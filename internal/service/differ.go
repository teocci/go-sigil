package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"go-sigil/internal/store"
)

// DiffSymbol describes a symbol change.
type DiffSymbol struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name"`
	File          string `json:"file"`
	ChangeType    string `json:"change_type,omitempty"` // modified|added|deleted
	Summary       string `json:"summary,omitempty"`
}

// DiffResult holds the symbol-level diff.
type DiffResult struct {
	Since    string       `json:"since"`
	Added    []DiffSymbol `json:"added"`
	Modified []DiffSymbol `json:"modified"`
	Deleted  []DiffSymbol `json:"deleted"`
	Errors   []string     `json:"errors,omitempty"`
}

// Differ produces symbol-level diffs using git.
type Differ struct {
	st       store.SymbolStore
	repoRoot string
}

// NewDiffer creates a Differ.
func NewDiffer(st store.SymbolStore, repoRoot string) *Differ {
	return &Differ{st: st, repoRoot: repoRoot}
}

// Diff returns symbol-level changes since the given git ref.
func (d *Differ) Diff(ctx context.Context, since string) (*DiffResult, error) {
	result := &DiffResult{Since: since}

	cmd := exec.CommandContext(ctx, "git", "-C", d.repoRoot, "diff", "--name-status", since)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff %q: %w", since, err)
	}

	changedFiles := parseGitNameStatus(string(out))

	for path, status := range changedFiles {
		syms, err := d.st.GetSymbolsByFile(ctx, path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}

		for _, sym := range syms {
			ds := DiffSymbol{
				ID:            sym.ID,
				Kind:          sym.Kind,
				Name:          sym.Name,
				QualifiedName: sym.QualifiedName,
				File:          sym.File,
				Summary:       sym.Summary,
			}
			switch status {
			case "A":
				result.Added = append(result.Added, ds)
			case "D":
				result.Deleted = append(result.Deleted, ds)
			default:
				ds.ChangeType = "modified"
				result.Modified = append(result.Modified, ds)
			}
		}
	}

	return result, nil
}

func parseGitNameStatus(output string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := string(parts[0][0])
		path := parts[len(parts)-1]
		result[path] = status
	}
	return result
}
