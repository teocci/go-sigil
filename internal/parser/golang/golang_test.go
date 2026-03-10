package golang_test

import (
	"os"
	"strings"
	"testing"

	"go-sigil/internal/models"
	golangparser "go-sigil/internal/parser/golang"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}

func TestGoParser_Language(t *testing.T) {
	p := golangparser.New()
	if got := p.Language(); got != "go" {
		t.Errorf("Language() = %q, want %q", got, "go")
	}
}

func TestGoParser_Parse_Symbols(t *testing.T) {
	p := golangparser.New()
	src := loadFixture(t, "sample.go")

	result, err := p.Parse("internal/parser/golang/testdata/sample.go", "internal/parser/golang/testdata", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Build name→kind index for assertions.
	byName := make(map[string]string, len(result.Symbols))
	for _, s := range result.Symbols {
		byName[s.Name] = s.Kind
	}

	tests := []struct {
		name string
		kind string
	}{
		{"MaxRetries", "const"},
		{"DefaultTimeout", "var"},
		{"Processor", "interface"},
		{"Worker", "type"},
		{"NewWorker", "function"},
		{"Run", "method"},
		{"helper", "function"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, ok := byName[tt.name]
			if !ok {
				t.Errorf("symbol %q not found; got: %s", tt.name, joinNames(result.Symbols))
				return
			}
			if kind != tt.kind {
				t.Errorf("symbol %q: kind = %q, want %q", tt.name, kind, tt.kind)
			}
		})
	}
}

func TestGoParser_Parse_SymbolIDs(t *testing.T) {
	p := golangparser.New()
	src := loadFixture(t, "sample.go")

	result, err := p.Parse("internal/parser/golang/testdata/sample.go", "pkg", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for _, s := range result.Symbols {
		if len(s.ID) != 8 {
			t.Errorf("symbol %q: ID len = %d, want 8 (got %q)", s.Name, len(s.ID), s.ID)
		}
		if s.QualifiedName == "" {
			t.Errorf("symbol %q: empty QualifiedName", s.Name)
		}
		if s.ByteStart == nil || s.ByteEnd == nil {
			t.Errorf("symbol %q: nil byte range", s.Name)
		} else if *s.ByteStart >= *s.ByteEnd {
			t.Errorf("symbol %q: ByteStart(%d) >= ByteEnd(%d)", s.Name, *s.ByteStart, *s.ByteEnd)
		}
	}
}

func TestGoParser_Parse_Signatures(t *testing.T) {
	p := golangparser.New()
	src := loadFixture(t, "sample.go")

	result, err := p.Parse("internal/parser/golang/testdata/sample.go", "pkg", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	byName := make(map[string]string, len(result.Symbols))
	for _, s := range result.Symbols {
		byName[s.Name] = s.Signature
	}

	tests := []struct {
		name    string
		wantSig string
	}{
		{"NewWorker", "func NewWorker(name string)"},
		{"Run", "func (Worker) Run()"},
		{"helper", "func helper(x, y int)"},
		{"Worker", "type Worker struct"},
		{"Processor", "type Processor interface"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, ok := byName[tt.name]
			if !ok {
				t.Errorf("symbol %q not found", tt.name)
				return
			}
			if sig != tt.wantSig {
				t.Errorf("symbol %q: sig = %q, want %q", tt.name, sig, tt.wantSig)
			}
		})
	}
}

func TestGoParser_Parse_Summaries(t *testing.T) {
	p := golangparser.New()
	src := loadFixture(t, "sample.go")

	result, err := p.Parse("internal/parser/golang/testdata/sample.go", "pkg", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	byName := make(map[string]string, len(result.Symbols))
	for _, s := range result.Symbols {
		byName[s.Name] = s.Summary
	}

	tests := []struct {
		name        string
		wantSummary string
	}{
		{"MaxRetries", "MaxRetries is the maximum number of retry attempts."},
		{"NewWorker", "NewWorker creates a Worker with the given name."},
		{"Run", "Run starts the worker processing loop."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := byName[tt.name]
			if !ok {
				t.Errorf("symbol %q not found", tt.name)
				return
			}
			if got != tt.wantSummary {
				t.Errorf("symbol %q: summary = %q, want %q", tt.name, got, tt.wantSummary)
			}
		})
	}
}

func TestGoParser_Parse_Edges(t *testing.T) {
	p := golangparser.New()
	src := loadFixture(t, "sample.go")

	result, err := p.Parse("internal/parser/golang/testdata/sample.go", "pkg", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Run() calls fmt.Println — should produce at least one edge.
	if len(result.Edges) == 0 {
		t.Error("expected at least one call edge, got none")
	}

	for _, e := range result.Edges {
		if e.CallerID == "" {
			t.Errorf("edge with empty CallerID: %+v", e)
		}
		if e.Confidence == "" {
			t.Errorf("edge with empty Confidence: %+v", e)
		}
	}
}

func TestGoParser_Parse_QualifiedNames(t *testing.T) {
	p := golangparser.New()
	src := loadFixture(t, "sample.go")

	result, err := p.Parse("f.go", "mypkg", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	byName := make(map[string]string)
	for _, s := range result.Symbols {
		byName[s.Name] = s.QualifiedName
	}

	tests := []struct {
		name   string
		wantQN string
	}{
		{"NewWorker", "mypkg.NewWorker"},
		{"Run", "mypkg.Worker.Run"},
		{"helper", "mypkg.helper"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := byName[tt.name]
			if !ok {
				t.Errorf("symbol %q not found", tt.name)
				return
			}
			if got != tt.wantQN {
				t.Errorf("symbol %q: qualified_name = %q, want %q", tt.name, got, tt.wantQN)
			}
		})
	}
}

func TestGoParser_Parse_EmptyFile(t *testing.T) {
	p := golangparser.New()
	result, err := p.Parse("empty.go", "pkg", []byte("package empty\n"))
	if err != nil {
		t.Fatalf("Parse empty file: %v", err)
	}
	if len(result.Symbols) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(result.Symbols))
	}
	if len(result.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(result.Edges))
	}
}

func joinNames(syms []*models.Symbol) string {
	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}
