package enrichment

import (
	"context"
	"fmt"
	"log/slog"

	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// provider is an internal LLM provider.
type provider interface {
	// Complete sends a prompt to the LLM and returns the response.
	Complete(ctx context.Context, prompt string) (string, error)
	// Name returns the provider name.
	Name() string
}

// LLMEnricher enriches symbols using an LLM provider with template fallback.
type LLMEnricher struct {
	p provider
}

// NewLLMEnricher creates a LLMEnricher using the given provider.
func NewLLMEnricher(p provider) *LLMEnricher {
	return &LLMEnricher{p: p}
}

// Provider returns the provider name.
func (e *LLMEnricher) Provider() string {
	return e.p.Name()
}

// Enrich generates a summary for sym using the LLM.
// If the LLM fails, falls back to the template summary.
func (e *LLMEnricher) Enrich(ctx context.Context, sym *models.Symbol, src []byte) error {
	if sym.Summary != "" {
		// Already has a docstring-derived summary — don't overwrite
		return nil
	}

	prompt := buildPrompt(sym, src)
	summary, err := e.p.Complete(ctx, prompt)
	if err != nil {
		slog.Debug("enrichment provider failed, using template",
			"provider", e.p.Name(), "symbol", sym.QualifiedName, "error", err)
		sym.Summary = parser.TemplateSummary(sym.Kind, sym.Name, sym.Signature, "")
		return nil
	}
	sym.Summary = truncateSummary(summary)
	return nil
}

// buildPrompt creates the enrichment prompt for a symbol.
func buildPrompt(sym *models.Symbol, src []byte) string {
	sourceStr := ""
	if sym.ByteStart != nil && sym.ByteEnd != nil {
		s, e := *sym.ByteStart, *sym.ByteEnd
		if s >= 0 && e <= len(src) && s <= e {
			sourceStr = string(src[s:e])
		}
	}
	return fmt.Sprintf(
		"Write a one-sentence description (max 120 chars) for this %s named %q.\n"+
			"Signature: %s\n"+
			"Source:\n%s\n"+
			"Return ONLY the description sentence, nothing else.",
		sym.Kind, sym.Name, sym.Signature, sourceStr,
	)
}

// truncateSummary ensures the summary is at most 200 chars and a single line.
func truncateSummary(s string) string {
	// Take first line only
	for i, c := range s {
		if c == '\n' {
			s = s[:i]
			break
		}
	}
	if len(s) > 200 {
		s = s[:197] + "..."
	}
	return s
}
