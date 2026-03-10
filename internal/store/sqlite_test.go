package store_test

import (
	"context"
	"testing"
	"time"

	"go-sigil/internal/db"
	"go-sigil/internal/models"
	"go-sigil/internal/store"
)

// openTestStore creates an in-memory SQLite store with schema applied.
func openTestStore(t *testing.T) store.SymbolStore {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Run(d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return store.New(d)
}

func TestFileCRUD(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	f := models.File{
		Path:        "internal/foo.go",
		BlobSHA:     "abc123",
		LastIndexed: time.Now().UTC().Format(time.RFC3339),
	}

	// Upsert
	if err := s.UpsertFile(ctx, f); err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	// Get
	got, err := s.GetFile(ctx, f.Path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if got == nil {
		t.Fatal("GetFile returned nil")
	}
	if got.BlobSHA != f.BlobSHA {
		t.Errorf("BlobSHA = %q, want %q", got.BlobSHA, f.BlobSHA)
	}

	// Update via upsert
	f.BlobSHA = "def456"
	if err := s.UpsertFile(ctx, f); err != nil {
		t.Fatalf("UpsertFile (update): %v", err)
	}
	got, _ = s.GetFile(ctx, f.Path)
	if got.BlobSHA != "def456" {
		t.Errorf("after update BlobSHA = %q, want def456", got.BlobSHA)
	}

	// List
	files, err := s.ListFiles(ctx)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("ListFiles len = %d, want 1", len(files))
	}

	// Delete
	if err := s.DeleteFile(ctx, f.Path); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	got, _ = s.GetFile(ctx, f.Path)
	if got != nil {
		t.Error("GetFile after delete should return nil")
	}
}

func TestGetFile_NotFound(t *testing.T) {
	s := openTestStore(t)
	got, err := s.GetFile(context.Background(), "missing.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing file")
	}
}

func makeSymbol(id, name, file string) models.Symbol {
	bs, be := 0, 50
	return models.Symbol{
		ID:            id,
		Kind:          "function",
		Name:          name,
		QualifiedName: file + "." + name,
		Language:      "go",
		File:          file,
		ByteStart:     &bs,
		ByteEnd:       &be,
		LineStart:     1,
		LineEnd:       5,
		Signature:     "func " + name + "()",
		Summary:       "does " + name,
		ContentHash:   "hash" + id,
		IndexedAt:     time.Now().UTC().Format(time.RFC3339),
	}
}

