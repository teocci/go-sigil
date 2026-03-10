package mcpserver

import "go-sigil/internal/models"

// Metadata is the standard envelope included in every MCP tool response.
type Metadata struct {
	TimingMs    float64            `json:"timing_ms"`
	TokensSaved int                `json:"tokens_saved"`
	Session     models.SessionInfo `json:"session"`
	RepoTotal   models.SavingsSummary `json:"repo_total"`
}

// SearchInput is the input for sigil_search.
type SearchInput struct {
	Query    string `json:"query" jsonschema:"symbol name, pattern, or full-text search term"`
	Kind     string `json:"kind,omitempty" jsonschema:"filter by symbol kind: function|method|class|type|interface|const|var"`
	Language string `json:"language,omitempty" jsonschema:"filter by language: go|typescript|javascript|python"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max results, default 10"`
	Path     string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// GetInput is the input for sigil_get.
type GetInput struct {
	IDs                 []string `json:"ids,omitempty" jsonschema:"symbol IDs to retrieve with source"`
	Files               []string `json:"files,omitempty" jsonschema:"file paths to return as raw content"`
	IncludeContextLines int      `json:"include_context_lines,omitempty" jsonschema:"context lines around each symbol body, 0-50"`
	Path                string   `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// DepsInput is the input for sigil_deps.
type DepsInput struct {
	ID        string `json:"id" jsonschema:"symbol ID to trace in the call graph"`
	Direction string `json:"direction,omitempty" jsonschema:"calls|callers|both, default both"`
	Depth     int    `json:"depth,omitempty" jsonschema:"traversal depth, default 1"`
	Path      string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// OutlineInput is the input for sigil_outline.
type OutlineInput struct {
	File string `json:"file" jsonschema:"file path relative to the repo root"`
	Path string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// TreeInput is the input for sigil_tree.
type TreeInput struct {
	Scope               string `json:"scope,omitempty" jsonschema:"directory scope to restrict the tree, default repo root"`
	Depth               int    `json:"depth,omitempty" jsonschema:"max depth levels, default 3"`
	IncludeSymbolCounts bool   `json:"include_symbol_counts,omitempty" jsonschema:"annotate files and directories with symbol counts"`
	Path                string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// OverviewInput is the input for sigil_overview.
type OverviewInput struct {
	Path string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// EnvInput is the input for sigil_env.
type EnvInput struct {
	Path string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// DiffInput is the input for sigil_diff.
type DiffInput struct {
	Since string `json:"since" jsonschema:"git ref to diff from, e.g. HEAD~1, main, or a commit SHA"`
	Path  string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}

// StatusInput is the input for sigil_status.
type StatusInput struct {
	Verify bool   `json:"verify,omitempty" jsonschema:"re-read symbol sources to verify content hashes, slower"`
	Path   string `json:"path,omitempty" jsonschema:"file or directory path used to select the repo (optional)"`
}
