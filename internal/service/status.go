package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-sigil/internal/store"
	"go-sigil/internal/storage"
)

// VerifyResult holds symbol content hash verification results.
type VerifyResult struct {
	Total      int `json:"total"`
	Mismatched int `json:"mismatched"`
}

// StatusResult holds the index health report.
type StatusResult struct {
	Repo                    string        `json:"repo"`
	Path                    string        `json:"path"`
	LastIndexedCommit       string        `json:"last_indexed_commit,omitempty"`
	CommitValid             bool          `json:"commit_valid"`
	IndexAgeSeconds         float64       `json:"index_age_seconds"`
	TotalFiles              int           `json:"total_files"`
	TotalSymbols            int           `json:"total_symbols"`
	PossibleUnresolvedCount int           `json:"possible_unresolved_count"`
	Mode                    string        `json:"mode"`
	Verification            *VerifyResult `json:"verification,omitempty"`
}

// StatusService checks index health.
type StatusService struct {
	st       store.SymbolStore
	meta     *storage.RepoMeta
	repoRoot string
}

// NewStatus creates a StatusService.
func NewStatus(st store.SymbolStore, meta *storage.RepoMeta, repoRoot string) *StatusService {
	return &StatusService{st: st, meta: meta, repoRoot: repoRoot}
}

// Check returns the index health report.
// If verify is true, re-reads symbol sources to detect content hash mismatches.
func (s *StatusService) Check(ctx context.Context, verify bool) (*StatusResult, error) {
	totalFiles, _ := s.st.CountFiles(ctx)
	totalSymbols, _ := s.st.CountSymbols(ctx)
	possUnresolved, _ := s.st.CountPossibleUnresolved(ctx)

	var indexAge float64
	var repoName, lastCommit, mode string
	commitValid := false

	if s.meta != nil {
		repoName = s.meta.Name
		lastCommit = s.meta.LastIndexedCommit
		mode = s.meta.Mode
		if s.meta.InitializedAt != "" {
			if t, err := time.Parse(time.RFC3339, s.meta.InitializedAt); err == nil {
				indexAge = time.Since(t).Seconds()
			}
		}
		if lastCommit != "" {
			commitValid = isCommitReachable(s.repoRoot, lastCommit)
		}
	}
	if mode == "" {
		mode = "filesystem"
	}

	result := &StatusResult{
		Repo:                    repoName,
		Path:                    s.repoRoot,
		LastIndexedCommit:       lastCommit,
		CommitValid:             commitValid,
		IndexAgeSeconds:         indexAge,
		TotalFiles:              totalFiles,
		TotalSymbols:            totalSymbols,
		PossibleUnresolvedCount: possUnresolved,
		Mode:                    mode,
	}

	if verify {
		vr, err := s.verifyContentHashes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify: %w", err)
		}
		result.Verification = vr
	}

	return result, nil
}

func (s *StatusService) verifyContentHashes(ctx context.Context) (*VerifyResult, error) {
	files, err := s.st.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	vr := &VerifyResult{}
	for _, f := range files {
		syms, err := s.st.GetSymbolsByFile(ctx, f.Path)
		if err != nil {
			continue
		}
		absPath := filepath.Join(s.repoRoot, filepath.FromSlash(f.Path))
		src, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		for _, sym := range syms {
			if sym.ByteStart == nil || sym.ByteEnd == nil {
				continue
			}
			s2, e2 := *sym.ByteStart, *sym.ByteEnd
			if s2 < 0 || e2 > len(src) || s2 > e2 {
				continue
			}
			vr.Total++
			h := sha256.Sum256(src[s2:e2])
			computed := hex.EncodeToString(h[:])[:16]
			if computed != sym.ContentHash {
				vr.Mismatched++
			}
		}
	}
	return vr, nil
}

func isCommitReachable(repoRoot, commit string) bool {
	cmd := exec.Command("git", "-C", repoRoot, "cat-file", "-t", commit)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "commit"
}
