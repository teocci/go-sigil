package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go-sigil/internal/models"
)

func TestRetriever_Get(t *testing.T) {
	ctx := context.Background()

	// Create a temp repo root with a real file to test source extraction.
	tmpDir := t.TempDir()
	relPath := "internal/foo.go"
	absPath := filepath.Join(tmpDir, "internal", "foo.go")
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := "package foo\n\nfunc Hello() string {\n\treturn \"world\"\n}\n"
	if err := os.WriteFile(absPath, []byte(src), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	byteStart := 13 // after "package foo\n\n"
	byteEnd := byteStart + len("func Hello() string {\n\treturn \"world\"\n}\n")

	sym := models.Symbol{
		ID:            "aabb0001",
		QualifiedName: "internal/foo.Hello",
		File:          relPath,
		LineStart:     3,
		LineEnd:       5,
		ByteStart:     &byteStart,
		ByteEnd:       &byteEnd,
	}

	t.Run("retrieves symbol source by ID", func(t *testing.T) {
		st := &mockStore{
			getSymbolsByIDsFn: func(_ context.Context, ids []string) ([]models.Symbol, error) {
				if len(ids) == 1 && ids[0] == "aabb0001" {
					return []models.Symbol{sym}, nil
				}
				return nil, nil
			},
		}
		r := NewRetriever(st, "", tmpDir)
		result, err := r.Get(ctx, []string{"aabb0001"}, nil, 0)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if len(result.Symbols) != 1 {
			t.Fatalf("len(Symbols) = %d, want 1", len(result.Symbols))
		}
		got := result.Symbols[0]
		if got.ID != sym.ID {
			t.Errorf("ID = %q, want %q", got.ID, sym.ID)
		}
		if got.Source == "" {
			t.Error("expected non-empty Source")
		}
	})

	t.Run("redacted symbol returns placeholder", func(t *testing.T) {
		redacted := models.Symbol{
			ID:            "redact01",
			QualifiedName: "internal/secret.key",
			File:          relPath,
			ByteStart:     nil,
			ByteEnd:       nil,
		}
		st := &mockStore{
			getSymbolsByIDsFn: func(_ context.Context, _ []string) ([]models.Symbol, error) {
				return []models.Symbol{redacted}, nil
			},
		}
		r := NewRetriever(st, "", tmpDir)
		result, err := r.Get(ctx, []string{"redact01"}, nil, 0)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if len(result.Symbols) == 0 {
			t.Fatal("expected 1 symbol")
		}
		if result.Symbols[0].Source != "[redacted]" {
			t.Errorf("Source = %q, want [redacted]", result.Symbols[0].Source)
		}
	})

	t.Run("context lines clamped to [0,50]", func(t *testing.T) {
		st := &mockStore{
			getSymbolsByIDsFn: func(_ context.Context, _ []string) ([]models.Symbol, error) {
				return []models.Symbol{sym}, nil
			},
		}
		r := NewRetriever(st, "", tmpDir)
		// negative context lines should not panic
		_, err := r.Get(ctx, []string{"aabb0001"}, nil, -5)
		if err != nil {
			t.Fatalf("Get() with negative context: %v", err)
		}
		// oversized context lines should not panic
		_, err = r.Get(ctx, []string{"aabb0001"}, nil, 200)
		if err != nil {
			t.Fatalf("Get() with large context: %v", err)
		}
	})

	t.Run("file content retrieved by path", func(t *testing.T) {
		st := &mockStore{}
		r := NewRetriever(st, "", tmpDir)
		result, err := r.Get(ctx, nil, []string{relPath}, 0)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if len(result.Files) != 1 {
			t.Fatalf("len(Files) = %d, want 1", len(result.Files))
		}
		fc := result.Files[0]
		if fc.Path != relPath {
			t.Errorf("Path = %q, want %q", fc.Path, relPath)
		}
		if fc.Content != src {
			t.Errorf("Content mismatch")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		st := &mockStore{}
		r := NewRetriever(st, "", tmpDir)
		_, err := r.Get(ctx, nil, []string{"nonexistent/file.go"}, 0)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty ids and files returns empty result", func(t *testing.T) {
		st := &mockStore{}
		r := NewRetriever(st, "", tmpDir)
		result, err := r.Get(ctx, nil, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Symbols) != 0 || len(result.Files) != 0 {
			t.Error("expected empty result")
		}
	})
}

func TestExpandBackForward(t *testing.T) {
	data := []byte("line1\nline2\nline3\nline4\nline5\n")
	//                                          ^18 = start of "line4"

	t.Run("expandBack 2 lines from mid-file", func(t *testing.T) {
		pos := 18 // start of "line4\n"
		got := expandBack(data, pos, 2)
		// 2 newlines back from 18: crosses '\n' at 17 (line3 end, count=1)
		// and '\n' at 11 (line2 end, count=2); exits loop with i=10, returns 11
		want := 11
		if got != want {
			t.Errorf("expandBack = %d, want %d", got, want)
		}
	})

	t.Run("expandBack from beginning returns 0", func(t *testing.T) {
		got := expandBack(data, 3, 10)
		if got != 0 {
			t.Errorf("expandBack = %d, want 0", got)
		}
	})

	t.Run("expandForward 1 line from mid-file", func(t *testing.T) {
		pos := 18 // start of "line4\n"
		got := expandForward(data, pos, 1)
		// 1 newline forward from 18 = position 24 (after "line4\n")
		want := 24
		if got != want {
			t.Errorf("expandForward = %d, want %d", got, want)
		}
	})
}
