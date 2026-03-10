package service

import (
	"context"

	"go-sigil/internal/models"
	"go-sigil/internal/store"
)

// mockStore is a test double for store.SymbolStore.
// Each method has a corresponding function field; nil fields return zero values.
type mockStore struct {
	upsertFileFn              func(ctx context.Context, f models.File) error
	getFileFn                 func(ctx context.Context, path string) (*models.File, error)
	deleteFileFn              func(ctx context.Context, path string) error
	listFilesFn               func(ctx context.Context) ([]models.File, error)
	replaceFileSymbolsFn      func(ctx context.Context, file string, symbols []models.Symbol) error
	getSymbolByIDFn           func(ctx context.Context, id string) (*models.Symbol, error)
	getSymbolsByFileFn        func(ctx context.Context, file string) ([]models.Symbol, error)
	getSymbolsByIDsFn         func(ctx context.Context, ids []string) ([]models.Symbol, error)
	searchSymbolsFn           func(ctx context.Context, query string, opts store.SearchOptions) ([]models.Symbol, error)
	markPossibleUnresolvedFn  func(ctx context.Context, file string) error
	replaceFileEdgesFn        func(ctx context.Context, file string, edges []models.CallEdge) error
	getCallsFn                func(ctx context.Context, symbolID string, depth int) ([]models.CallEdge, error)
	getCalledByFn             func(ctx context.Context, symbolID string, depth int) ([]models.CallEdge, error)
	appendSavingsFn           func(ctx context.Context, entry models.SavingsEntry) error
	getSessionSavingsFn       func(ctx context.Context, sessionID string) (models.SavingsSummary, error)
	getRepoSavingsFn          func(ctx context.Context) (models.SavingsSummary, error)
	countSymbolsFn            func(ctx context.Context) (int, error)
	countFilesFn              func(ctx context.Context) (int, error)
	countPossibleUnresolvedFn func(ctx context.Context) (int, error)
	getLanguageStatsFn        func(ctx context.Context) ([]models.LanguageStat, error)
	getPackageStatsFn         func(ctx context.Context) ([]models.PackageStat, error)
	listSessionsFn            func(ctx context.Context) ([]models.SavingsSession, error)
	getTopSessionsFn          func(ctx context.Context, n int) ([]models.SavingsSession, error)
}

func (m *mockStore) UpsertFile(ctx context.Context, f models.File) error {
	if m.upsertFileFn != nil {
		return m.upsertFileFn(ctx, f)
	}
	return nil
}

func (m *mockStore) GetFile(ctx context.Context, path string) (*models.File, error) {
	if m.getFileFn != nil {
		return m.getFileFn(ctx, path)
	}
	return nil, nil
}

func (m *mockStore) DeleteFile(ctx context.Context, path string) error {
	if m.deleteFileFn != nil {
		return m.deleteFileFn(ctx, path)
	}
	return nil
}

func (m *mockStore) ListFiles(ctx context.Context) ([]models.File, error) {
	if m.listFilesFn != nil {
		return m.listFilesFn(ctx)
	}
	return nil, nil
}

func (m *mockStore) ReplaceFileSymbols(ctx context.Context, file string, symbols []models.Symbol) error {
	if m.replaceFileSymbolsFn != nil {
		return m.replaceFileSymbolsFn(ctx, file, symbols)
	}
	return nil
}

func (m *mockStore) GetSymbolByID(ctx context.Context, id string) (*models.Symbol, error) {
	if m.getSymbolByIDFn != nil {
		return m.getSymbolByIDFn(ctx, id)
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

func (m *mockStore) SearchSymbols(ctx context.Context, query string, opts store.SearchOptions) ([]models.Symbol, error) {
	if m.searchSymbolsFn != nil {
		return m.searchSymbolsFn(ctx, query, opts)
	}
	return nil, nil
}

func (m *mockStore) MarkPossibleUnresolved(ctx context.Context, file string) error {
	if m.markPossibleUnresolvedFn != nil {
		return m.markPossibleUnresolvedFn(ctx, file)
	}
	return nil
}

func (m *mockStore) ReplaceFileEdges(ctx context.Context, file string, edges []models.CallEdge) error {
	if m.replaceFileEdgesFn != nil {
		return m.replaceFileEdgesFn(ctx, file, edges)
	}
	return nil
}

func (m *mockStore) GetCalls(ctx context.Context, symbolID string, depth int) ([]models.CallEdge, error) {
	if m.getCallsFn != nil {
		return m.getCallsFn(ctx, symbolID, depth)
	}
	return nil, nil
}

func (m *mockStore) GetCalledBy(ctx context.Context, symbolID string, depth int) ([]models.CallEdge, error) {
	if m.getCalledByFn != nil {
		return m.getCalledByFn(ctx, symbolID, depth)
	}
	return nil, nil
}

func (m *mockStore) AppendSavings(ctx context.Context, entry models.SavingsEntry) error {
	if m.appendSavingsFn != nil {
		return m.appendSavingsFn(ctx, entry)
	}
	return nil
}

func (m *mockStore) GetSessionSavings(ctx context.Context, sessionID string) (models.SavingsSummary, error) {
	if m.getSessionSavingsFn != nil {
		return m.getSessionSavingsFn(ctx, sessionID)
	}
	return models.SavingsSummary{}, nil
}

func (m *mockStore) GetRepoSavings(ctx context.Context) (models.SavingsSummary, error) {
	if m.getRepoSavingsFn != nil {
		return m.getRepoSavingsFn(ctx)
	}
	return models.SavingsSummary{}, nil
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
	if m.countPossibleUnresolvedFn != nil {
		return m.countPossibleUnresolvedFn(ctx)
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

func (m *mockStore) ListSessions(ctx context.Context) ([]models.SavingsSession, error) {
	if m.listSessionsFn != nil {
		return m.listSessionsFn(ctx)
	}
	return nil, nil
}

func (m *mockStore) GetTopSessions(ctx context.Context, n int) ([]models.SavingsSession, error) {
	if m.getTopSessionsFn != nil {
		return m.getTopSessionsFn(ctx, n)
	}
	return nil, nil
}

func (m *mockStore) Close() error { return nil }
