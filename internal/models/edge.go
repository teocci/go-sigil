package models

// CallEdge represents a caller-callee relationship in the call graph.
type CallEdge struct {
	CallerID      string `json:"caller_id"`
	CalleeID      string `json:"callee_id,omitempty"`      // nil for dynamic/unresolvable edges
	RawExpression string `json:"raw_expression,omitempty"` // populated for dynamic edges
	Confidence    string `json:"confidence"`                // static|inferred|dynamic
}
