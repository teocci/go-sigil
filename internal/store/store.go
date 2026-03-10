// Package store defines the data access interface for the Sigil symbol index.
// The SymbolStore interface is defined here (consumer side) and implemented
// by the SQLite backend. Consumers of this interface (service layer) depend
// only on this interface, enabling mock implementations in tests.
package store

import (
	"context"

	"go-sigil/internal/models"
)

// SearchOptions controls filtering and pagination for symbol searches.
type SearchOptions struct {
	Kind     string // filter by symbol kind (e.g. "function", "method")
	Language string // filter by language (e.g. "go", "typescript")
	File     string // filter by source file path
	Limit    int    // max results; 0 means use default
	Offset   int    // pagination offset
}

// SymbolStore defines all data access operations for the Sigil index.
// All methods accept a context for cancellation and timeout propagation.
//
// Implementations must be safe for concurrent use (WAL mode SQLite satisfies this).
type SymbolStore interface {
	// File operations

	// UpsertFile inserts or replaces a file tracking record.
	UpsertFile(ctx context.Context, f models.File) error
	// GetFile returns the file record for the given path, or nil if not found.
	GetFile(ctx context.Context, path string) (*models.File, error)
	// DeleteFile removes the file record and cascades to its symbols.
	DeleteFile(ctx context.Context, path string) error
	// ListFiles returns all tracked files.
	ListFiles(ctx context.Context) ([]models.File, error)

	// Symbol operations

	// ReplaceFileSymbols atomically replaces all symbols for a file.
	// Deletes existing symbols (cascading to call_edges), then inserts the new set
	// in a single transaction.
	ReplaceFileSymbols(ctx context.Context, file string, symbols []models.Symbol) error
	// GetSymbolByID retrieves a symbol by its ID. Returns nil if not found.
	GetSymbolByID(ctx context.Context, id string) (*models.Symbol, error)
	// GetSymbolsByFile retrieves all symbols for a file, ordered by line_start.
	GetSymbolsByFile(ctx context.Context, file string) ([]models.Symbol, error)
	// GetSymbolsByIDs retrieves symbols for a set of IDs. Missing IDs are silently skipped.
	GetSymbolsByIDs(ctx context.Context, ids []string) ([]models.Symbol, error)
	// SearchSymbols performs an FTS5 full-text search with optional filters.
	SearchSymbols(ctx context.Context, query string, opts SearchOptions) ([]models.Symbol, error)
	// MarkPossibleUnresolved flags all symbols in a file as possibly having unresolved
	// call edges, used before removing a file during sync.
	MarkPossibleUnresolved(ctx context.Context, file string) error

	// Call graph operations

	// ReplaceFileEdges atomically replaces all call edges originating from symbols in a file.
	ReplaceFileEdges(ctx context.Context, file string, edges []models.CallEdge) error
	// GetCalls returns all edges where the symbol is the caller.
	// depth is reserved for future multi-hop support; currently depth=1 is always used.
	GetCalls(ctx context.Context, symbolID string, depth int) ([]models.CallEdge, error)
	// GetCalledBy returns all edges where the symbol is the callee.
	GetCalledBy(ctx context.Context, symbolID string, depth int) ([]models.CallEdge, error)

	// Savings ledger

	// AppendSavings records a single token savings measurement.
	AppendSavings(ctx context.Context, entry models.SavingsEntry) error
	// GetSessionSavings returns aggregated savings for a session.
	GetSessionSavings(ctx context.Context, sessionID string) (models.SavingsSummary, error)
	// GetRepoSavings returns aggregated savings across all sessions for this repo.
	GetRepoSavings(ctx context.Context) (models.SavingsSummary, error)

	// Lifecycle

	// Close releases all database resources.
	Close() error
}
