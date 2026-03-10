// Package python implements the Sigil parser for Python using tree-sitter.
package python

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	pysitter "github.com/smacker/go-tree-sitter/python"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// PyParser extracts symbols and call edges from Python source files.
// A single instance is safe for concurrent use.
type PyParser struct {
	lang *sitter.Language
}

// New returns a PyParser ready for use.
func New() *PyParser {
	return &PyParser{lang: pysitter.GetLanguage()}
}

// Language returns the canonical language name.
func (py *PyParser) Language() string { return "python" }

// Parse extracts symbols and call edges from Python source.
// filePath is the repo-relative path; pkgPath is the directory portion.
func (py *PyParser) Parse(filePath, pkgPath string, src []byte) (*parser.ParseResult, error) {
	p := sitter.NewParser()
	p.SetLanguage(py.lang)
	defer p.Close()

	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse %s: %w", filePath, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	result := &parser.ParseResult{}

	// Classes first so method association via parent-walk works correctly.
	if err := py.extractClasses(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := py.extractFunctions(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	py.extractEdges(root, src, result)

	return result, nil
}

// --- symbol extraction ---

func (py *PyParser) extractClasses(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryClasses, py.lang)
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
		sym := py.buildSymbol(constants.KindClass, name, "", pkgPath, "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "class")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

// extractFunctions queries all function_definition nodes and classifies each as
// a module-level function or a class method by walking up the parent tree.
func (py *PyParser) extractFunctions(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryFunctions, py.lang)
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
		className := enclosingPythonClass(defNode, src)

		kind := constants.KindFunction
		if className != "" {
			kind = constants.KindMethod
		}

		result.Symbols = append(result.Symbols, py.buildSymbol(
			kind, name, className, pkgPath, params, defNode, src, filePath,
		))
	}
	return nil
}

// --- call edge extraction ---

func (py *PyParser) extractEdges(root *sitter.Node, src []byte, result *parser.ParseResult) {
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

	if q, err := sitter.NewQuery(queryDirectCalls, py.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
	if q, err := sitter.NewQuery(queryAttributeCalls, py.lang); err == nil {
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

// enclosingPythonClass walks up from a function_definition to detect if it is
// a class method. Path: function_definition → block → class_definition.
// Returns the class name, or "" for module-level functions.
func enclosingPythonClass(node *sitter.Node, src []byte) string {
	n := node.Parent() // expected: block
	if n == nil || n.Type() != "block" {
		return ""
	}
	n = n.Parent() // expected: class_definition or decorated_definition
	// Skip decorated_definition wrapper if present.
	if n != nil && n.Type() == "decorated_definition" {
		n = n.Parent()
	}
	if n != nil && n.Type() == "class_definition" {
		nameNode := n.ChildByFieldName("name")
		if nameNode != nil {
			return nameNode.Content(src)
		}
	}
	return ""
}

// --- helpers ---

func (py *PyParser) buildSymbol(
	kind constants.SymbolKind,
	name, receiver, pkgPath string,
	params string,
	defNode *sitter.Node, src []byte, filePath string,
) *models.Symbol {
	qualName := buildQualName(pkgPath, receiver, name)
	id := parser.SymbolID(qualName)

	start := int(defNode.StartByte())
	end := int(defNode.EndByte())

	// Python prefers docstrings over preceding comments.
	summary := parser.FirstSentence(docstring(defNode, src))
	if summary == "" {
		summary = parser.FirstSentence(precedingComment(defNode, src))
	}
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
		Language:      "python",
		File:          filePath,
		ByteStart:     &start,
		ByteEnd:       &end,
		Signature:     sig,
		Summary:       summary,
	}
}

// docstring extracts the first string literal from a function_definition or
// class_definition body block, which is the conventional Python docstring.
func docstring(defNode *sitter.Node, src []byte) string {
	body := defNode.ChildByFieldName("body")
	if body == nil || body.ChildCount() == 0 {
		return ""
	}
	first := body.Child(0)
	if first == nil || first.Type() != "expression_statement" || first.ChildCount() == 0 {
		return ""
	}
	strNode := first.Child(0)
	if strNode == nil || strNode.Type() != "string" {
		return ""
	}
	// Extract string_content child (avoids the quote characters).
	for i := 0; i < int(strNode.ChildCount()); i++ {
		child := strNode.Child(i)
		if child.Type() == "string_content" {
			return child.Content(src)
		}
	}
	return ""
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
