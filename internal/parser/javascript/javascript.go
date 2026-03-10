// Package javascript implements the Sigil parser for JavaScript using tree-sitter.
package javascript

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	jssitter "github.com/smacker/go-tree-sitter/javascript"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// JSParser extracts symbols and call edges from JavaScript source files.
// A single instance is safe for concurrent use.
type JSParser struct {
	lang *sitter.Language
}

// New returns a JSParser ready for use.
func New() *JSParser {
	return &JSParser{lang: jssitter.GetLanguage()}
}

// Language returns the canonical language name.
func (j *JSParser) Language() string { return "javascript" }

// Parse extracts symbols and call edges from JavaScript source.
// filePath is the repo-relative path; pkgPath is the directory portion.
func (j *JSParser) Parse(filePath, pkgPath string, src []byte) (*parser.ParseResult, error) {
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

	if err := j.extractFunctions(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := j.extractClasses(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := j.extractMethods(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	j.extractEdges(root, src, result)

	return result, nil
}

// --- symbol extraction ---

func (j *JSParser) extractFunctions(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryFunctions, j.lang)
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
		result.Symbols = append(result.Symbols, j.buildSymbol(
			constants.KindFunction, name, "", pkgPath, params, defNode, src, filePath,
		))
	}
	return nil
}

func (j *JSParser) extractClasses(
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
		sym := j.buildSymbol(constants.KindClass, name, "", pkgPath, "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "class")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (j *JSParser) extractMethods(
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
		className := enclosingClassName(defNode, src)
		params := stripOuterParens(caps["method.params"])
		result.Symbols = append(result.Symbols, j.buildSymbol(
			constants.KindMethod, name, className, pkgPath, params, defNode, src, filePath,
		))
	}
	return nil
}

// --- call edge extraction ---

func (j *JSParser) extractEdges(root *sitter.Node, src []byte, result *parser.ParseResult) {
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
	if q, err := sitter.NewQuery(querySelectorCalls, j.lang); err == nil {
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
// In JavaScript, class names use (identifier); method_definition → class_body → class_declaration.
func enclosingClassName(node *sitter.Node, src []byte) string {
	n := node.Parent() // class_body
	if n == nil {
		return ""
	}
	n = n.Parent() // class_declaration (or export_statement)
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

func (j *JSParser) buildSymbol(
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
		Language:      "javascript",
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
