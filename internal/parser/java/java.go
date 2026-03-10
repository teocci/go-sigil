// Package java implements the Sigil parser for the Java language using tree-sitter.
package java

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	javasitter "github.com/smacker/go-tree-sitter/java"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// JavaParser extracts symbols and call edges from Java source files.
// A single instance is safe for concurrent use.
type JavaParser struct {
	lang *sitter.Language
}

// New returns a JavaParser ready for use.
func New() *JavaParser {
	return &JavaParser{lang: javasitter.GetLanguage()}
}

// Language returns the canonical language name.
func (j *JavaParser) Language() string { return "java" }

// Parse extracts symbols and call edges from Java source.
// filePath is the repo-relative file path; pkgPath is the directory portion.
func (j *JavaParser) Parse(filePath, pkgPath string, src []byte) (*parser.ParseResult, error) {
	p := sitter.NewParser()
	p.SetLanguage(j.lang)
	defer p.Close()

	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse %s: %w", filePath, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	result := &parser.ParseResult{}

	if err := j.extractClasses(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := j.extractInterfaces(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := j.extractEnums(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := j.extractMethods(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	j.extractEdges(root, src, result)

	return result, nil
}

// --- symbol extraction ---

func (j *JavaParser) extractClasses(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryClasses, j.lang)
	if err != nil {
		return fmt.Errorf("classes query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		caps := captureMap(m.Captures, q, src)
		name := caps["class.name"]
		defNode := captureNode(m.Captures, q, "class.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := j.buildSymbol(constants.KindType, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "class")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (j *JavaParser) extractInterfaces(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryInterfaces, j.lang)
	if err != nil {
		return fmt.Errorf("interfaces query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		caps := captureMap(m.Captures, q, src)
		name := caps["interface.name"]
		defNode := captureNode(m.Captures, q, "interface.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := j.buildSymbol(constants.KindInterface, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "interface")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (j *JavaParser) extractEnums(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryEnums, j.lang)
	if err != nil {
		return fmt.Errorf("enums query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		caps := captureMap(m.Captures, q, src)
		name := caps["enum.name"]
		defNode := captureNode(m.Captures, q, "enum.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := j.buildSymbol(constants.KindType, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "enum")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (j *JavaParser) extractMethods(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryMethods, j.lang)
	if err != nil {
		return fmt.Errorf("methods query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		caps := captureMap(m.Captures, q, src)
		name := caps["method.name"]
		defNode := captureNode(m.Captures, q, "method.def")
		if name == "" || defNode == nil {
			continue
		}
		receiver := enclosingClassName(defNode, src)
		params := stripOuterParens(caps["method.params"])
		result.Symbols = append(result.Symbols, j.buildSymbol(
			constants.KindMethod, name, receiver, pkgPath, params, "", defNode, src, filePath,
		))
	}
	return nil
}

// --- call edge extraction ---

func (j *JavaParser) extractEdges(root *sitter.Node, src []byte, result *parser.ParseResult) {
	ranges := make([]symRange, 0, len(result.Symbols))
	for _, s := range result.Symbols {
		if s.ByteStart != nil && s.ByteEnd != nil {
			ranges = append(ranges, symRange{
				start: uint32(*s.ByteStart),
				end:   uint32(*s.ByteEnd),
				id:    s.ID,
			})
		}
	}

	runCallQuery := func(q *sitter.Query) {
		qc := sitter.NewQueryCursor()
		defer qc.Close()
		qc.Exec(q, root)

		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}
			caps := captureMap(m.Captures, q, src)
			if caps["call.func"] == "" {
				continue
			}
			exprNode := captureNode(m.Captures, q, "call.expr")
			if exprNode == nil {
				continue
			}

			callerID := enclosingSymbol(exprNode.StartByte(), ranges)
			if callerID == "" {
				continue
			}

			result.Edges = append(result.Edges, &models.CallEdge{
				CallerID:      callerID,
				CalleeID:      "",
				RawExpression: caps["call.expr"],
				Confidence:    string(constants.ConfidenceDynamic),
			})
		}
	}

	if q, err := sitter.NewQuery(queryDirectCalls, j.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
	if q, err := sitter.NewQuery(queryScopedCalls, j.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
}

// symRange associates a byte range with a symbol ID.
type symRange struct {
	start, end uint32
	id         string
}

// enclosingSymbol returns the ID of the innermost symbol range containing byteOffset.
func enclosingSymbol(byteOffset uint32, ranges []symRange) string {
	best := ""
	bestSize := uint32(0)
	for _, r := range ranges {
		if byteOffset >= r.start && byteOffset < r.end {
			size := r.end - r.start
			if best == "" || size < bestSize {
				best = r.id
				bestSize = size
			}
		}
	}
	return best
}

// enclosingClassName walks up the AST from defNode to find the nearest
// class_declaration and returns its name.
func enclosingClassName(defNode *sitter.Node, src []byte) string {
	n := defNode.Parent()
	for n != nil {
		if n.Type() == "class_declaration" {
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				return nameNode.Content(src)
			}
		}
		n = n.Parent()
	}
	return ""
}

// --- helpers ---

func (j *JavaParser) buildSymbol(
	kind constants.SymbolKind,
	name, receiver, pkgPath string,
	params, resultType string,
	defNode *sitter.Node, src []byte, filePath string,
) *models.Symbol {
	qualName := buildQualName(pkgPath, receiver, name)
	id := parser.SymbolID(qualName)

	start := int(defNode.StartByte())
	end := int(defNode.EndByte())

	comment := precedingComment(defNode, src)
	summary := parser.FirstSentence(comment)
	if summary == "" {
		summary = parser.TemplateSummary(string(kind), name, params, resultType)
	}

	sig := ""
	switch kind {
	case constants.KindFunction:
		sig = parser.BuildFuncSignature(receiver, name, params, resultType)
	case constants.KindMethod:
		sig = parser.BuildFuncSignature(receiver, name, params, resultType)
	}

	return &models.Symbol{
		ID:            id,
		Kind:          string(kind),
		Name:          name,
		QualifiedName: qualName,
		Language:      "java",
		File:          filePath,
		ByteStart:     &start,
		ByteEnd:       &end,
		Signature:     sig,
		Summary:       summary,
	}
}

func buildQualName(pkgPath, receiver, name string) string {
	var b strings.Builder
	b.WriteString(pkgPath)
	b.WriteByte('.')
	if receiver != "" {
		b.WriteString(receiver)
		b.WriteByte('.')
	}
	b.WriteString(name)
	return b.String()
}

func captureMap(captures []sitter.QueryCapture, q *sitter.Query, src []byte) map[string]string {
	m := make(map[string]string, len(captures))
	for _, c := range captures {
		m[q.CaptureNameForId(c.Index)] = c.Node.Content(src)
	}
	return m
}

func captureNode(captures []sitter.QueryCapture, q *sitter.Query, captureName string) *sitter.Node {
	for _, c := range captures {
		if q.CaptureNameForId(c.Index) == captureName {
			return c.Node
		}
	}
	return nil
}

func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	return parser.NormalizeWhitespace(s)
}

func precedingComment(defNode *sitter.Node, src []byte) string {
	parent := defNode.Parent()
	if parent == nil {
		return ""
	}
	defStart := defNode.StartByte()

	var comments []string
	for i := 0; i < int(parent.ChildCount()); i++ {
		child := parent.Child(i)
		if child.StartByte() >= defStart {
			break
		}
		t := child.Type()
		if t == "line_comment" || t == "block_comment" {
			comments = append(comments, child.Content(src))
		} else {
			comments = nil
		}
	}
	return strings.Join(comments, "\n")
}
