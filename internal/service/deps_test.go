package service

import (
	"context"
	"errors"
	"testing"

	"go-sigil/internal/models"
)

func TestDeps_Trace(t *testing.T) {
	ctx := context.Background()

	rootSym := &models.Symbol{ID: "root0001", Name: "Handler", QualifiedName: "pkg.Handler"}
	calleeSym := &models.Symbol{ID: "callee01", Name: "Validate", QualifiedName: "pkg.Validate", Summary: "validates input"}
	callerSym := &models.Symbol{ID: "caller01", Name: "Router", QualifiedName: "pkg.Router"}

	callEdge := models.CallEdge{CallerID: "root0001", CalleeID: "callee01", Confidence: "static"}
	callerEdge := models.CallEdge{CallerID: "caller01", CalleeID: "root0001", Confidence: "static"}

	byID := map[string]*models.Symbol{
		"root0001": rootSym,
		"callee01": calleeSym,
		"caller01": callerSym,
	}

	tests := []struct {
		name          string
		symbolID      string
		direction     string
		depth         int
		wantCallsLen  int
		wantCallersLen int
		wantErr       bool
	}{
		{
			name:         "calls direction",
			symbolID:     "root0001",
			direction:    "calls",
			depth:        1,
			wantCallsLen: 1,
		},
		{
			name:           "callers direction",
			symbolID:       "root0001",
			direction:      "callers",
			depth:          1,
			wantCallersLen: 1,
		},
		{
			name:           "both direction",
			symbolID:       "root0001",
			direction:      "both",
			depth:          1,
			wantCallsLen:   1,
			wantCallersLen: 1,
		},
		{
			name:      "depth zero defaults to 1",
			symbolID:  "root0001",
			direction: "calls",
			depth:     0,
			wantCallsLen: 1,
		},
		{
			name:      "symbol not found",
			symbolID:  "missing0",
			direction: "calls",
			depth:     1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &mockStore{
				getSymbolByIDFn: func(_ context.Context, id string) (*models.Symbol, error) {
					if id == "missing0" {
						return nil, nil
					}
					if id == "" {
						return nil, errors.New("empty id")
					}
					return byID[id], nil
				},
				getCallsFn: func(_ context.Context, id string, _ int) ([]models.CallEdge, error) {
					if id == "root0001" {
						return []models.CallEdge{callEdge}, nil
					}
					return nil, nil
				},
				getCalledByFn: func(_ context.Context, id string, _ int) ([]models.CallEdge, error) {
					if id == "root0001" {
						return []models.CallEdge{callerEdge}, nil
					}
					return nil, nil
				},
			}

			d := NewDeps(st)
			result, err := d.Trace(ctx, tt.symbolID, tt.direction, tt.depth)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Trace() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(result.Calls) != tt.wantCallsLen {
				t.Errorf("len(Calls) = %d, want %d", len(result.Calls), tt.wantCallsLen)
			}
			if len(result.CalledBy) != tt.wantCallersLen {
				t.Errorf("len(CalledBy) = %d, want %d", len(result.CalledBy), tt.wantCallersLen)
			}
			// verify callee name is enriched from store lookup
			if tt.wantCallsLen > 0 {
				if result.Calls[0].QualifiedName != calleeSym.QualifiedName {
					t.Errorf("call QualifiedName = %q, want %q", result.Calls[0].QualifiedName, calleeSym.QualifiedName)
				}
			}
		})
	}
}

func TestDeps_Trace_DynamicEdge(t *testing.T) {
	ctx := context.Background()
	sym := &models.Symbol{ID: "sym00001", Name: "Handler"}
	dynEdge := models.CallEdge{CallerID: "sym00001", CalleeID: "", Confidence: "dynamic", RawExpression: "obj.method()"}

	st := &mockStore{
		getSymbolByIDFn: func(_ context.Context, _ string) (*models.Symbol, error) { return sym, nil },
		getCallsFn: func(_ context.Context, _ string, _ int) ([]models.CallEdge, error) {
			return []models.CallEdge{dynEdge}, nil
		},
	}

	d := NewDeps(st)
	result, err := d.Trace(ctx, "sym00001", "calls", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasDynamicEdges {
		t.Error("expected HasDynamicEdges=true for dynamic confidence edge")
	}
	if len(result.Calls) != 1 {
		t.Errorf("expected 1 call edge, got %d", len(result.Calls))
	}
	if result.Calls[0].RawExpression != "obj.method()" {
		t.Errorf("RawExpression = %q, want %q", result.Calls[0].RawExpression, "obj.method()")
	}
}
