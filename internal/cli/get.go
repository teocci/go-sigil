package cli

import (
	"encoding/json"
	"fmt"

	"go-sigil/internal/service"
	"go-sigil/internal/storage"

	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var contextLines int
	var files []string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "get <id> [id...] [path]",
		Short: "Retrieve symbol source by ID",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, pathArg := splitIDsAndPath(args)

			root, err := resolveRepoRoot([]string{pathArg})
			if err != nil {
				return err
			}

			st, repoHash, err := openStoreForRepo(root)
			if err != nil {
				return err
			}
			defer st.Close()

			filesDir := storage.FilesDir(cfg.CacheRoot, repoHash)
			svc := service.NewRetriever(st, filesDir, root)
			result, err := svc.Get(cmd.Context(), ids, files, contextLines)
			if err != nil {
				return fmt.Errorf("get: %w", err)
			}

			w := cmd.OutOrStdout()
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			for _, sym := range result.Symbols {
				fmt.Fprintf(w, "=== %s (%s) — %s:%d-%d ===\n",
					sym.QualifiedName, sym.ID, sym.File, sym.LineStart, sym.LineEnd)
				fmt.Fprintln(w, sym.Source)
				fmt.Fprintln(w)
			}
			for _, f := range result.Files {
				fmt.Fprintf(w, "=== %s ===\n", f.Path)
				fmt.Fprintln(w, f.Content)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&contextLines, "context", 0, "context lines before and after (0-50)")
	cmd.Flags().StringSliceVar(&files, "file", nil, "raw file paths to retrieve")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

// splitIDsAndPath separates symbol IDs from an optional trailing path argument.
// A path arg is distinguishable from IDs by being longer than 12 chars
// or starting with a path-like character.
func splitIDsAndPath(args []string) (ids []string, pathArg string) {
	pathArg = "."
	if len(args) == 0 {
		return nil, pathArg
	}
	last := args[len(args)-1]
	if isPathLike(last) {
		return args[:len(args)-1], last
	}
	return args, pathArg
}

func isPathLike(s string) bool {
	if len(s) == 0 {
		return false
	}
	switch s[0] {
	case '/', '.', '~':
		return true
	}
	// Windows absolute path
	if len(s) >= 2 && s[1] == ':' {
		return true
	}
	// Longer than a symbol ID (8 hex chars)
	return len(s) > 12
}
