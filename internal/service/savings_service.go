package service

import (
	"context"
	"fmt"

	"go-sigil/internal/models"
	"go-sigil/internal/store"
)

// SavingsService provides token savings analytics.
type SavingsService struct {
	st store.SymbolStore
}

// NewSavings creates a SavingsService.
func NewSavings(st store.SymbolStore) *SavingsService {
	return &SavingsService{st: st}
}

// RepoSavings returns aggregate savings for this repo.
func (s *SavingsService) RepoSavings(ctx context.Context) (models.SavingsSummary, error) {
	return s.st.GetRepoSavings(ctx)
}

// SessionSavings returns savings for a specific session.
func (s *SavingsService) SessionSavings(ctx context.Context, sessionID string) (models.SavingsSummary, error) {
	return s.st.GetSessionSavings(ctx, sessionID)
}

// ListSessions returns all sessions ordered by recency.
func (s *SavingsService) ListSessions(ctx context.Context) ([]models.SavingsSession, error) {
	sessions, err := s.st.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return sessions, nil
}

// TopSessions returns the top n sessions by tokens saved.
func (s *SavingsService) TopSessions(ctx context.Context, n int) ([]models.SavingsSession, error) {
	if n <= 0 {
		n = 10
	}
	sessions, err := s.st.GetTopSessions(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("top sessions: %w", err)
	}
	return sessions, nil
}
