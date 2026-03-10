package discovery

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// ignoreRule is a compiled gitignore pattern.
type ignoreRule struct {
	re      *regexp.Regexp
	negate  bool // pattern starts with !
	dirOnly bool // pattern had trailing /
}

// Ignorer applies gitignore-style rules to file paths.
// Rules are processed in order; the last matching rule wins.
type Ignorer struct {
	rules []ignoreRule
}

// NewIgnorer compiles the given patterns into an Ignorer.
func NewIgnorer(patterns []string) *Ignorer {
	ig := &Ignorer{}
	for _, p := range patterns {
		if r, ok := parseIgnorePattern(p); ok {
			ig.rules = append(ig.rules, r)
		}
	}
	return ig
}

// Ignore reports whether relPath should be ignored.
// isDir indicates whether relPath refers to a directory.
func (ig *Ignorer) Ignore(relPath string, isDir bool) bool {
	ignored, _ := ig.Check(relPath, isDir)
	return ignored
}

// Check returns the final ignore state and whether any rule matched.
// The last matching rule wins; negation rules (!) unignore the path.
func (ig *Ignorer) Check(relPath string, isDir bool) (ignored bool, matched bool) {
	for _, r := range ig.rules {
		if r.dirOnly && !isDir {
			continue
		}
		if r.re.MatchString(relPath) {
			ignored = !r.negate
			matched = true
		}
	}
	return
}

// LoadIgnoreFile reads a .gitignore or .sigilignore file and returns its lines.
// Returns nil, nil if the file does not exist.
func LoadIgnoreFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r")
		patterns = append(patterns, line)
	}
	return patterns, sc.Err()
}

// parseIgnorePattern converts a single gitignore line to a compiled rule.
// Returns (rule, false) for blank lines, comments, and invalid patterns.
func parseIgnorePattern(line string) (ignoreRule, bool) {
	if line == "" || strings.HasPrefix(line, "#") {
		return ignoreRule{}, false
	}

	// Literal # escaping: \# → #
	if strings.HasPrefix(line, "\\") {
		line = line[1:]
	}

	negate := false
	if strings.HasPrefix(line, "!") {
		negate = true
		line = line[1:]
		if line == "" {
			return ignoreRule{}, false
		}
	}

	dirOnly := strings.HasSuffix(line, "/")
	if dirOnly {
		line = strings.TrimSuffix(line, "/")
	}

	re, err := gitignorePatternToRegex(line)
	if err != nil {
		return ignoreRule{}, false
	}

	return ignoreRule{re: re, negate: negate, dirOnly: dirOnly}, true
}

// gitignorePatternToRegex converts a gitignore glob pattern to a regexp.
//
// Anchoring rules (from the gitignore spec):
//   - Pattern contains '/' not at the trailing position → anchored to root
//   - Pattern starts with '/' → anchored to root
//   - Otherwise → matches at any level
func gitignorePatternToRegex(pattern string) (*regexp.Regexp, error) {
	// Determine anchoring before any modifications
	trimmed := strings.TrimSuffix(pattern, "/")
	anchored := strings.Contains(trimmed, "/")

	// Remove leading slash for anchored patterns
	pattern = strings.TrimPrefix(pattern, "/")

	var sb strings.Builder
	if anchored {
		sb.WriteString("^")
	} else {
		sb.WriteString("(?:^|/)")
	}

	i := 0
	for i < len(pattern) {
		switch {
		case i+1 < len(pattern) && pattern[i] == '*' && pattern[i+1] == '*':
			// ** — match any number of path components
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				// **/ → zero or more directories
				sb.WriteString("(?:.+/)?")
				i += 3
			} else {
				// ** at end → match any characters including /
				sb.WriteString(".*")
				i += 2
			}
		case pattern[i] == '*':
			sb.WriteString("[^/]*")
			i++
		case pattern[i] == '?':
			sb.WriteString("[^/]")
			i++
		case pattern[i] == '[':
			// Character class — pass through literally
			end := strings.IndexByte(pattern[i:], ']')
			if end < 0 {
				sb.WriteString(regexp.QuoteMeta(string(pattern[i])))
				i++
			} else {
				sb.WriteString(pattern[i : i+end+1])
				i += end + 1
			}
		default:
			sb.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}

	// Allow trailing path components: pattern "foo" also matches "foo/bar"
	sb.WriteString("(?:/.*)?$")

	return regexp.Compile(sb.String())
}
