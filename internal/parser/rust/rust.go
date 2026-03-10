// Package rust implements the Sigil parser for the Rust language using tree-sitter.
package rust

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	rustsitter "github.com/smacker/go-tree-sitter/rust"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
	"go-sigil/internal/parser"
)

// RustParser extracts symbols and call edges from Rust source files.
// A single instance is safe for concurrent use.
type RustParser struct {
	lang *sitter.Language
}

// New returns a RustParser ready for use.
func New() *RustParser {
	return &RustParser{lang: rustsitter.GetLanguage()}
}

// Language returns the canonical language name.
func (r *RustParser) Language() string { return "rust" }

// Parse extracts symbols and call edges from Rust source.
// filePath is the repo-relative file path; pkgPath is the directory portion.
func (r *RustParser) Parse(filePath, pkgPath string, src []byte) (*parser.ParseResult, error) {
	p := sitter.NewParser()
	p.SetLanguage(r.lang)
	defer p.Close()

	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse %s: %w", filePath, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	result := &parser.ParseResult{}

	if err := r.extractFunctions(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := r.extractImplMethods(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := r.extractStructs(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := r.extractTraits(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := r.extractEnums(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := r.extractTypeAliases(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	if err := r.extractConsts(root, src, filePath, pkgPath, result); err != nil {
		return nil, err
	}
	r.extractEdges(root, src, result)

	return result, nil
}

// --- symbol extraction ---

func (r *RustParser) extractFunctions(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryFunctions, r.lang)
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
		name := caps["fn.name"]
		defNode := captureNode(m.Captures, q, "fn.def")
		if name == "" || defNode == nil {
			continue
		}
		// Only extract top-level functions (parent is source_file).
		if defNode.Parent() == nil || defNode.Parent().Type() != "source_file" {
			continue
		}
		params := stripOuterParens(caps["fn.params"])
		result.Symbols = append(result.Symbols, r.buildSymbol(
			constants.KindFunction, name, "", pkgPath, params, "", defNode, src, filePath,
		))
	}
	return nil
}

func (r *RustParser) extractImplMethods(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryImplMethods, r.lang)
	if err != nil {
		return fmt.Errorf("impl methods query: %w", err)
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
		implType := caps["impl.type"]
		defNode := captureNode(m.Captures, q, "method.def")
		if name == "" || defNode == nil {
			continue
		}
		receiver := stripAngleBrackets(implType)
		params := stripOuterParens(caps["method.params"])
		result.Symbols = append(result.Symbols, r.buildSymbol(
			constants.KindMethod, name, receiver, pkgPath, params, "", defNode, src, filePath,
		))
	}
	return nil
}

func (r *RustParser) extractStructs(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryStructs, r.lang)
	if err != nil {
		return fmt.Errorf("structs query: %w", err)
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
		name := caps["struct.name"]
		defNode := captureNode(m.Captures, q, "struct.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := r.buildSymbol(constants.KindType, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "struct")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (r *RustParser) extractTraits(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryTraits, r.lang)
	if err != nil {
		return fmt.Errorf("traits query: %w", err)
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
		name := caps["trait.name"]
		defNode := captureNode(m.Captures, q, "trait.def")
		if name == "" || defNode == nil {
			continue
		}
		sym := r.buildSymbol(constants.KindInterface, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "trait")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (r *RustParser) extractEnums(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryEnums, r.lang)
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
		sym := r.buildSymbol(constants.KindType, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "enum")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (r *RustParser) extractTypeAliases(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryTypeAliases, r.lang)
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
		sym := r.buildSymbol(constants.KindType, name, "", pkgPath, "", "", defNode, src, filePath)
		sym.Signature = parser.BuildTypeSignature(name, "type")
		result.Symbols = append(result.Symbols, sym)
	}
	return nil
}

func (r *RustParser) extractConsts(
	root *sitter.Node, src []byte, filePath, pkgPath string, result *parser.ParseResult,
) error {
	q, err := sitter.NewQuery(queryConsts, r.lang)
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
		// Only extract top-level consts.
		if defNode.Parent() == nil || defNode.Parent().Type() != "source_file" {
			continue
		}
		result.Symbols = append(result.Symbols,
			r.buildSymbol(constants.KindConst, name, "", pkgPath, "", "", defNode, src, filePath),
		)
	}
	return nil
}

// --- call edge extraction ---

func (r *RustParser) extractEdges(root *sitter.Node, src []byte, result *parser.ParseResult) {
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

	if q, err := sitter.NewQuery(queryDirectCalls, r.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
	if q, err := sitter.NewQuery(queryScopedCalls, r.lang); err == nil {
		runCallQuery(q)
		q.Close()
	}
	if q, err := sitter.NewQuery(queryMethodCalls, r.lang); err == nil {
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

func (r *RustParser) buildSymbol(
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
		Language:      "rust",
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

// stripAngleBrackets removes generic type parameters: "Vec<T>" → "Vec".
func stripAngleBrackets(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '<'); idx >= 0 {
		s = s[:idx]
	}
	return s
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
		if t == "line_comment" || t == "block_comment" || t == "doc_comment" {
			comments = append(comments, child.Content(src))
		} else {
			comments = nil
		}
	}
	return strings.Join(comments, "\n")
}
