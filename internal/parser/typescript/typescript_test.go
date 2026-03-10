package typescript_test

import (
	"os"
	"strings"
	"testing"

	"go-sigil/internal/models"
	tsparser "go-sigil/internal/parser/typescript"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}

func TestTSParser_Language(t *testing.T) {
	p := tsparser.New()
	if got := p.Language(); got != "typescript" {
		t.Errorf("Language() = %q, want %q", got, "typescript")
	}
}

func TestTSParser_Parse_Symbols(t *testing.T) {
	p := tsparser.New()
	src := loadFixture(t, "sample.ts")

	result, err := p.Parse("src/sample.ts", "src", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	byName := make(map[string]string, len(result.Symbols))
	for _, s := range result.Symbols {
		byName[s.Name] = s.Kind
	}

	tests := []struct {
		name string
		kind string
	}{
		{"Animal", "class"},
		{"Greeter", "interface"},
		{"Point", "type"},
		{"greet", "function"},
		{"helper", "function"},
		{"speak", "method"},
		{"constructor", "method"},
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

func TestTSParser_Parse_SymbolIDs(t *testing.T) {
	p := tsparser.New()
	src := loadFixture(t, "sample.ts")

	result, err := p.Parse("src/sample.ts", "src", src)
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

func TestTSParser_Parse_QualifiedNames(t *testing.T) {
	p := tsparser.New()
	src := loadFixture(t, "sample.ts")

	result, err := p.Parse("src/sample.ts", "mypkg", src)
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
		{"greet", "mypkg.greet"},
		{"Animal", "mypkg.Animal"},
		{"speak", "mypkg.Animal.speak"},
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

func TestTSParser_Parse_Summaries(t *testing.T) {
	p := tsparser.New()
	src := loadFixture(t, "sample.ts")

	result, err := p.Parse("src/sample.ts", "src", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	byName := make(map[string]string)
	for _, s := range result.Symbols {
		byName[s.Name] = s.Summary
	}

	tests := []struct {
		name        string
		wantSummary string
	}{
		{"greet", "greet creates a greeting string for the given name."},
		{"speak", "speak outputs the animal's sound."},
		{"Animal", "Animal is the base class for all animals."},
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

func TestTSParser_Parse_Edges(t *testing.T) {
	p := tsparser.New()
	src := loadFixture(t, "sample.ts")

	result, err := p.Parse("src/sample.ts", "src", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

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

func TestTSParser_Parse_EmptyFile(t *testing.T) {
	p := tsparser.New()
	result, err := p.Parse("empty.ts", "pkg", []byte(""))
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
