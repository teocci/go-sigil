// Package typescript implements the Sigil parser for TypeScript using tree-sitter.
package typescript

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tssitter "github.com/smacker/go-tree-sitter/typescript/typescript"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// TSParser extracts symbols and call edges from TypeScript source files.
// A single instance is safe for concurrent use.
type TSParser struct {
	lang *sitter.Language
}

// New returns a TSParser ready for use.
func New() *TSParser {
	return &TSParser{lang: tssitter.GetLanguage()}
}

// Language returns the canonical language name.
func (t *TSParser) Language() string { return "typescript" }

// Parse extracts symbols and call edges from TypeScript source.
// filePath is the repo-relative path; pkgPath is the directory portion.
func (t *TSParser) Parse(filePath, pkgPath string, src []byte) (*parser.ParseResult, error) {
	p := sitter.NewParser()
	p.SetLanguage(t.lang)
	defer p.Close()

	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse %s: %w", filePath, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	result := &parser.ParseResult{}

	if err := t.extractFunctions(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := t.extractClasses(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := t.extractMethods(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := t.extractInterfaces(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := t.extractTypeAliases(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	t.extractEdges(root, src, result)

	return result, nil
}

// --- symbol extraction ---

func (t *TSParser) extractFunctions(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryFunctions, t.lang)
	if err != nil {
		return fmt.Errorf("functions query: %w", err)
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
		name := caps["func.name"]
		defNode := captureNode(m.Captures, q, "func.def")
		if name == "" || defNode == nil {
			continue
		}
		params := stripOuterParens(caps["func.params"])
		result.Symbols = append(result.Symbols, t.buildSymbol(
			constants.KindFunction, name, "", pkgPath, params, defNode, src, filePath,
		))
	}
	return nil
}

func (t *TSParser) extractClasses(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryClasses, t.lang)
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
		sym := t.buildSymbol(constants.KindClass, name, "", pkgPath, "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "class")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (t *TSParser) extractMethods(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryMethods, t.lang)
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
		className := enclosingClassName(defNode, src)
		params := stripOuterParens(caps["method.params"])
		result.Symbols = append(result.Symbols, t.buildSymbol(
			constants.KindMethod, name, className, pkgPath, params, defNode, src, filePath,
		))
	}
	return nil
}

func (t *TSParser) extractInterfaces(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryInterfaces, t.lang)
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
		name := caps["iface.name"]
		defNode := captureNode(m.Captures, q, "iface.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := t.buildSymbol(constants.KindInterface, name, "", pkgPath, "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "interface")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (t *TSParser) extractTypeAliases(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryTypeAliases, t.lang)
	if err != nil {
		return fmt.Errorf("type aliases query: %w", err)
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
		name := caps["type.name"]
		defNode := captureNode(m.Captures, q, "type.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := t.buildSymbol(constants.KindType, name, "", pkgPath, "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "type")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

// --- call edge extraction ---

func (t *TSParser) extractEdges(root *sitter.Node, src []byte, result *parser.ParseResult) {
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

	if q, err := sitter.NewQuery(queryDirectCalls, t.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
	if q, err := sitter.NewQuery(querySelectorCalls, t.lang); err == nil {
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

// enclosingClassName walks up from a method_definition to find the parent class name.
func enclosingClassName(node *sitter.Node, src []byte) string {
	n := node.Parent() // class_body
	if n == nil {
		return ""
	}
	n = n.Parent() // class_declaration (or export_statement)
	for n != nil {
		switch n.Type() {
		case "class_declaration", "abstract_class_declaration":
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

func (t *TSParser) buildSymbol(
	kind constants.SymbolKind,
	name, receiver, pkgPath string,
	params string,
	defNode *sitter.Node, src []byte, filePath string,
) *models.Symbol {
	qualName := buildQualName(pkgPath, receiver, name)
	id := parser.SymbolID(qualName)

	start := int(defNode.StartByte())
	end := int(defNode.EndByte())

	comment := precedingComment(defNode, src)
	summary := parser.FirstSentence(comment)
	if summary == "" {
		summary = parser.TemplateSummary(string(kind), name, params, "")
	}

	sig := ""
	switch kind {
	case constants.KindFunction:
		sig = parser.BuildFuncSignature("", name, params, "")
	case constants.KindMethod:
		sig = parser.BuildFuncSignature(receiver, name, params, "")
	}

	return &models.Symbol{
		ID:            id,
		Kind:          string(kind),
		Name:          name,
		QualifiedName: qualName,
		Language:      "typescript",
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

// precedingComment returns the text of comment nodes immediately before defNode.
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
		if child.Type() == "comment" {
			comments = append(comments, child.Content(src))
		} else {
			comments = nil // non-comment breaks the doc-comment window
		}
	}
	return strings.Join(comments, "\n")
}
