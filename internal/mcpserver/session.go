package mcpserver

import (
	"sync"
	"time"

	"go-sigil/internal/service"
)

// mcpSession tracks per-process MCP session state.
// One session lives for the lifetime of the sigil-mcp process.
type mcpSession struct {
	mu          sync.Mutex
	id          string
	startedAt   time.Time
	callCount   int
	tokensSaved int
}

func newMCPSession() *mcpSession {
	return &mcpSession{
		id:        service.NewSessionID(),
		startedAt: time.Now(),
	}
}

func (s *mcpSession) record(tokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount++
	s.tokensSaved += tokens
}

// snapshot returns a consistent view of session state.
func (s *mcpSession) snapshot() (id string, callCount, tokensSaved int, startedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.id, s.callCount, s.tokensSaved, s.startedAt
}
