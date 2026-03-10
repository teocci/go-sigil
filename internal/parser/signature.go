package parser

import "strings"

// BuildFuncSignature builds a function signature string:
// "func Name(params) result" or "func (recv) Name(params) result".
func BuildFuncSignature(receiver, name, params, result string) string {
	var b strings.Builder
	b.WriteString("func ")
	if receiver != "" {
		b.WriteByte('(')
		b.WriteString(receiver)
		b.WriteString(") ")
	}
	b.WriteString(name)
	b.WriteByte('(')
	b.WriteString(params)
	b.WriteByte(')')
	if result != "" {
		b.WriteByte(' ')
		b.WriteString(result)
	}
	return b.String()
}

// BuildTypeSignature builds a short type signature: "type Name kind".
func BuildTypeSignature(name, kind string) string {
	return "type " + name + " " + kind
}

// NormalizeWhitespace collapses runs of whitespace/newlines into a single space.
func NormalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
