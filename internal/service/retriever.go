package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-sigil/internal/constants"
	"go-sigil/internal/store"
)

// SymbolWithSource extends Symbol with its extracted source and context lines.
type SymbolWithSource struct {
	ID                 string `json:"id"`
	QualifiedName      string `json:"qualified_name"`
	File               string `json:"file"`
	LineStart          int    `json:"line_start"`
	LineEnd            int    `json:"line_end"`
	Source             string `json:"source"`
	ContextBeforeLines int    `json:"context_before_lines"`
	ContextAfterLines  int    `json:"context_after_lines"`
	PossibleUnresolved bool   `json:"possible_unresolved,omitempty"`
}

// FileContent holds the raw content of a requested file.
type FileContent struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Language  string `json:"language,omitempty"`
	Supported bool   `json:"supported"`
	SizeBytes int64  `json:"size_bytes"`
}

// GetResult holds the result of a sigil get request.
type GetResult struct {
	Symbols []SymbolWithSource `json:"symbols,omitempty"`
	Files   []FileContent      `json:"files,omitempty"`
}

// Retriever retrieves symbols and files by ID or path.
type Retriever struct {
	st       store.SymbolStore
	filesDir string // ~/.sigil/{hash}/files/
	repoRoot string
}

// NewRetriever creates a Retriever.
func NewRetriever(st store.SymbolStore, filesDir, repoRoot string) *Retriever {
	return &Retriever{st: st, filesDir: filesDir, repoRoot: repoRoot}
}

// Get retrieves symbols by IDs and raw file contents by paths.
// contextLines is clamped to [0, 50].
func (r *Retriever) Get(ctx context.Context, ids, files []string, contextLines int) (*GetResult, error) {
	if contextLines < 0 {
		contextLines = 0
	}
	if contextLines > 50 {
		contextLines = 50
	}

	result := &GetResult{}

	if len(ids) > 0 {
		syms, err := r.st.GetSymbolsByIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("get symbols: %w", err)
		}
		for _, sym := range syms {
			src, err := r.extractSource(sym.File, sym.ByteStart, sym.ByteEnd, contextLines)
			if err != nil {
				src = fmt.Sprintf("[source unavailable: %v]", err)
			}
			result.Symbols = append(result.Symbols, SymbolWithSource{
				ID:                 sym.ID,
				QualifiedName:      sym.QualifiedName,
				File:               sym.File,
				LineStart:          sym.LineStart,
				LineEnd:            sym.LineEnd,
				Source:             src,
				ContextBeforeLines: contextLines,
				ContextAfterLines:  contextLines,
				PossibleUnresolved: sym.PossibleUnresolved,
			})
		}
	}

	for _, path := range files {
		content, size, err := r.readFile(path)
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}
		ext := strings.ToLower(filepath.Ext(path))
		lang := constants.LanguageExtensions[ext]
		result.Files = append(result.Files, FileContent{
			Path:      path,
			Content:   content,
			Language:  lang,
			Supported: lang != "",
			SizeBytes: size,
		})
	}

	return result, nil
}

// extractSource reads symbol source from the repo root.
// Returns the symbol body plus context lines.
func (r *Retriever) extractSource(relPath string, byteStart, byteEnd *int, contextLines int) (string, error) {
	if byteStart == nil || byteEnd == nil {
		return "[redacted]", nil
	}
	data, err := r.readFileRaw(relPath)
	if err != nil {
		return "", err
	}
	start, end := *byteStart, *byteEnd
	if start < 0 || end > len(data) || start > end {
		return "", fmt.Errorf("invalid byte range [%d:%d] for file size %d", start, end, len(data))
	}

	ctxStart := start
	ctxEnd := end
	if contextLines > 0 {
		ctxStart = expandBack(data, start, contextLines)
		ctxEnd = expandForward(data, end, contextLines)
	}

	return string(data[ctxStart:ctxEnd]), nil
}

func (r *Retriever) readFileRaw(relPath string) ([]byte, error) {
	absPath := filepath.Join(r.repoRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("file not found: %s", relPath)
}

func (r *Retriever) readFile(relPath string) (string, int64, error) {
	data, err := r.readFileRaw(relPath)
	if err != nil {
		return "", 0, err
	}
	return string(data), int64(len(data)), nil
}

// expandBack counts n newlines backward from pos, returns the start of that line.
func expandBack(data []byte, pos, n int) int {
	count := 0
	i := pos - 1
	for i >= 0 && count < n {
		if data[i] == '\n' {
			count++
		}
		i--
	}
	if i < 0 {
		return 0
	}
	return i + 1
}

// expandForward counts n newlines forward from pos, returns the end of that line.
func expandForward(data []byte, pos, n int) int {
	count := 0
	i := pos
	for i < len(data) && count < n {
		if data[i] == '\n' {
			count++
		}
		i++
	}
	return i
}
