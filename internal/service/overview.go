package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-sigil/internal/models"
	"go-sigil/internal/store"
	"go-sigil/internal/storage"
)

// OverviewResult holds the high-level repository summary.
type OverviewResult struct {
	Repo                    string                `json:"repo"`
	Languages               []models.LanguageStat `json:"languages"`
	Packages                []models.PackageStat  `json:"packages"`
	TopLevelDirs            []string              `json:"top_level_dirs"`
	TotalSymbols            int                   `json:"total_symbols"`
	TotalFiles              int                   `json:"total_files"`
	IndexAgeSeconds         float64               `json:"index_age_seconds"`
	LastIndexedCommit       string                `json:"last_indexed_commit,omitempty"`
	PossibleUnresolvedCount int                   `json:"possible_unresolved_count"`
}

// OverviewService produces the repository overview.
type OverviewService struct {
	st   store.SymbolStore
	meta *storage.RepoMeta
}

// NewOverview creates an OverviewService.
func NewOverview(st store.SymbolStore, meta *storage.RepoMeta) *OverviewService {
	return &OverviewService{st: st, meta: meta}
}

// Summary returns the high-level repository summary.
func (o *OverviewService) Summary(ctx context.Context) (*OverviewResult, error) {
	totalSymbols, err := o.st.CountSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("count symbols: %w", err)
	}
	totalFiles, err := o.st.CountFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}
	possibleUnresolved, err := o.st.CountPossibleUnresolved(ctx)
	if err != nil {
		return nil, fmt.Errorf("count unresolved: %w", err)
	}
	langs, err := o.st.GetLanguageStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get language stats: %w", err)
	}
	pkgs, err := o.st.GetPackageStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get package stats: %w", err)
	}

	files, err := o.st.ListFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	topDirs := make(map[string]bool)
	for _, f := range files {
		parts := strings.SplitN(f.Path, "/", 2)
		topDirs[parts[0]] = true
	}
	dirs := make([]string, 0, len(topDirs))
	for d := range topDirs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	var indexAge float64
	var repoName string
	var lastCommit string
	if o.meta != nil {
		repoName = o.meta.Name
		lastCommit = o.meta.LastIndexedCommit
		if o.meta.InitializedAt != "" {
			if t, err := time.Parse(time.RFC3339, o.meta.InitializedAt); err == nil {
				indexAge = time.Since(t).Seconds()
			}
		}
	}

	return &OverviewResult{
		Repo:                    repoName,
		Languages:               langs,
		Packages:                pkgs,
		TopLevelDirs:            dirs,
		TotalSymbols:            totalSymbols,
		TotalFiles:              totalFiles,
		IndexAgeSeconds:         indexAge,
		LastIndexedCommit:       lastCommit,
		PossibleUnresolvedCount: possibleUnresolved,
	}, nil
}
