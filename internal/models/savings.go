package models

// SavingsEntry represents a single token savings measurement.
type SavingsEntry struct {
	SessionID   string  `json:"session_id"`
	ToolName    string  `json:"tool_name"`
	CallAt      string  `json:"call_at"`
	TimingMs    float64 `json:"timing_ms"`
	TokensSaved int     `json:"tokens_saved"`
}

// SavingsSummary aggregates token savings.
type SavingsSummary struct {
	TokensSaved int `json:"tokens_saved"`
	CallCount   int `json:"call_count"`
}

// SessionInfo holds session-level savings context included in metadata envelopes.
type SessionInfo struct {
	ID          string `json:"id"`
	TokensSaved int    `json:"tokens_saved"`
	CallCount   int    `json:"call_count"`
	StartedAt   string `json:"started_at"`
}
