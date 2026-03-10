package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newDepsCmd() *cobra.Command {
	var direction string
	var depth int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "deps <symbol-id> [path]",
		Short: "Trace call graph for a symbol",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbolID := args[0]
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

			svc := service.NewDeps(st)
			result, err := svc.Trace(cmd.Context(), symbolID, direction, depth)
			if err != nil {
				return fmt.Errorf("deps: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			printDepsResult(w, result)
			return nil
		},
	}

	cmd.Flags().StringVar(&direction, "direction", "both", "calls|callers|both")
	cmd.Flags().IntVar(&depth, "depth", 1, "traversal depth")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func printDepsResult(w io.Writer, result *service.DepsResult) {
	if len(result.Calls) > 0 {
		fmt.Fprintf(w, "Calls (%d):\n", len(result.Calls))
		for _, e := range result.Calls {
			printDepsEdge(w, e)
		}
	}
	if len(result.CalledBy) > 0 {
		fmt.Fprintf(w, "\nCalled by (%d):\n", len(result.CalledBy))
		for _, e := range result.CalledBy {
			printDepsEdge(w, e)
		}
	}
	if result.HasDynamicEdges {
		fmt.Fprintln(w, "\nNote: dynamic dispatch detected — call graph may be incomplete.")
	}
	if result.PossibleUnresolved {
		fmt.Fprintln(w, "Note: symbol has possible unresolved call edges.")
	}
}

func printDepsEdge(w io.Writer, e service.DepsEdge) {
	id := e.ID
	if id == "" {
		id = "(unresolved)"
	}
	name := e.QualifiedName
	if name == "" {
		name = e.RawExpression
	}
	if name == "" {
		name = id
	}
	fmt.Fprintf(w, "  [%s] %s  (%s)\n", e.Confidence, name, id)
	if e.Summary != "" {
		fmt.Fprintf(w, "    %s\n", e.Summary)
	}
}
