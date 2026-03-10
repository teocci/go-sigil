package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newOutlineCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "outline <file> [path]",
		Short: "Show all symbols in a file as a structured hierarchy",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
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

			svc := service.NewOutline(st)
			result, err := svc.ForFile(cmd.Context(), file)
			if err != nil {
				return fmt.Errorf("outline: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(w, "%s  (%d symbols)\n\n", result.File, len(result.Symbols))
			for _, sym := range result.Symbols {
				indent := strings.Repeat("  ", sym.Depth)
				fmt.Fprintf(w, "%s%s %s", indent, sym.Kind, sym.Name)
				if sym.Signature != "" {
					fmt.Fprintf(w, " — %s", sym.Signature)
				}
				fmt.Fprintln(w)
				if sym.Summary != "" {
					fmt.Fprintf(w, "%s  %s\n", indent, sym.Summary)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
