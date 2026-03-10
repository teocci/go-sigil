package parser

import "go-sigil/internal/models"

// ParseResult holds all symbols and call edges extracted from a single file.
type ParseResult struct {
	Symbols []*models.Symbol
	Edges   []*models.CallEdge
}
