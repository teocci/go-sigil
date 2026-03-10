package security

import (
	"path/filepath"
	"regexp"
	"strings"

	"go-sigil/internal/constants"
)

// SecurityFilter implements Filter using the three-tier model.
type SecurityFilter struct {
	extraIgnore  []string         // TierIgnored patterns (config extra_ignore_filenames)
	extraSecret  []string         // TierRedacted patterns (config extra_secret_filenames)
	secretValues []*regexp.Regexp // compiled constant-value secret patterns
}

// NewFilter builds a SecurityFilter from the given configuration overrides.
// extraIgnore are additional fully-ignored file patterns.
// extraSecret are additional redacted file patterns.
// extraValuePatterns are additional regex patterns for secret constant values.
func NewFilter(extraIgnore, extraSecret, extraValuePatterns []string) (*SecurityFilter, error) {
	patterns := append(constants.SecretValuePatterns, extraValuePatterns...)
	compiled, err := compilePatterns(patterns)
	if err != nil {
		return nil, err
	}
	return &SecurityFilter{
		extraIgnore:  extraIgnore,
		extraSecret:  extraSecret,
		secretValues: compiled,
	}, nil
}

// ClassifyFile returns the security tier for path.
// Checks are ordered: ignored first, then redacted, then normal.
func (f *SecurityFilter) ClassifyFile(path string) Tier {
	name := filepath.Base(path)
	// Extra ignore patterns → TierIgnored
	for _, pat := range f.extraIgnore {
		if patternMatches(pat, name) {
			return TierIgnored
		}
	}
	// Built-in redacted patterns + extra secret patterns → TierRedacted
	for _, pat := range append(constants.BuiltinRedactedPatterns, f.extraSecret...) {
		if patternMatches(pat, name) {
			return TierRedacted
		}
	}
	return TierNormal
}

// IsSecretValue reports whether value matches any compiled secret pattern.
func (f *SecurityFilter) IsSecretValue(value string) bool {
	for _, re := range f.secretValues {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}

// patternMatches checks whether name matches a single gitignore-style pattern.
// Patterns may use * and ? glob characters. Plain names require exact match.
func patternMatches(pat, name string) bool {
	// Exact match check (fast path for patterns like ".env")
	if !strings.ContainsAny(pat, "*?[") {
		return name == pat
	}
	ok, _ := filepath.Match(pat, name)
	return ok
}

// compilePatterns compiles a slice of regex strings into *regexp.Regexp values.
func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}
