package cli

import (
	"encoding/json"
	"fmt"

	"go-sigil/internal/service"
	"go-sigil/internal/storage"

	"github.com/spf13/cobra"
)

func newOverviewCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "overview [path]",
		Short: "Show high-level repository summary",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			st, repoHash, err := openStoreForRepo(root)
			if err != nil {
				return err
			}
			defer st.Close()

			meta, err := storage.ReadMeta(cfg.CacheRoot, repoHash)
			if err != nil {
				return fmt.Errorf("read meta: %w", err)
			}

			svc := service.NewOverview(st, meta)
			result, err := svc.Summary(cmd.Context())
			if err != nil {
				return fmt.Errorf("overview: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(w, "Repository: %s\n", result.Repo)
			fmt.Fprintf(w, "  Files:    %d\n", result.TotalFiles)
			fmt.Fprintf(w, "  Symbols:  %d\n", result.TotalSymbols)
			if result.PossibleUnresolvedCount > 0 {
				fmt.Fprintf(w, "  Unresolved edges: %d\n", result.PossibleUnresolvedCount)
			}
			if result.LastIndexedCommit != "" {
				fmt.Fprintf(w, "  Last commit: %s\n", result.LastIndexedCommit)
			}
			fmt.Fprintf(w, "  Index age: %.0fs\n", result.IndexAgeSeconds)

			if len(result.Languages) > 0 {
				fmt.Fprintln(w, "\nLanguages:")
				for _, l := range result.Languages {
					fmt.Fprintf(w, "  %-14s  %d files  %d symbols\n", l.Language, l.Files, l.Symbols)
				}
			}

			if len(result.Packages) > 0 {
				fmt.Fprintln(w, "\nPackages:")
				for _, p := range result.Packages {
					fmt.Fprintf(w, "  %-40s  %d symbols\n", p.Root, p.Symbols)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
