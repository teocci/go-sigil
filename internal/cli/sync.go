package cli

import (
	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [path]",
		Short: "Incrementally sync a repository (used by git hook)",
		Long: `Sync performs an incremental update of the symbol index.

Only files that have changed since the last index run are re-parsed.
Intended for use by the pre-commit git hook (installed via sigil hook install).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			return runIndex(cmd, root, service.IndexOptions{Force: false}, false, false)
		},
	}
}
