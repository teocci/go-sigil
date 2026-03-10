package enrichment

import (
	"context"

	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// TemplateEnricher generates summaries using the template fallback only.
// Used when no LLM provider is available.
type TemplateEnricher struct{}

// NewTemplateEnricher creates a TemplateEnricher.
func NewTemplateEnricher() *TemplateEnricher {
	return &TemplateEnricher{}
}

// Provider returns the provider name.
func (e *TemplateEnricher) Provider() string { return "template" }

// Enrich sets a template-based summary if the symbol has none.
func (e *TemplateEnricher) Enrich(_ context.Context, sym *models.Symbol, _ []byte) error {
	if sym.Summary == "" {
		sym.Summary = parser.TemplateSummary(sym.Kind, sym.Name, sym.Signature, "")
	}
	return nil
}
