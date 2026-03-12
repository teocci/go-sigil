package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go-sigil/internal/constants"
	"go-sigil/internal/store"
)

// TreeNode represents a file or directory in the repo tree.
type TreeNode struct {
	Path        string     `json:"path"`
	IsDir       bool       `json:"is_dir"`
	Language    string     `json:"language,omitempty"`
	SymbolCount int        `json:"symbol_count,omitempty"`
	Supported   bool       `json:"supported"`
	SizeBytes   int64      `json:"size_bytes,omitempty"`
	Children    []TreeNode `json:"children,omitempty"`
}

// TreeResult holds the repository tree.
type TreeResult struct {
	Root  string     `json:"root"`
	Nodes []TreeNode `json:"nodes"`
}

// Tree builds repository file trees with symbol count annotation.
type Tree struct {
	st       store.SymbolStore
	repoRoot string
}

// NewTree creates a Tree service.
func NewTree(st store.SymbolStore, repoRoot string) *Tree {
	return &Tree{st: st, repoRoot: repoRoot}
}

// Build constructs the directory tree for scope, up to maxDepth levels.
// If sourceOnly is true, non-code files and directories with no code descendants are pruned.
func (t *Tree) Build(ctx context.Context, scope string, maxDepth int, includeSymbolCounts bool, sourceOnly ...bool) (*TreeResult, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	pruneUnsupported := len(sourceOnly) > 0 && sourceOnly[0]

	files, err := t.st.ListFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	symCounts := make(map[string]int)
	if includeSymbolCounts {
		for _, f := range files {
			syms, err := t.st.GetSymbolsByFile(ctx, f.Path)
			if err != nil {
				continue
			}
			symCounts[f.Path] = len(syms)
		}
	}

	scope = filepath.ToSlash(filepath.Clean(scope))
	if scope == "." {
		scope = ""
	}

	root := &dirNode{name: scope, children: map[string]*dirNode{}}
	for _, f := range files {
		p := f.Path
		if scope != "" {
			if !strings.HasPrefix(p, scope+"/") && p != scope {
				continue
			}
			p = strings.TrimPrefix(p, scope+"/")
		}
		insertPath(root, p, symCounts[f.Path], getFileSize(filepath.Join(t.repoRoot, f.Path)))
	}

	nodes := collectNodes(root, 0, maxDepth, pruneUnsupported)
	return &TreeResult{Root: scope, Nodes: nodes}, nil
}

type dirNode struct {
	name        string
	children    map[string]*dirNode
	isFile      bool
	language    string
	symbolCount int
	sizeBytes   int64
}

func insertPath(parent *dirNode, relPath string, symCount int, size int64) {
	parts := strings.SplitN(relPath, "/", 2)
	name := parts[0]
	if _, ok := parent.children[name]; !ok {
		parent.children[name] = &dirNode{name: name, children: map[string]*dirNode{}}
	}
	child := parent.children[name]
	if len(parts) == 1 {
		child.isFile = true
		child.symbolCount = symCount
		child.sizeBytes = size
		ext := strings.ToLower(filepath.Ext(name))
		child.language = constants.LanguageExtensions[ext]
	} else {
		insertPath(child, parts[1], symCount, size)
	}
}

func collectNodes(node *dirNode, depth, maxDepth int, sourceOnly bool) []TreeNode {
	if depth >= maxDepth {
		return nil
	}
	names := make([]string, 0, len(node.children))
	for n := range node.children {
		names = append(names, n)
	}
	sort.Strings(names)

	var nodes []TreeNode
	for _, name := range names {
		child := node.children[name]
		supported := child.language != ""

		// sourceOnly: skip unsupported leaf files entirely.
		if sourceOnly && child.isFile && !supported {
			continue
		}

		tn := TreeNode{
			Path:        name,
			IsDir:       !child.isFile,
			Language:    child.language,
			SymbolCount: child.symbolCount,
			Supported:   supported,
			SizeBytes:   child.sizeBytes,
		}
		if !child.isFile {
			tn.Children = collectNodes(child, depth+1, maxDepth, sourceOnly)
			// sourceOnly: prune directories that have no code descendants.
			if sourceOnly && len(tn.Children) == 0 {
				continue
			}
		}
		nodes = append(nodes, tn)
	}
	return nodes
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
