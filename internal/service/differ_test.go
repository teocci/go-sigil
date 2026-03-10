package service

import (
	"testing"
)

// TestParseGitNameStatus tests the internal git name-status parser.
// The Diff() method itself requires git and is covered by integration tests.
func TestParseGitNameStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantKeys map[string]string
	}{
		{
			name: "modified file",
			input: "M\tinternal/service/searcher.go\n",
			wantLen: 1,
			wantKeys: map[string]string{"internal/service/searcher.go": "M"},
		},
		{
			name: "added file",
			input: "A\tcmd/new.go\n",
			wantLen: 1,
			wantKeys: map[string]string{"cmd/new.go": "A"},
		},
		{
			name: "deleted file",
			input: "D\told/file.go\n",
			wantLen: 1,
			wantKeys: map[string]string{"old/file.go": "D"},
		},
		{
			name: "multiple files",
			input: "M\ta.go\nA\tb.go\nD\tc.go\n",
			wantLen: 3,
			wantKeys: map[string]string{"a.go": "M", "b.go": "A", "c.go": "D"},
		},
		{
			name:    "empty output",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "blank lines ignored",
			input:   "\n\n",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitNameStatus(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d (result: %v)", len(result), tt.wantLen, result)
			}
			for path, wantStatus := range tt.wantKeys {
				got, ok := result[path]
				if !ok {
					t.Errorf("missing path %q in result", path)
					continue
				}
				if got != wantStatus {
					t.Errorf("result[%q] = %q, want %q", path, got, wantStatus)
				}
			}
		})
	}
}
