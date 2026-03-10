// Package golang implements the Sigil parser for the Go language using tree-sitter.
package golang

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	gositter "github.com/smacker/go-tree-sitter/golang"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// GoParser extracts symbols and call edges from Go source files.
// A single instance is safe for concurrent use.
type GoParser struct {
	lang *sitter.Language
}

// New returns a GoParser ready for use.
func New() *GoParser {
	return &GoParser{lang: gositter.GetLanguage()}
}

// Language returns the canonical language name.
func (g *GoParser) Language() string { return "go" }

// Parse extracts symbols and call edges from Go source.
// filePath is the repo-relative file path; pkgPath is the directory portion.
func (g *GoParser) Parse(filePath, pkgPath string, src []byte) (*parser.ParseResult, error) {
	p := sitter.NewParser()
	p.SetLanguage(g.lang)
	defer p.Close()

	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse %s: %w", filePath, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	result := &parser.ParseResult{}

	if err := g.extractFunctions(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := g.extractMethods(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := g.extractTypes(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := g.extractConsts(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := g.extractVars(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	g.extractEdges(root, src, result)

	return result, nil
}

// --- symbol extraction ---

func (g *GoParser) extractFunctions(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryFunctions, g.lang)
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
		result.Symbols = append(result.Symbols, g.buildSymbol(
			constants.KindFunction, name, "", pkgPath, params, "", defNode, src, filePath,
		))
	}
	return nil
}

func (g *GoParser) extractMethods(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryMethods, g.lang)
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
		receiver := extractReceiverType(caps["method.receiver"])
		params := stripOuterParens(caps["method.params"])
		result.Symbols = append(result.Symbols, g.buildSymbol(
			constants.KindMethod, name, receiver, pkgPath, params, "", defNode, src, filePath,
		))
	}
	return nil
}

func (g *GoParser) extractTypes(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryTypes, g.lang)
	if err != nil {
		return fmt.Errorf("types query: %w", err)
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
		bodyNode := captureNode(m.Captures, q, "type.body")
		if name == "" || defNode == nil {
			continue
		}

		kind := constants.KindType
		typeKindWord := "struct"
		if bodyNode != nil {
			switch bodyNode.Type() {
			case "interface_type":
				kind = constants.KindInterface
				typeKindWord = "interface"
			case "struct_type":
				kind = constants.KindType
				typeKindWord = "struct"
			default:
				typeKindWord = bodyNode.Type()
			}
		}

		sym := g.buildSymbol(kind, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, typeKindWord)
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (g *GoParser) extractConsts(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryConsts, g.lang)
	if err != nil {
		return fmt.Errorf("consts query: %w", err)
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
		name := caps["const.name"]
		defNode := captureNode(m.Captures, q, "const.def")
		if name == "" || defNode == nil {
			continue
		}
		// Skip function-local const declarations — only extract package-level ones.
		if defNode.Parent() == nil || defNode.Parent().Type() != "source_file" {
			continue
		}
		result.Symbols = append(result.Symbols,
			g.buildSymbol(constants.KindConst, name, "", pkgPath, "", "", defNode, src, filePath),
		)
	}
	return nil
}

func (g *GoParser) extractVars(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryVars, g.lang)
	if err != nil {
		return fmt.Errorf("vars query: %w", err)
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
		name := caps["var.name"]
		defNode := captureNode(m.Captures, q, "var.def")
		if name == "" || defNode == nil {
			continue
		}
		// Skip function-local var declarations — only extract package-level ones.
		if defNode.Parent() == nil || defNode.Parent().Type() != "source_file" {
			continue
		}
		result.Symbols = append(result.Symbols,
			g.buildSymbol(constants.KindVar, name, "", pkgPath, "", "", defNode, src, filePath),
		)
	}
	return nil
}

// --- call edge extraction ---

// extractEdges finds all call expressions in the file and records them as edges
// with the enclosing symbol as caller. callee_id is resolved later by the indexer.
func (g *GoParser) extractEdges(root *sitter.Node, src []byte, result *parser.ParseResult) {
	// Build byte-range → symbol ID index for caller lookup.
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
				CalleeID:      "", // resolved later by indexer
				RawExpression: caps["call.expr"],
				Confidence:    string(constants.ConfidenceDynamic),
			})
		}
	}

	if q, err := sitter.NewQuery(queryDirectCalls, g.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
	if q, err := sitter.NewQuery(querySelectorCalls, g.lang); err == nil {
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

// --- helpers ---

// buildSymbol constructs a models.Symbol from extracted node data.
func (g *GoParser) buildSymbol(
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
		sig = parser.BuildFuncSignature("", name, params, resultType)
	case constants.KindMethod:
		sig = parser.BuildFuncSignature(receiver, name, params, resultType)
	}

	return &models.Symbol{
		ID:            id,
		Kind:          string(kind),
		Name:          name,
		QualifiedName: qualName,
		Language:      "go",
		File:          filePath,
		ByteStart:     &start,
		ByteEnd:       &end,
		Signature:     sig,
		Summary:       summary,
	}
}

// buildQualName returns "{pkgPath}.{Receiver}.{Name}" or "{pkgPath}.{Name}".
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

// captureMap returns capture name → node text for all captures in a match.
func captureMap(captures []sitter.QueryCapture, q *sitter.Query, src []byte) map[string]string {
	m := make(map[string]string, len(captures))
	for _, c := range captures {
		m[q.CaptureNameForId(c.Index)] = c.Node.Content(src)
	}
	return m
}

// captureNode returns the node for the first occurrence of captureName, or nil.
func captureNode(captures []sitter.QueryCapture, q *sitter.Query, captureName string) *sitter.Node {
	for _, c := range captures {
		if q.CaptureNameForId(c.Index) == captureName {
			return c.Node
		}
	}
	return nil
}

// stripOuterParens removes surrounding '(' ')' from a parameter list text.
func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	return parser.NormalizeWhitespace(s)
}

// extractReceiverType parses "(r *ReceiverType)" → "ReceiverType".
func extractReceiverType(receiverList string) string {
	inner := strings.TrimSpace(receiverList)
	inner = strings.TrimPrefix(inner, "(")
	inner = strings.TrimSuffix(inner, ")")
	parts := strings.Fields(strings.TrimSpace(inner))
	if len(parts) == 0 {
		return ""
	}
	typ := strings.TrimPrefix(parts[len(parts)-1], "*")
	if idx := strings.IndexByte(typ, '['); idx >= 0 {
		typ = typ[:idx]
	}
	return typ
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
