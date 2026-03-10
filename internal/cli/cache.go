package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the ~/.sigil/ index cache",
	}
	cmd.AddCommand(newCacheStatusCmd())
	cmd.AddCommand(newCacheInvalidateCmd())
	cmd.AddCommand(newCacheRemoveCmd())
	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCachePruneCmd())
	return cmd
}

func newCacheStatusCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show cache size and registered repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := service.NewCacheManager(cfg.CacheRoot)
			status, err := svc.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("cache status: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(status)
			}

			fmt.Fprintf(w, "Cache root: %s\n", status.CacheRoot)
			fmt.Fprintf(w, "Total size: %s\n", formatBytes(status.TotalSize))
			fmt.Fprintf(w, "Repos: %d\n\n", len(status.Repos))
			for _, r := range status.Repos {
				fmt.Fprintf(w, "  %s  %-12s  %s  %s\n",
					r.Hash, formatBytes(r.SizeBytes), r.Name, r.Path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func newCacheInvalidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "invalidate [path]",
		Short: "Remove the index DB for a repo, forcing a full rebuild on next index",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			svc := service.NewCacheManager(cfg.CacheRoot)
			if err := svc.Invalidate(cmd.Context(), root); err != nil {
				return fmt.Errorf("cache invalidate: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Invalidated cache for %s\n", root)
			return nil
		},
	}
}

func newCacheRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [path]",
		Short: "Completely remove a repo's cache directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			svc := service.NewCacheManager(cfg.CacheRoot)
			if err := svc.Remove(cmd.Context(), root); err != nil {
				return fmt.Errorf("cache remove: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed cache for %s\n", root)
			return nil
		},
	}
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all cached repo data",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := service.NewCacheManager(cfg.CacheRoot)
			if err := svc.ClearAll(cmd.Context()); err != nil {
				return fmt.Errorf("cache clear: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "All cache data cleared.")
			return nil
		},
	}
}

func newCachePruneCmd() *cobra.Command {
	var olderThan string
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove cache entries not accessed within a duration",
		RunE: func(cmd *cobra.Command, args []string) error {
			dur, err := parseDuration(olderThan)
			if err != nil {
				return fmt.Errorf("invalid --older-than: %w", err)
			}
			svc := service.NewCacheManager(cfg.CacheRoot)
			n, err := svc.Prune(cmd.Context(), dur)
			if err != nil {
				return fmt.Errorf("cache prune: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Pruned %d repo(s).\n", n)
			return nil
		},
	}
	cmd.Flags().StringVar(&olderThan, "older-than", "720h", "remove entries older than this duration (e.g. 30d, 720h)")
	return cmd
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// parseDuration parses a duration string supporting "30d" in addition to
// standard Go durations like "720h", "24h", "10m".
func parseDuration(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		numStr := s[:len(s)-1]
		// Parse as hours then multiply by 24
		hours, err := time.ParseDuration(numStr + "h")
		if err != nil {
			return 0, err
		}
		return hours * 24, nil
	}
	return time.ParseDuration(s)
}