func TestReplaceFileSymbols(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	file := "internal/foo.go"
	syms := []models.Symbol{
		makeSymbol("id1", "Alpha", file),
		makeSymbol("id2", "Beta", file),
	}

	// Insert first batch
	if err := s.ReplaceFileSymbols(ctx, file, syms); err != nil {
		t.Fatalf("ReplaceFileSymbols: %v", err)
	}

	got, err := s.GetSymbolsByFile(ctx, file)
	if err != nil {
		t.Fatalf("GetSymbolsByFile: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}

	// Replace with only one symbol — old ones should be gone
	newSyms := []models.Symbol{makeSymbol("id3", "Gamma", file)}
	if err := s.ReplaceFileSymbols(ctx, file, newSyms); err != nil {
		t.Fatalf("ReplaceFileSymbols (replace): %v", err)
	}

	got, _ = s.GetSymbolsByFile(ctx, file)
	if len(got) != 1 {
		t.Errorf("after replace len = %d, want 1", len(got))
	}
	if got[0].Name != "Gamma" {
		t.Errorf("name = %q, want Gamma", got[0].Name)
	}
}

func TestGetSymbolByID(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sym := makeSymbol("abc00001", "TestFunc", "main.go")
	_ = s.ReplaceFileSymbols(ctx, "main.go", []models.Symbol{sym})

	got, err := s.GetSymbolByID(ctx, "abc00001")
	if err != nil {
		t.Fatalf("GetSymbolByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected symbol, got nil")
	}
	if got.Name != "TestFunc" {
		t.Errorf("Name = %q, want TestFunc", got.Name)
	}

	// Missing
	missing, err := s.GetSymbolByID(ctx, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for missing symbol")
	}
}

func TestGetSymbolsByIDs(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	syms := []models.Symbol{
		makeSymbol("x1", "One", "a.go"),
		makeSymbol("x2", "Two", "a.go"),
		makeSymbol("x3", "Three", "a.go"),
	}
	_ = s.ReplaceFileSymbols(ctx, "a.go", syms)

	got, err := s.GetSymbolsByIDs(ctx, []string{"x1", "x3"})
	if err != nil {
		t.Fatalf("GetSymbolsByIDs: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

func TestMarkPossibleUnresolved(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	file := "pkg/bar.go"
	_ = s.ReplaceFileSymbols(ctx, file, []models.Symbol{makeSymbol("m1", "Foo", file)})

	if err := s.MarkPossibleUnresolved(ctx, file); err != nil {
		t.Fatalf("MarkPossibleUnresolved: %v", err)
	}

	sym, _ := s.GetSymbolByID(ctx, "m1")
	if !sym.PossibleUnresolved {
		t.Error("expected PossibleUnresolved = true")
	}
}

func TestSearchSymbols(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_ = s.ReplaceFileSymbols(ctx, "search.go", []models.Symbol{
		makeSymbol("s1", "ParseRequest", "search.go"),
		makeSymbol("s2", "FormatResponse", "search.go"),
	})

	results, err := s.SearchSymbols(ctx, "Parse*", store.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("SearchSymbols: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("results len = %d, want 1", len(results))
	}
	if len(results) > 0 && results[0].Name != "ParseRequest" {
		t.Errorf("name = %q, want ParseRequest", results[0].Name)
	}
}

func TestSearchSymbols_KindFilter(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	method := makeSymbol("sm1", "Handle", "http.go")
	method.Kind = "method"
	fn := makeSymbol("sf1", "HandleFunc", "http.go")
	fn.Kind = "function"
	_ = s.ReplaceFileSymbols(ctx, "http.go", []models.Symbol{method, fn})

	results, err := s.SearchSymbols(ctx, "Handle*", store.SearchOptions{Kind: "method", Limit: 10})
	if err != nil {
		t.Fatalf("SearchSymbols: %v", err)
	}
	if len(results) != 1 || results[0].Kind != "method" {
		t.Errorf("expected 1 method result, got %d", len(results))
	}
}

func TestCallEdges(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	file := "edges.go"
	caller := makeSymbol("c1", "Caller", file)
	callee := makeSymbol("ca1", "Callee", file)
	_ = s.ReplaceFileSymbols(ctx, file, []models.Symbol{caller, callee})

	edges := []models.CallEdge{
		{CallerID: "c1", CalleeID: "ca1", Confidence: "static"},
	}
	if err := s.ReplaceFileEdges(ctx, file, edges); err != nil {
		t.Fatalf("ReplaceFileEdges: %v", err)
	}

	// GetCalls
	calls, err := s.GetCalls(ctx, "c1", 1)
	if err != nil {
		t.Fatalf("GetCalls: %v", err)
	}
	if len(calls) != 1 {
		t.Errorf("GetCalls len = %d, want 1", len(calls))
	}
	if calls[0].CalleeID != "ca1" {
		t.Errorf("CalleeID = %q, want ca1", calls[0].CalleeID)
	}

	// GetCalledBy
	calledBy, err := s.GetCalledBy(ctx, "ca1", 1)
	if err != nil {
		t.Fatalf("GetCalledBy: %v", err)
	}
	if len(calledBy) != 1 || calledBy[0].CallerID != "c1" {
		t.Errorf("GetCalledBy unexpected: %+v", calledBy)
	}

	// Replace with empty — edges gone
	if err := s.ReplaceFileEdges(ctx, file, nil); err != nil {
		t.Fatalf("ReplaceFileEdges (empty): %v", err)
	}
	calls, _ = s.GetCalls(ctx, "c1", 1)
	if len(calls) != 0 {
		t.Errorf("after empty replace, GetCalls len = %d, want 0", len(calls))
	}
}

func TestSavingsLog(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := "sess-001"
	entries := []models.SavingsEntry{
		{SessionID: sess, ToolName: "sigil_search", TimingMs: 12.5, TokensSaved: 500},
		{SessionID: sess, ToolName: "sigil_get", TimingMs: 8.3, TokensSaved: 300},
		{SessionID: "other-sess", ToolName: "sigil_search", TimingMs: 5.0, TokensSaved: 200},
	}
	for _, e := range entries {
		if err := s.AppendSavings(ctx, e); err != nil {
			t.Fatalf("AppendSavings: %v", err)
		}
	}

	sessSum, err := s.GetSessionSavings(ctx, sess)
	if err != nil {
		t.Fatalf("GetSessionSavings: %v", err)
	}
	if sessSum.TokensSaved != 800 {
		t.Errorf("session tokens_saved = %d, want 800", sessSum.TokensSaved)
	}
	if sessSum.CallCount != 2 {
		t.Errorf("session call_count = %d, want 2", sessSum.CallCount)
	}

	repoSum, err := s.GetRepoSavings(ctx)
	if err != nil {
		t.Fatalf("GetRepoSavings: %v", err)
	}
	if repoSum.TokensSaved != 1000 {
		t.Errorf("repo tokens_saved = %d, want 1000", repoSum.TokensSaved)
	}
	if repoSum.CallCount != 3 {
		t.Errorf("repo call_count = %d, want 3", repoSum.CallCount)
	}
}
