package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-sigil/internal/models"
	"go-sigil/internal/service"
	"go-sigil/internal/store"
)

// jsonResult marshals v to a JSON text content result.
func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.Marshal(v)
	if err != nil {
		return toolError(fmt.Errorf("marshal result: %w", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}
}

// toolError wraps an error as an MCP error tool result.
func toolError(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}

// appendSavings persists a savings record to the ledger.
func (s *Server) appendSavings(ctx context.Context, st store.SymbolStore, toolName string, timingMs float64, tokens int) {
	if st == nil || tokens <= 0 {
		return
	}
	sessID, _, _, _ := s.sess.snapshot()
	_ = st.AppendSavings(ctx, models.SavingsEntry{
		SessionID:   sessID,
		ToolName:    toolName,
		CallAt:      time.Now().UTC().Format(time.RFC3339),
		TimingMs:    timingMs,
		TokensSaved: tokens,
	})
}

// handleSearch implements sigil_search.
func (s *Server) handleSearch(ctx context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, _, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	result, err := service.NewSearcher(st).Search(ctx, input.Query, store.SearchOptions{
		Kind:     input.Kind,
		Language: input.Language,
		Limit:    limit,
	})
	if err != nil {
		return toolError(fmt.Errorf("search: %w", err)), struct{}{}, nil
	}

	tokens := result.Total * 15 // ~15 tokens saved per result (signature+summary vs full source)
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_search", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		Symbols  []models.Symbol `json:"symbols"`
		Total    int             `json:"total"`
		Metadata Metadata        `json:"metadata"`
	}{
		Symbols:  result.Symbols,
		Total:    result.Total,
		Metadata: s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleGet implements sigil_get.
func (s *Server) handleGet(ctx context.Context, _ *mcp.CallToolRequest, input GetInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, _, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	filesDir := ""
	retriever := service.NewRetriever(st, filesDir, repoRoot)
	result, err := retriever.Get(ctx, input.IDs, input.Files, input.IncludeContextLines)
	if err != nil {
		return toolError(fmt.Errorf("get: %w", err)), struct{}{}, nil
	}

	// Savings: (estimated_file_size - symbol_size) / 4 per symbol
	tokens := 0
	for _, sym := range result.Symbols {
		symBytes := len(sym.Source)
		estimated := 5000 // average file size estimate
		if symBytes < estimated {
			tokens += (estimated - symBytes) / 4
		}
	}
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_get", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		Symbols  []service.SymbolWithSource `json:"symbols,omitempty"`
		Files    []service.FileContent      `json:"files,omitempty"`
		Metadata Metadata                   `json:"metadata"`
	}{
		Symbols:  result.Symbols,
		Files:    result.Files,
		Metadata: s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleDeps implements sigil_deps.
func (s *Server) handleDeps(ctx context.Context, _ *mcp.CallToolRequest, input DepsInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, _, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	direction := input.Direction
	if direction == "" {
		direction = "both"
	}

	result, err := service.NewDeps(st).Trace(ctx, input.ID, direction, input.Depth)
	if err != nil {
		return toolError(fmt.Errorf("deps: %w", err)), struct{}{}, nil
	}

	edges := len(result.Calls) + len(result.CalledBy)
	tokens := edges * 20 // ~20 tokens saved per edge (summary vs full source)
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_deps", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		Calls              []service.DepsEdge `json:"calls"`
		CalledBy           []service.DepsEdge `json:"called_by"`
		HasDynamicEdges    bool               `json:"has_dynamic_edges"`
		PossibleUnresolved bool               `json:"possible_unresolved"`
		Metadata           Metadata           `json:"metadata"`
	}{
		Calls:              result.Calls,
		CalledBy:           result.CalledBy,
		HasDynamicEdges:    result.HasDynamicEdges,
		PossibleUnresolved: result.PossibleUnresolved,
		Metadata:           s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleOutline implements sigil_outline.
func (s *Server) handleOutline(ctx context.Context, _ *mcp.CallToolRequest, input OutlineInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, _, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	result, err := service.NewOutline(st).ForFile(ctx, input.File)
	if err != nil {
		return toolError(fmt.Errorf("outline: %w", err)), struct{}{}, nil
	}

	tokens := len(result.Symbols) * 10 // ~10 tokens saved per symbol (outline vs full source)
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_outline", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		File     string                  `json:"file"`
		Symbols  []service.OutlineEntry  `json:"symbols"`
		Metadata Metadata                `json:"metadata"`
	}{
		File:     result.File,
		Symbols:  result.Symbols,
		Metadata: s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleTree implements sigil_tree.
func (s *Server) handleTree(ctx context.Context, _ *mcp.CallToolRequest, input TreeInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, _, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	result, err := service.NewTree(st, repoRoot).Build(ctx, input.Scope, input.Depth, input.IncludeSymbolCounts)
	if err != nil {
		return toolError(fmt.Errorf("tree: %w", err)), struct{}{}, nil
	}

	tokens := 80 // static: ~80 tokens for a 50-file repo tree (design §8.1)
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_tree", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		Root     string              `json:"root"`
		Nodes    []service.TreeNode  `json:"nodes"`
		Metadata Metadata            `json:"metadata"`
	}{
		Root:     result.Root,
		Nodes:    result.Nodes,
		Metadata: s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleOverview implements sigil_overview.
func (s *Server) handleOverview(ctx context.Context, _ *mcp.CallToolRequest, input OverviewInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, meta, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	result, err := service.NewOverview(st, meta).Summary(ctx)
	if err != nil {
		return toolError(fmt.Errorf("overview: %w", err)), struct{}{}, nil
	}

	tokens := 60 // static: ~60 tokens (design §8.1)
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_overview", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		*service.OverviewResult
		Metadata Metadata `json:"metadata"`
	}{
		OverviewResult: result,
		Metadata:       s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleEnv implements sigil_env.
func (s *Server) handleEnv(ctx context.Context, _ *mcp.CallToolRequest, input EnvInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}

	result, err := service.NewEnvService(repoRoot).Inspect(ctx)
	if err != nil {
		return toolError(fmt.Errorf("env: %w", err)), struct{}{}, nil
	}

	tokens := 40 // static: env summary is small
	s.sess.record(tokens)
	// No store for env — savings not persisted per repo, but session is updated above.

	out := struct {
		*service.EnvResult
		Metadata Metadata `json:"metadata"`
	}{
		EnvResult: result,
		Metadata:  s.buildMetaNoStore(start, tokens),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleDiff implements sigil_diff.
func (s *Server) handleDiff(ctx context.Context, _ *mcp.CallToolRequest, input DiffInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, _, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	result, err := service.NewDiffer(st, repoRoot).Diff(ctx, input.Since)
	if err != nil {
		return toolError(fmt.Errorf("diff: %w", err)), struct{}{}, nil
	}

	changed := len(result.Added) + len(result.Modified) + len(result.Deleted)
	tokens := changed * 15 // ~15 tokens saved per changed symbol (summary vs full git diff)
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_diff", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		*service.DiffResult
		Metadata Metadata `json:"metadata"`
	}{
		DiffResult: result,
		Metadata:   s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}

// handleStatus implements sigil_status.
func (s *Server) handleStatus(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, struct{}, error) {
	start := time.Now()

	repoRoot, err := resolveRepoRoot(input.Path, s.serverCWD)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	st, meta, err := s.openRepoFn(repoRoot, s.cacheRoot)
	if err != nil {
		return toolError(err), struct{}{}, nil
	}
	defer st.Close()

	result, err := service.NewStatus(st, meta, repoRoot).Check(ctx, input.Verify)
	if err != nil {
		return toolError(fmt.Errorf("status: %w", err)), struct{}{}, nil
	}

	tokens := 30 // static: status is a small summary
	elapsed := time.Since(start)
	s.sess.record(tokens)
	s.appendSavings(ctx, st, "sigil_status", float64(elapsed.Milliseconds()), tokens)

	out := struct {
		*service.StatusResult
		Metadata Metadata `json:"metadata"`
	}{
		StatusResult: result,
		Metadata:     s.buildMeta(ctx, start, tokens, st),
	}
	return jsonResult(out), struct{}{}, nil
}
