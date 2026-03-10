// Package enrichment generates LLM-based summaries for indexed symbols.
// Provider priority: Anthropic → Google → OpenAI-compatible → Ollama → template.
package enrichment

import (
	"context"

	"go-sigil/internal/models"
)

// Enricher generates summaries for symbols.
// Implementations must be safe for concurrent use.
type Enricher interface {
	// Enrich generates a summary for sym, updating sym.Summary in place.
	// Returns nil on success. Falls back to template if provider unavailable.
	Enrich(ctx context.Context, sym *models.Symbol, src []byte) error
	// Provider returns the human-readable provider name.
	Provider() string
}
