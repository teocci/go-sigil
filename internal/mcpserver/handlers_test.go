package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-sigil/internal/models"
	"go-sigil/internal/storage"
	"go-sigil/internal/store"
)

// mockStore is a test double for store.SymbolStore.
type mockStore struct {
	searchSymbolsFn       func(ctx context.Context, q string, opts store.SearchOptions) ([]models.Symbol, error)
	getSymbolsByFileFn    func(ctx context.Context, file string) ([]models.Symbol, error)
	getSymbolsByIDsFn     func(ctx context.Context, ids []string) ([]models.Symbol, error)
	getSymbolByIDFn       func(ctx context.Context, id string) (*models.Symbol, error)
	getCallsFn            func(ctx context.Context, id string, depth int) ([]models.CallEdge, error)
	getCalledByFn         func(ctx context.Context, id string, depth int) ([]models.CallEdge, error)
	listFilesFn           func(ctx context.Context) ([]models.File, error)
	countSymbolsFn        func(ctx context.Context) (int, error)
	countFilesFn          func(ctx context.Context) (int, error)
	countPossibleFn       func(ctx context.Context) (int, error)
	getLanguageStatsFn    func(ctx context.Context) ([]models.LanguageStat, error)
	getPackageStatsFn     func(ctx context.Context) ([]models.PackageStat, error)
	getRepoSavingsFn      func(ctx context.Context) (models.SavingsSummary, error)
}

func (m *mockStore) SearchSymbols(ctx context.Context, q string, opts store.SearchOptions) ([]models.Symbol, error) {
	if m.searchSymbolsFn != nil {
		return m.searchSymbolsFn(ctx, q, opts)
	}
	return nil, nil
}
func (m *mockStore) GetSymbolsByFile(ctx context.Context, file string) ([]models.Symbol, error) {
	if m.getSymbolsByFileFn != nil {
		return m.getSymbolsByFileFn(ctx, file)
	}
	return nil, nil
}
func (m *mockStore) GetSymbolsByIDs(ctx context.Context, ids []string) ([]models.Symbol, error) {
	if m.getSymbolsByIDsFn != nil {
		return m.getSymbolsByIDsFn(ctx, ids)
	}
	return nil, nil
}
func (m *mockStore) GetSymbolByID(ctx context.Context, id string) (*models.Symbol, error) {
	if m.getSymbolByIDFn != nil {
		return m.getSymbolByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockStore) GetCalls(ctx context.Context, id string, depth int) ([]models.CallEdge, error) {
	if m.getCallsFn != nil {
		return m.getCallsFn(ctx, id, depth)
	}
	return nil, nil
}
func (m *mockStore) GetCalledBy(ctx context.Context, id string, depth int) ([]models.CallEdge, error) {
	if m.getCalledByFn != nil {
		return m.getCalledByFn(ctx, id, depth)
	}
	return nil, nil
}
func (m *mockStore) ListFiles(ctx context.Context) ([]models.File, error) {
	if m.listFilesFn != nil {
		return m.listFilesFn(ctx)
	}
	return nil, nil
}
func (m *mockStore) CountSymbols(ctx context.Context) (int, error) {
	if m.countSymbolsFn != nil {
		return m.countSymbolsFn(ctx)
	}
	return 0, nil
}
func (m *mockStore) CountFiles(ctx context.Context) (int, error) {
	if m.countFilesFn != nil {
		return m.countFilesFn(ctx)
	}
	return 0, nil
}
func (m *mockStore) CountPossibleUnresolved(ctx context.Context) (int, error) {
	if m.countPossibleFn != nil {
		return m.countPossibleFn(ctx)
	}
	return 0, nil
}
func (m *mockStore) GetLanguageStats(ctx context.Context) ([]models.LanguageStat, error) {
	if m.getLanguageStatsFn != nil {
		return m.getLanguageStatsFn(ctx)
	}
	return nil, nil
}
func (m *mockStore) GetPackageStats(ctx context.Context) ([]models.PackageStat, error) {
	if m.getPackageStatsFn != nil {
		return m.getPackageStatsFn(ctx)
	}
	return nil, nil
}
func (m *mockStore) GetRepoSavings(ctx context.Context) (models.SavingsSummary, error) {
	if m.getRepoSavingsFn != nil {
		return m.getRepoSavingsFn(ctx)
	}
	return models.SavingsSummary{}, nil
}
func (m *mockStore) UpsertFile(ctx context.Context, f models.File) error         { return nil }
func (m *mockStore) GetFile(ctx context.Context, path string) (*models.File, error) { return nil, nil }
func (m *mockStore) DeleteFile(ctx context.Context, path string) error           { return nil }
func (m *mockStore) ReplaceFileSymbols(ctx context.Context, file string, syms []models.Symbol) error {
	return nil
}
func (m *mockStore) MarkPossibleUnresolved(ctx context.Context, file string) error { return nil }
func (m *mockStore) ReplaceFileEdges(ctx context.Context, file string, edges []models.CallEdge) error {
	return nil
}
func (m *mockStore) AppendSavings(ctx context.Context, e models.SavingsEntry) error {
	return nil
}
func (m *mockStore) GetSessionSavings(ctx context.Context, id string) (models.SavingsSummary, error) {
	return models.SavingsSummary{}, nil
}
func (m *mockStore) ListSessions(ctx context.Context) ([]models.SavingsSession, error) {
	return nil, nil
}
func (m *mockStore) GetTopSessions(ctx context.Context, n int) ([]models.SavingsSession, error) {
	return nil, nil
}
func (m *mockStore) Close() error { return nil }

// newTestServer builds a Server wired to the provided mock store.
// serverCWD is set to the current working directory (a real git repo).
func newTestServer(t *testing.T, st store.SymbolStore, meta *storage.RepoMeta) *Server {
	t.Helper()
	s := &Server{
		serverCWD: ".",
		cacheRoot: t.TempDir(),
		sess:      newMCPSession(),
		openRepoFn: func(repoRoot, cacheRoot string) (store.SymbolStore, *storage.RepoMeta, error) {
			return st, meta, nil
		},
	}
	return s
}

// newTestServerError builds a Server whose openRepoFn always returns an error.
func newTestServerError(t *testing.T, msg string) *Server {
	t.Helper()
	s := &Server{
		serverCWD: ".",
		cacheRoot: t.TempDir(),
		sess:      newMCPSession(),
		openRepoFn: func(repoRoot, cacheRoot string) (store.SymbolStore, *storage.RepoMeta, error) {
			return nil, nil, errors.New(msg)
		},
	}
	return s
}

// textContent extracts the first text content from a tool result.
func textContent(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if len(r.Content) == 0 {
		t.Fatal("empty content in tool result")
	}
	tc, ok := r.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", r.Content[0])
	}
	return tc.Text
}

