package parser

import (
	"strings"
)

// FirstSentence extracts the first sentence (up to '.', '!', '?' or end) from a comment block.
// Leading slashes and '*' are stripped. Empty string is returned if no comment is present.
func FirstSentence(comment string) string {
	if comment == "" {
		return ""
	}

	// Strip comment markers line by line.
	var lines []string
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		// Strip // prefix
		line = strings.TrimPrefix(line, "//")
		// Strip /* and */ markers
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		// Strip leading * (block comment style)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return ""
	}

	text := strings.Join(lines, " ")

	// Extract first sentence — up to the first '.', '!', or '?'.
	for i, ch := range text {
		if ch == '.' || ch == '!' || ch == '?' {
			return text[:i+1]
		}
	}
	return text
}

// TemplateSummary returns a template-based fallback summary when no docstring is present.
// kind is one of: function, method, type, interface, const, var.
func TemplateSummary(kind, name, params, result string) string {
	switch kind {
	case "function", "method":
		if params != "" && result != "" {
			return kind + " " + name + " accepting " + params + ", returns " + result
		}
		if params != "" {
			return kind + " " + name + " accepting " + params
		}
		if result != "" {
			return kind + " " + name + " returns " + result
		}
		return kind + " " + name
	case "type", "interface":
		return kind + " " + name
	case "const":
		return "constant " + name
	case "var":
		return "variable " + name
	default:
		return kind + " " + name
	}
}
