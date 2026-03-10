package service

import (
	"context"
	"fmt"

	"go-sigil/internal/store"
)

// OutlineEntry is a symbol in the file outline.
type OutlineEntry struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Signature string `json:"signature,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Depth     int    `json:"depth"`
	ParentID  string `json:"parent_id,omitempty"`
}

// OutlineResult holds the hierarchical symbol outline of a file.
type OutlineResult struct {
	File    string         `json:"file"`
	Symbols []OutlineEntry `json:"symbols"`
}

// Outline produces hierarchical symbol outlines for files.
type Outline struct {
	st store.SymbolStore
}

// NewOutline creates an Outline service.
func NewOutline(st store.SymbolStore) *Outline {
	return &Outline{st: st}
}

// ForFile returns all symbols in the given file, ordered by line.
func (o *Outline) ForFile(ctx context.Context, file string) (*OutlineResult, error) {
	syms, err := o.st.GetSymbolsByFile(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("get symbols for file: %w", err)
	}
	entries := make([]OutlineEntry, 0, len(syms))
	for _, sym := range syms {
		entries = append(entries, OutlineEntry{
			ID:        sym.ID,
			Kind:      sym.Kind,
			Name:      sym.Name,
			Signature: sym.Signature,
			Summary:   sym.Summary,
			Depth:     sym.Depth,
			ParentID:  sym.ParentID,
		})
	}
	return &OutlineResult{File: file, Symbols: entries}, nil
}
