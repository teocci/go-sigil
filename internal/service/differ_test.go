package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go-sigil/internal/models"
)

// TestParseGitNameStatus tests the internal git name-status parser.
// The Diff() method itself requires git and is covered by integration tests.
func TestParseGitNameStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantKeys map[string]string
	}{
		{
			name: "modified file",
			input: "M\tinternal/service/searcher.go\n",
			wantLen: 1,
			wantKeys: map[string]string{"internal/service/searcher.go": "M"},
		},
		{
			name: "added file",
			input: "A\tcmd/new.go\n",
			wantLen: 1,
			wantKeys: map[string]string{"cmd/new.go": "A"},
		},
		{
			name: "deleted file",
			input: "D\told/file.go\n",
			wantLen: 1,
			wantKeys: map[string]string{"old/file.go": "D"},
		},
		{
			name: "multiple files",
			input: "M\ta.go\nA\tb.go\nD\tc.go\n",
			wantLen: 3,
			wantKeys: map[string]string{"a.go": "M", "b.go": "A", "c.go": "D"},
		},
		{
			name:    "empty output",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "blank lines ignored",
			input:   "\n\n",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitNameStatus(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d (result: %v)", len(result), tt.wantLen, result)
			}
			for path, wantStatus := range tt.wantKeys {
				got, ok := result[path]
				if !ok {
					t.Errorf("missing path %q in result", path)
					continue
				}
				if got != wantStatus {
					t.Errorf("result[%q] = %q, want %q", path, got, wantStatus)
				}
			}
		})
	}
}

// ---- git helpers for Diff() integration tests --------------------------------

func gitAvailableForDiffer() bool {
	cmd := exec.Command("git", "version")
	return cmd.Run() == nil
}

func initDifferRepo(t *testing.T, dir string) {
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

func writeAndCommit(t *testing.T, dir, name, content, msg string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	add := exec.Command("git", "add", name)
	add.Dir = dir
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	commit := exec.Command("git", "commit", "-m", msg)
	commit.Dir = dir
	if out, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// TestDiffer_Diff exercises the Diff() method against a real temp git repository.
func TestDiffer_Diff(t *testing.T) {
	if !gitAvailableForDiffer() {
		t.Skip("git not available")
	}

	t.Run("modified file appears in result", func(t *testing.T) {
		dir := t.TempDir()
		initDifferRepo(t, dir)

		// Commit 1: add a.go
		writeAndCommit(t, dir, "a.go", "package main\n", "add a.go")
		// Commit 2: modify a.go
		writeAndCommit(t, dir, "a.go", "package main\n\nfunc Foo() {}\n", "modify a.go")

		st := &mockStore{
			getSymbolsByFileFn: func(_ context.Context, file string) ([]models.Symbol, error) {
				if file == "a.go" {
					return []models.Symbol{{ID: "abc1", Kind: "function", Name: "Foo", QualifiedName: "main.Foo", File: "a.go"}}, nil
				}
				return nil, nil
			},
		}

		d := NewDiffer(st, dir)
		result, err := d.Diff(context.Background(), "HEAD~1")
		if err != nil {
			t.Fatalf("Diff: %v", err)
		}

		if result.Since != "HEAD~1" {
			t.Errorf("Since = %q, want %q", result.Since, "HEAD~1")
		}
		if len(result.Modified) != 1 || result.Modified[0].Name != "Foo" {
			t.Errorf("Modified = %v, want 1 symbol Foo", result.Modified)
		}
		if len(result.Added) != 0 || len(result.Deleted) != 0 {
			t.Errorf("unexpected Added=%v Deleted=%v", result.Added, result.Deleted)
		}
	})

	t.Run("added file symbols appear in Added", func(t *testing.T) {
		dir := t.TempDir()
		initDifferRepo(t, dir)

		// Commit 1: add a.go
		writeAndCommit(t, dir, "a.go", "package main\n", "initial")
		// Commit 2: add b.go (new file)
		writeAndCommit(t, dir, "b.go", "package main\nfunc Bar() {}\n", "add b.go")

		st := &mockStore{
			getSymbolsByFileFn: func(_ context.Context, file string) ([]models.Symbol, error) {
				if file == "b.go" {
					return []models.Symbol{{ID: "def2", Kind: "function", Name: "Bar", QualifiedName: "main.Bar", File: "b.go"}}, nil
				}
				return nil, nil
			},
		}

		d := NewDiffer(st, dir)
		result, err := d.Diff(context.Background(), "HEAD~1")
		if err != nil {
			t.Fatalf("Diff: %v", err)
		}

		if len(result.Added) != 1 || result.Added[0].Name != "Bar" {
			t.Errorf("Added = %v, want 1 symbol Bar", result.Added)
		}
		if len(result.Modified) != 0 || len(result.Deleted) != 0 {
			t.Errorf("unexpected Modified=%v Deleted=%v", result.Modified, result.Deleted)
		}
	})

	t.Run("no changes between HEAD and working tree returns empty", func(t *testing.T) {
		dir := t.TempDir()
		initDifferRepo(t, dir)
		writeAndCommit(t, dir, "a.go", "package main\n", "init")

		st := &mockStore{}
		d := NewDiffer(st, dir)
		result, err := d.Diff(context.Background(), "HEAD")
		if err != nil {
			t.Fatalf("Diff: %v", err)
		}

		if len(result.Added)+len(result.Modified)+len(result.Deleted) != 0 {
			t.Errorf("expected empty diff, got added=%d modified=%d deleted=%d",
				len(result.Added), len(result.Modified), len(result.Deleted))
		}
	})

	t.Run("invalid ref returns error", func(t *testing.T) {
		dir := t.TempDir()
		initDifferRepo(t, dir)
		writeAndCommit(t, dir, "a.go", "package main\n", "init")

		st := &mockStore{}
		d := NewDiffer(st, dir)
		_, err := d.Diff(context.Background(), "not-a-real-ref-xyz")
		if err == nil {
			t.Fatal("expected error for invalid ref, got nil")
		}
		if !strings.Contains(err.Error(), "git diff") {
			t.Errorf("error %q should mention 'git diff'", err.Error())
		}
	})
}
