package models

// Metadata is the envelope included in every MCP tool response.
type Metadata struct {
	TimingMs    float64         `json:"timing_ms"`
	TokensSaved int             `json:"tokens_saved"`
	Session     *SessionInfo    `json:"session,omitempty"`
	RepoTotal   *SavingsSummary `json:"repo_total,omitempty"`
}
