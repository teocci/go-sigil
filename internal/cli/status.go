package cli

import (
	"encoding/json"
	"fmt"

	"go-sigil/internal/service"
	"go-sigil/internal/storage"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var verify bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status [path]",
		Short: "Show index health for a repository",
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

			svc := service.NewStatus(st, meta, root)
			result, err := svc.Check(cmd.Context(), verify)
			if err != nil {
				return fmt.Errorf("status: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(w, "Repository: %s\n", result.Repo)
			fmt.Fprintf(w, "  Path:      %s\n", result.Path)
			fmt.Fprintf(w, "  Mode:      %s\n", result.Mode)
			fmt.Fprintf(w, "  Files:     %d\n", result.TotalFiles)
			fmt.Fprintf(w, "  Symbols:   %d\n", result.TotalSymbols)
			fmt.Fprintf(w, "  Index age: %.0fs\n", result.IndexAgeSeconds)
			if result.LastIndexedCommit != "" {
				valid := "invalid"
				if result.CommitValid {
					valid = "valid"
				}
				fmt.Fprintf(w, "  Commit:    %s (%s)\n", result.LastIndexedCommit, valid)
			}
			if result.PossibleUnresolvedCount > 0 {
				fmt.Fprintf(w, "  Unresolved: %d\n", result.PossibleUnresolvedCount)
			}
			if result.Verification != nil {
				fmt.Fprintf(w, "\nVerification:\n")
				fmt.Fprintf(w, "  Checked:    %d\n", result.Verification.Total)
				fmt.Fprintf(w, "  Mismatched: %d\n", result.Verification.Mismatched)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&verify, "verify", false, "verify content hashes against source files")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
