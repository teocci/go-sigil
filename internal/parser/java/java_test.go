package java_test

import (
	"os"
	"strings"
	"testing"

	"go-sigil/internal/models"
	javaparser "go-sigil/internal/parser/java"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}

func joinNames(syms []*models.Symbol) string {
	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

func TestJavaParser_Language(t *testing.T) {
	p := javaparser.New()
	if got := p.Language(); got != "java" {
		t.Errorf("Language() = %q, want %q", got, "java")
	}
}

func TestJavaParser_Parse_Symbols(t *testing.T) {
	p := javaparser.New()
	src := loadFixture(t, "Sample.java")

	result, err := p.Parse("src/main/Sample.java", "src/main", src)
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
		{"Sample", "type"},
		{"Processor", "interface"},
		{"Status", "type"},
		{"getName", "method"},
		{"run", "method"},
		{"helper", "method"},
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

func TestJavaParser_Parse_SymbolIDs(t *testing.T) {
	p := javaparser.New()
	src := loadFixture(t, "Sample.java")

	result, err := p.Parse("src/main/Sample.java", "src/main", src)
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

func TestJavaParser_Parse_QualifiedNames(t *testing.T) {
	p := javaparser.New()
	src := loadFixture(t, "Sample.java")

	result, err := p.Parse("src/main/Sample.java", "mypkg", src)
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
		{"Sample", "mypkg.Sample"},
		{"Processor", "mypkg.Processor"},
		{"getName", "mypkg.Sample.getName"},
		{"run", "mypkg.Sample.run"},
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

func TestJavaParser_Parse_Edges(t *testing.T) {
	p := javaparser.New()
	src := loadFixture(t, "Sample.java")

	result, err := p.Parse("src/main/Sample.java", "src/main", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// run() calls System.out.println — should produce at least one edge.
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

func TestJavaParser_Parse_EmptyFile(t *testing.T) {
	p := javaparser.New()
	result, err := p.Parse("Empty.java", "pkg", []byte(""))
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

func TestJavaParser_Parse_Signatures(t *testing.T) {
	p := javaparser.New()
	src := loadFixture(t, "Sample.java")

	result, err := p.Parse("src/main/Sample.java", "pkg", src)
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
		{"Sample", "type Sample class"},
		{"Processor", "type Processor interface"},
		{"Status", "type Status enum"},
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
