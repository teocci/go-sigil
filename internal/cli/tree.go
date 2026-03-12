package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newTreeCmd() *cobra.Command {
	var depth int
	var noSymbols bool
	var sourceOnly bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "tree [scope] [path]",
		Short: "Show repository file structure with symbol counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := "."
			pathArg := "."
			if len(args) >= 1 {
				scope = args[0]
			}
			if len(args) >= 2 {
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

			svc := service.NewTree(st, root)
			result, err := svc.Build(cmd.Context(), scope, depth, !noSymbols, sourceOnly)
			if err != nil {
				return fmt.Errorf("tree: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			printNodes(w, result.Nodes, "")
			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 2, "max depth to display")
	cmd.Flags().BoolVar(&noSymbols, "no-symbols", false, "skip symbol count annotation")
	cmd.Flags().BoolVar(&sourceOnly, "source-only", false, "show only source code files and their parent directories")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func printNodes(w io.Writer, nodes []service.TreeNode, indent string) {
	for _, n := range nodes {
		if n.IsDir {
			fmt.Fprintf(w, "%s%s/\n", indent, n.Path)
			printNodes(w, n.Children, indent+"  ")
		} else {
			sym := ""
			if n.SymbolCount > 0 {
				sym = fmt.Sprintf("  [%d]", n.SymbolCount)
			}
			lang := ""
			if n.Language != "" {
				lang = fmt.Sprintf("  (%s)", n.Language)
			}
			fmt.Fprintf(w, "%s%s%s%s\n", indent, n.Path, lang, sym)
		}
	}
}