// ---- sigil_search -------------------------------------------------------

func TestHandleSearch_HappyPath(t *testing.T) {
	sym := models.Symbol{ID: "aabb0001", Kind: "function", Name: "Search"}
	st := &mockStore{
		searchSymbolsFn: func(_ context.Context, _ string, _ store.SearchOptions) ([]models.Symbol, error) {
			return []models.Symbol{sym}, nil
		},
	}
	s := newTestServer(t, st, nil)
	result, _, err := s.handleSearch(context.Background(), nil, SearchInput{Query: "Search"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
	text := textContent(t, result)
	var out struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Total != 1 {
		t.Errorf("total = %d, want 1", out.Total)
	}
}

func TestHandleSearch_OpenRepoError(t *testing.T) {
	s := newTestServerError(t, "repo not indexed")
	result, _, err := s.handleSearch(context.Background(), nil, SearchInput{Query: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestHandleSearch_EmptyResults(t *testing.T) {
	st := &mockStore{}
	s := newTestServer(t, st, nil)
	result, _, err := s.handleSearch(context.Background(), nil, SearchInput{Query: "nothing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
	text := textContent(t, result)
	var out struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Total != 0 {
		t.Errorf("total = %d, want 0", out.Total)
	}
}

// ---- sigil_deps --------------------------------------------------------

func TestHandleDeps_HappyPath(t *testing.T) {
	sym := &models.Symbol{ID: "root0001", Name: "Handler"}
	st := &mockStore{
		getSymbolByIDFn: func(_ context.Context, id string) (*models.Symbol, error) {
			if id == "root0001" {
				return sym, nil
			}
			return nil, nil
		},
		getCallsFn: func(_ context.Context, _ string, _ int) ([]models.CallEdge, error) {
			return []models.CallEdge{{CallerID: "root0001", CalleeID: "callee01", Confidence: "static"}}, nil
		},
	}
	s := newTestServer(t, st, nil)
	result, _, err := s.handleDeps(context.Background(), nil, DepsInput{ID: "root0001", Direction: "calls"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
}

func TestHandleDeps_OpenRepoError(t *testing.T) {
	s := newTestServerError(t, "not indexed")
	result, _, err := s.handleDeps(context.Background(), nil, DepsInput{ID: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestHandleDeps_DefaultDirectionBoth(t *testing.T) {
	sym := &models.Symbol{ID: "sym00001", Name: "Fn"}
	st := &mockStore{
		getSymbolByIDFn: func(_ context.Context, _ string) (*models.Symbol, error) { return sym, nil },
	}
	s := newTestServer(t, st, nil)
	// Direction empty → defaults to "both"
	result, _, err := s.handleDeps(context.Background(), nil, DepsInput{ID: "sym00001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
}

// ---- sigil_outline -------------------------------------------------------

func TestHandleOutline_HappyPath(t *testing.T) {
	syms := []models.Symbol{
		{ID: "aabb0001", Kind: "function", Name: "Init"},
	}
	st := &mockStore{
		getSymbolsByFileFn: func(_ context.Context, _ string) ([]models.Symbol, error) {
			return syms, nil
		},
	}
	s := newTestServer(t, st, nil)
	result, _, err := s.handleOutline(context.Background(), nil, OutlineInput{File: "internal/foo.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
	text := textContent(t, result)
	var out struct {
		Symbols []struct{ ID string } `json:"symbols"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Symbols) != 1 {
		t.Errorf("len(symbols) = %d, want 1", len(out.Symbols))
	}
}

func TestHandleOutline_OpenRepoError(t *testing.T) {
	s := newTestServerError(t, "not indexed")
	result, _, err := s.handleOutline(context.Background(), nil, OutlineInput{File: "foo.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

// ---- sigil_tree ----------------------------------------------------------

func TestHandleTree_HappyPath(t *testing.T) {
	st := &mockStore{
		listFilesFn: func(_ context.Context) ([]models.File, error) {
			return []models.File{{Path: "cmd/main.go"}}, nil
		},
	}
	s := newTestServer(t, st, nil)
	result, _, err := s.handleTree(context.Background(), nil, TreeInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
}

// ---- sigil_overview -------------------------------------------------------

func TestHandleOverview_HappyPath(t *testing.T) {
	st := &mockStore{
		countSymbolsFn: func(_ context.Context) (int, error) { return 50, nil },
		countFilesFn:   func(_ context.Context) (int, error) { return 5, nil },
	}
	meta := &storage.RepoMeta{Name: "myrepo"}
	s := newTestServer(t, st, meta)
	result, _, err := s.handleOverview(context.Background(), nil, OverviewInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
}

// ---- sigil_status --------------------------------------------------------

func TestHandleStatus_HappyPath(t *testing.T) {
	st := &mockStore{
		countFilesFn:   func(_ context.Context) (int, error) { return 10, nil },
		countSymbolsFn: func(_ context.Context) (int, error) { return 100, nil },
	}
	s := newTestServer(t, st, nil)
	result, _, err := s.handleStatus(context.Background(), nil, StatusInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(t, result))
	}
	text := textContent(t, result)
	var out struct {
		TotalFiles   int `json:"total_files"`
		TotalSymbols int `json:"total_symbols"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.TotalFiles != 10 {
		t.Errorf("total_files = %d, want 10", out.TotalFiles)
	}
}

func TestHandleStatus_OpenRepoError(t *testing.T) {
	s := newTestServerError(t, "not indexed")
	result, _, err := s.handleStatus(context.Background(), nil, StatusInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

// ---- helper tests -------------------------------------------------------

func TestToolError(t *testing.T) {
	r := toolError(errors.New("something went wrong"))
	if !r.IsError {
		t.Error("expected IsError=true")
	}
	if len(r.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

func TestJsonResult(t *testing.T) {
	r := jsonResult(map[string]string{"key": "value"})
	if r.IsError {
		t.Error("expected IsError=false")
	}
	if len(r.Content) == 0 {
		t.Error("expected non-empty content")
	}
	tc, ok := r.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", r.Content[0])
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(tc.Text), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["key"] != "value" {
		t.Errorf("key = %q, want value", m["key"])
	}
}
