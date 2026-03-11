package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var jsonOut bool
	var since string

	cmd := &cobra.Command{
		Use:   "diff [--since <ref>] [ref] [path]",
		Short: "Show symbol-level diff since a git ref",
		Args:  cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --since flag takes precedence over positional arg
			if since == "" {
				if len(args) == 0 {
					return fmt.Errorf("git ref required: use --since <ref> or pass ref as first argument")
				}
				since = args[0]
				args = args[1:]
			}
			pathArg := "."
			if len(args) > 0 {
				pathArg = args[0]
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

			svc := service.NewDiffer(st, root)
			result, err := svc.Diff(cmd.Context(), since)
			if err != nil {
				return fmt.Errorf("diff: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			printDiffResult(w, result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().StringVar(&since, "since", "", "git ref to diff since (commit hash, branch, or tag)")
	return cmd
}

func printDiffResult(w io.Writer, result *service.DiffResult) {
	fmt.Fprintf(w, "Symbol diff since %s\n\n", result.Since)
	if len(result.Added) > 0 {
		fmt.Fprintf(w, "Added (%d):\n", len(result.Added))
		for _, s := range result.Added {
			fmt.Fprintf(w, "  + %s  %s  %s\n", s.ID, s.Kind, s.QualifiedName)
		}
		fmt.Fprintln(w)
	}
	if len(result.Modified) > 0 {
		fmt.Fprintf(w, "Modified (%d):\n", len(result.Modified))
		for _, s := range result.Modified {
			fmt.Fprintf(w, "  ~ %s  %s  %s\n", s.ID, s.Kind, s.QualifiedName)
		}
		fmt.Fprintln(w)
	}
	if len(result.Deleted) > 0 {
		fmt.Fprintf(w, "Deleted (%d):\n", len(result.Deleted))
		for _, s := range result.Deleted {
			fmt.Fprintf(w, "  - %s  %s  %s\n", s.ID, s.Kind, s.QualifiedName)
		}
		fmt.Fprintln(w)
	}
	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "Errors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(w, "  ! %s\n", e)
		}
	}
}
