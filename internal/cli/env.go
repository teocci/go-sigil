package cli

import (
	"encoding/json"
	"fmt"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "env [path]",
		Short: "Inspect environment variable configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}

			svc := service.NewEnvService(root)
			result, err := svc.Inspect(cmd.Context())
			if err != nil {
				return fmt.Errorf("env: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintln(w, "Env files:")
			for _, f := range result.EnvFiles {
				status := "missing"
				if f.Exists {
					status = "found"
				}
				fmt.Fprintf(w, "  %-30s  %s\n", f.Path, status)
			}

			if len(result.Variables) > 0 {
				fmt.Fprintln(w, "\nVariables:")
				for _, v := range result.Variables {
					note := ""
					if v.Note != "" {
						note = "  # " + v.Note
					}
					fmt.Fprintf(w, "  %-40s  %s%s\n", v.Key, v.State, note)
				}
			}

			if len(result.Warnings) > 0 {
				fmt.Fprintln(w, "\nWarnings:")
				for _, w2 := range result.Warnings {
					fmt.Fprintf(w, "  ! %s\n", w2)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
