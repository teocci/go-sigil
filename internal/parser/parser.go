// Package parser defines the Parser interface and a language registry for
// extracting symbols and call-graph edges from source files using tree-sitter.
package parser

import (
	"fmt"
	"sync"
)

// Parser extracts symbols and call edges from source file content.
// Each language provides one implementation. Implementations must be
// safe for concurrent use (one Parser instance reused across goroutines).
type Parser interface {
	// Language returns the canonical language name (e.g. "go", "typescript").
	Language() string

	// Parse extracts symbols and edges from src.
	// filePath is the repo-relative path (used to build qualified names).
	// pkgPath is the package directory relative to the repo root.
	Parse(filePath, pkgPath string, src []byte) (*ParseResult, error)
}

// Registry maps language names to their Parser implementations.
type Registry struct {
	mu      sync.RWMutex
	parsers map[string]Parser
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{parsers: make(map[string]Parser)}
}

// Register adds a parser for the given language. Panics on duplicate.
func (r *Registry) Register(p Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	lang := p.Language()
	if _, exists := r.parsers[lang]; exists {
		panic(fmt.Sprintf("parser: language %q already registered", lang))
	}
	r.parsers[lang] = p
}

// Get returns the Parser for a language, or (nil, false) if not registered.
func (r *Registry) Get(language string) (Parser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.parsers[language]
	return p, ok
}

// Languages returns all registered language names.
func (r *Registry) Languages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	langs := make([]string, 0, len(r.parsers))
	for k := range r.parsers {
		langs = append(langs, k)
	}
	return langs
}
