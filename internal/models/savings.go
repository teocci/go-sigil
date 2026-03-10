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

// LanguageStat holds per-language aggregate counts.
type LanguageStat struct {
	Language string `json:"language"`
	Files    int    `json:"files"`
	Symbols  int    `json:"symbols"`
}

// PackageStat holds per-package aggregate counts.
type PackageStat struct {
	Root    string `json:"root"`
	Symbols int    `json:"symbols"`
}

// SavingsSession holds per-session aggregate savings.
type SavingsSession struct {
	SessionID   string `json:"session_id"`
	TokensSaved int    `json:"tokens_saved"`
	CallCount   int    `json:"call_count"`
	FirstCall   string `json:"first_call"`
	LastCall    string `json:"last_call"`
}
