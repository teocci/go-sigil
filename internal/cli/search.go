package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"go-sigil/internal/service"
	"go-sigil/internal/store"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var kind, language, file string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "search <query> [path]",
		Short: "Search symbols by name or full-text",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			pathArg := "."
			if len(args) > 1 {
				pathArg = args[1]
			}
			root, err := resolveRepoRoot([]string{pathArg})
			if err != nil {
				return err
			}
			st, _, err := openStoreForRepo(root)
			if err != nil {
				return err
			}
			defer st.Close()

			svc := service.NewSearcher(st)
			if limit <= 0 {
				limit = 20
			}
			result, err := svc.Search(cmd.Context(), query, store.SearchOptions{
				Kind:     kind,
				Language: language,
				File:     file,
				Limit:    limit,
			})
			if err != nil {
				return fmt.Errorf("search: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			printSearchResult(w, result)
			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "filter by kind: function|method|class|interface|type|const|var")
	cmd.Flags().StringVar(&language, "language", "", "filter by language: go|typescript|javascript|python")
	cmd.Flags().StringVar(&file, "file", "", "filter by source file path")
	cmd.Flags().IntVar(&limit, "limit", 20, "max results")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func printSearchResult(w io.Writer, result *service.SearchResult) {
	fmt.Fprintf(w, "Found %d symbol(s)\n\n", result.Total)
	for _, sym := range result.Symbols {
		fmt.Fprintf(w, "  %s  %-12s  %s\n", sym.ID, sym.Kind, sym.QualifiedName)
		if sym.Signature != "" {
			fmt.Fprintf(w, "    sig: %s\n", sym.Signature)
		}
		if sym.Summary != "" {
			fmt.Fprintf(w, "    %s\n", sym.Summary)
		}
		fmt.Fprintln(w)
	}
}
