package enrichment

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Detect auto-detects the best available enrichment provider.
// Priority: Anthropic → Google → OpenAI-compatible → Ollama → template.
// Returns a TemplateEnricher if no LLM provider is available.
func Detect(timeout time.Duration) Enricher {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		slog.Info("enrichment: using anthropic provider")
		return NewLLMEnricher(newAnthropicProvider(timeout))
	}
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		slog.Info("enrichment: using google provider")
		return NewLLMEnricher(newGoogleProvider(timeout))
	}
	if base := os.Getenv("OPENAI_API_BASE"); base != "" {
		slog.Info("enrichment: using openai-compatible provider", "base", base)
		return NewLLMEnricher(newOpenAIProvider(base, "openai", timeout))
	}
	// Try Ollama at :11434
	if isOllamaRunning() {
		slog.Info("enrichment: using ollama provider")
		return NewLLMEnricher(newOpenAIProvider("http://localhost:11434", "ollama", timeout))
	}
	slog.Info("enrichment: no llm provider detected, using template summaries")
	return NewTemplateEnricher()
}

// isOllamaRunning checks if Ollama is accessible at :11434.
func isOllamaRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:11434/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// IsAvailable returns true if any LLM provider is configured.
func IsAvailable() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != "" ||
		os.Getenv("GOOGLE_API_KEY") != "" ||
		os.Getenv("OPENAI_API_BASE") != "" ||
		isOllamaRunning()
}

// FormatProviderInfo returns a human-readable string for the detected provider.
func FormatProviderInfo(e Enricher) string {
	return fmt.Sprintf("enrichment provider: %s", e.Provider())
}
