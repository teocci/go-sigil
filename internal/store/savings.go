package store

import (
	"context"
	"fmt"
	"time"

	"go-sigil/internal/models"
)

// AppendSavings records a single token savings measurement.
func (s *SQLiteStore) AppendSavings(ctx context.Context, entry models.SavingsEntry) error {
	callAt := entry.CallAt
	if callAt == "" {
		callAt = time.Now().UTC().Format(time.RFC3339)
	}

	const q = `
INSERT INTO savings_log (session_id, tool_name, call_at, timing_ms, tokens_saved)
VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, q,
		entry.SessionID,
		entry.ToolName,
		callAt,
		entry.TimingMs,
		entry.TokensSaved,
	)
	if err != nil {
		return fmt.Errorf("append savings: %w", err)
	}
	return nil
}

// GetSessionSavings returns aggregated savings for a specific session.
func (s *SQLiteStore) GetSessionSavings(ctx context.Context, sessionID string) (models.SavingsSummary, error) {
	const q = `
SELECT COALESCE(SUM(tokens_saved), 0), COUNT(*)
FROM savings_log
WHERE session_id = ?`

	var summary models.SavingsSummary
	err := s.db.QueryRowContext(ctx, q, sessionID).Scan(
		&summary.TokensSaved,
		&summary.CallCount,
	)
	if err != nil {
		return models.SavingsSummary{}, fmt.Errorf("get session savings: %w", err)
	}
	return summary, nil
}

// GetRepoSavings returns aggregated savings across all sessions.
func (s *SQLiteStore) GetRepoSavings(ctx context.Context) (models.SavingsSummary, error) {
	const q = `SELECT COALESCE(SUM(tokens_saved), 0), COUNT(*) FROM savings_log`

	var summary models.SavingsSummary
	err := s.db.QueryRowContext(ctx, q).Scan(
		&summary.TokensSaved,
		&summary.CallCount,
	)
	if err != nil {
		return models.SavingsSummary{}, fmt.Errorf("get repo savings: %w", err)
	}
	return summary, nil
}
