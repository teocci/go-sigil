package service

import (
	"context"
	"fmt"

	"go-sigil/internal/store"
)

// DepsEdge is an enriched call graph edge.
type DepsEdge struct {
	ID            string `json:"id,omitempty"`
	Confidence    string `json:"confidence"`
	QualifiedName string `json:"qualified_name,omitempty"`
	Summary       string `json:"summary,omitempty"`
	RawExpression string `json:"raw_expression,omitempty"`
}

// DepsResult holds call graph traversal results.
type DepsResult struct {
	Calls              []DepsEdge `json:"calls"`
	CalledBy           []DepsEdge `json:"called_by"`
	HasDynamicEdges    bool       `json:"has_dynamic_edges"`
	PossibleUnresolved bool       `json:"possible_unresolved"`
}

// Deps traverses the call graph.
type Deps struct {
	st store.SymbolStore
}

// NewDeps creates a Deps service.
func NewDeps(st store.SymbolStore) *Deps {
	return &Deps{st: st}
}

// Trace returns calls and callers for a symbol.
// direction: "calls", "callers", or "both"
func (d *Deps) Trace(ctx context.Context, symbolID, direction string, depth int) (*DepsResult, error) {
	if depth <= 0 {
		depth = 1
	}

	sym, err := d.st.GetSymbolByID(ctx, symbolID)
	if err != nil {
		return nil, fmt.Errorf("get symbol: %w", err)
	}
	if sym == nil {
		return nil, fmt.Errorf("symbol %q not found", symbolID)
	}

	result := &DepsResult{
		PossibleUnresolved: sym.PossibleUnresolved,
	}

	if direction == "calls" || direction == "both" {
		calls, err := d.st.GetCalls(ctx, symbolID, depth)
		if err != nil {
			return nil, fmt.Errorf("get calls: %w", err)
		}
		for _, edge := range calls {
			de := DepsEdge{
				ID:            edge.CalleeID,
				Confidence:    edge.Confidence,
				RawExpression: edge.RawExpression,
			}
			if edge.Confidence == "dynamic" {
				result.HasDynamicEdges = true
			}
			if edge.CalleeID != "" {
				callee, _ := d.st.GetSymbolByID(ctx, edge.CalleeID)
				if callee != nil {
					de.QualifiedName = callee.QualifiedName
					de.Summary = callee.Summary
				}
			}
			result.Calls = append(result.Calls, de)
		}
	}

	if direction == "callers" || direction == "both" {
		callers, err := d.st.GetCalledBy(ctx, symbolID, depth)
		if err != nil {
			return nil, fmt.Errorf("get callers: %w", err)
		}
		for _, edge := range callers {
			de := DepsEdge{
				ID:         edge.CallerID,
				Confidence: edge.Confidence,
			}
			caller, _ := d.st.GetSymbolByID(ctx, edge.CallerID)
			if caller != nil {
				de.QualifiedName = caller.QualifiedName
				de.Summary = caller.Summary
			}
			result.CalledBy = append(result.CalledBy, de)
		}
	}

	return result, nil
}
