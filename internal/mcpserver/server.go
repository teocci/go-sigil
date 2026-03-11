package mcpserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-sigil/internal/config"
	"go-sigil/internal/models"
	"go-sigil/internal/storage"
	"go-sigil/internal/store"
)

const (
	serverName    = "sigil"
	serverVersion = "v0.1.2"
)

// Server is the Sigil MCP stdio server.
// One instance runs for the lifetime of the sigil-mcp process.
type Server struct {
	cfg        *config.Config
	serverCWD  string
	cacheRoot  string
	sess       *mcpSession
	impl       *mcp.Server
	openRepoFn func(repoRoot, cacheRoot string) (store.SymbolStore, *storage.RepoMeta, error)
}

// NewServer creates and configures the MCP server, registering all 9 tools.
func NewServer(cfg *config.Config) (*Server, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	s := &Server{
		cfg:        cfg,
		serverCWD:  cwd,
		cacheRoot:  cfg.CacheRoot,
		sess:       newMCPSession(),
		openRepoFn: openRepo,
	}

	impl := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_search",
		Description: "Find symbols by name, pattern, or full-text search. Returns signatures and summaries only — no source code. ~15 tokens/result.",
	}, s.handleSearch)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_get",
		Description: "Retrieve symbol source by ID, or raw file content by path. Use ids for parsed symbols, files for unsupported file types.",
	}, s.handleGet)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_deps",
		Description: "Traverse the call graph for a symbol. Returns callers/callees with confidence levels and summaries — no source.",
	}, s.handleDeps)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_outline",
		Description: "List all symbols in a file as a structured outline with kinds, signatures, and summaries. No source code.",
	}, s.handleOutline)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_tree",
		Description: "Repository file structure: directories, files, languages, symbol counts. ~80 tokens for a 50-file repo.",
	}, s.handleTree)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_overview",
		Description: "High-level repo summary: languages, packages, symbol counts, index age. Call this first at every session start.",
	}, s.handleOverview)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_env",
		Description: "Inspect .env file configuration: which variables are set, empty, or placeholder. Never returns values. Call first for auth/connection debugging.",
	}, s.handleEnv)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_diff",
		Description: "Symbol-level diff since a git ref (HEAD~1, branch name, commit SHA). Groups changes as added, modified, or deleted.",
	}, s.handleDiff)

	mcp.AddTool(impl, &mcp.Tool{
		Name:        "sigil_status",
		Description: "Index health: last indexed commit, symbol counts, possible unresolved call edges, session savings summary.",
	}, s.handleStatus)

	s.impl = impl
	return s, nil
}

// Run starts the MCP server over stdin/stdout (stdio transport).
// Blocks until the client disconnects or ctx is cancelled.
// IMPORTANT: nothing must write to stdout while this is running.
func (s *Server) Run(ctx context.Context) error {
	return s.impl.Run(ctx, &mcp.StdioTransport{})
}

// buildMeta constructs the metadata envelope using the store for repo totals.
func (s *Server) buildMeta(ctx context.Context, start time.Time, tokensSaved int, st store.SymbolStore) Metadata {
	sessID, callCount, sessSaved, sessStart := s.sess.snapshot()

	var repoTotal models.SavingsSummary
	if st != nil {
		repoTotal, _ = st.GetRepoSavings(ctx)
	}

	return Metadata{
		TimingMs:    float64(time.Since(start).Milliseconds()),
		TokensSaved: tokensSaved,
		Session: models.SessionInfo{
			ID:          sessID,
			TokensSaved: sessSaved,
			CallCount:   callCount,
			StartedAt:   sessStart.Format(time.RFC3339),
		},
		RepoTotal: repoTotal,
	}
}

// buildMetaNoStore constructs metadata without repo total (for tools that don't open a store).
func (s *Server) buildMetaNoStore(start time.Time, tokensSaved int) Metadata {
	sessID, callCount, sessSaved, sessStart := s.sess.snapshot()
	return Metadata{
		TimingMs:    float64(time.Since(start).Milliseconds()),
		TokensSaved: tokensSaved,
		Session: models.SessionInfo{
			ID:          sessID,
			TokensSaved: sessSaved,
			CallCount:   callCount,
			StartedAt:   sessStart.Format(time.RFC3339),
		},
	}
}
