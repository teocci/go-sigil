package security

import (
	"fmt"
	"regexp"
	"strings"

	"go-sigil/internal/constants"
)

// RedactedSummaryPrefix is prepended to the summary of any redacted symbol.
const RedactedSummaryPrefix = "[REDACTED — matched secret pattern"

// RedactSummary returns a redacted summary string for a symbol whose value
// matched secretPattern (a human-readable description of the pattern).
func RedactSummary(secretPattern string) string {
	return fmt.Sprintf("%s: %s]", RedactedSummaryPrefix, secretPattern)
}

// PlaceholderRegexes holds compiled regexes for placeholder detection.
// Used by sigil_env to classify env var values as set/empty/placeholder/unset.
var PlaceholderRegexes []*regexp.Regexp

func init() {
	for _, p := range constants.PlaceholderPatterns {
		re, err := regexp.Compile(p)
		if err == nil {
			PlaceholderRegexes = append(PlaceholderRegexes, re)
		}
	}
}

// IsPlaceholder reports whether value looks like a placeholder (not a real secret).
func IsPlaceholder(value string) bool {
	if value == "" {
		return false
	}
	for _, re := range PlaceholderRegexes {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}

// SecretPatternName returns a human-readable name for the secret pattern that
// matched value, or "" if no pattern matched.
func SecretPatternName(value string, compiled []*regexp.Regexp) string {
	names := []string{
		"API key assignment",
		"secret assignment",
		"password assignment",
		"token assignment",
		"private key header",
		"database connection string",
	}
	for i, re := range compiled {
		if re.MatchString(value) {
			if i < len(names) {
				return names[i]
			}
			return "secret value"
		}
	}
	return ""
}

// DefaultSecretPatterns compiles the built-in SecretValuePatterns.
func DefaultSecretPatterns() ([]*regexp.Regexp, error) {
	return compilePatterns(constants.SecretValuePatterns)
}

// MatchSecretPattern reports whether value matches any of the provided patterns
// and returns a human-readable description of the first match.
func MatchSecretPattern(value string, patterns []*regexp.Regexp) (bool, string) {
	for i, re := range patterns {
		if re.MatchString(value) {
			return true, patternDescription(i)
		}
	}
	return false, ""
}

// patternDescription maps a pattern index to a human-readable description.
func patternDescription(i int) string {
	descriptions := []string{
		"API key assignment",
		"secret assignment",
		"password assignment",
		"token assignment",
		"private key header",
		"database connection string",
	}
	if i < len(descriptions) {
		return descriptions[i]
	}
	return "secret value"
}

// RedactedSummaryFor returns a redacted summary string.
func RedactedSummaryFor(value string, patterns []*regexp.Regexp) string {
	_, desc := MatchSecretPattern(value, patterns)
	if desc == "" {
		desc = "secret pattern"
	}
	return fmt.Sprintf("[REDACTED — matched %s]", desc)
}

// ContainsSecretHint reports whether a variable name suggests a secret
// (fast heuristic without regex, used for quick pre-filtering).
func ContainsSecretHint(name string) bool {
	lower := strings.ToLower(name)
	for _, hint := range []string{"key", "secret", "password", "passwd", "token", "credential", "private"} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}
