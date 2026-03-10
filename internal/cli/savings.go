package cli

import (
	"encoding/json"
	"fmt"

	"go-sigil/internal/service"

	"github.com/spf13/cobra"
)

func newSavingsCmd() *cobra.Command {
	var sessions bool
	var sessionID string
	var top int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "savings [path]",
		Short: "Show token savings analytics",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			st, _, err := openStoreForRepo(root)
			if err != nil {
				return err
			}
			defer st.Close()

			svc := service.NewSavings(st)
			w := cmd.OutOrStdout()

			switch {
			case sessionID != "":
				summary, err := svc.SessionSavings(cmd.Context(), sessionID)
				if err != nil {
					return fmt.Errorf("session savings: %w", err)
				}
				if jsonOut {
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(summary)
				}
				fmt.Fprintf(w, "Session %s\n", sessionID)
				fmt.Fprintf(w, "  Tokens saved: %d\n", summary.TokensSaved)
				fmt.Fprintf(w, "  API calls:    %d\n", summary.CallCount)

			case top > 0:
				list, err := svc.TopSessions(cmd.Context(), top)
				if err != nil {
					return fmt.Errorf("top sessions: %w", err)
				}
				if jsonOut {
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(list)
				}
				fmt.Fprintf(w, "Top %d sessions by tokens saved:\n\n", top)
				for i, s := range list {
					fmt.Fprintf(w, "  %2d. %s  tokens=%d  calls=%d\n",
						i+1, s.SessionID, s.TokensSaved, s.CallCount)
				}

			case sessions:
				list, err := svc.ListSessions(cmd.Context())
				if err != nil {
					return fmt.Errorf("list sessions: %w", err)
				}
				if jsonOut {
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(list)
				}
				fmt.Fprintf(w, "Sessions (%d):\n\n", len(list))
				for _, s := range list {
					fmt.Fprintf(w, "  %s  tokens=%d  calls=%d  last=%s\n",
						s.SessionID, s.TokensSaved, s.CallCount, s.LastCall)
				}

			default:
				summary, err := svc.RepoSavings(cmd.Context())
				if err != nil {
					return fmt.Errorf("repo savings: %w", err)
				}
				if jsonOut {
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(summary)
				}
				fmt.Fprintf(w, "Repo savings:\n")
				fmt.Fprintf(w, "  Tokens saved: %d\n", summary.TokensSaved)
				fmt.Fprintf(w, "  API calls:    %d\n", summary.CallCount)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&sessions, "sessions", false, "list all sessions")
	cmd.Flags().StringVar(&sessionID, "session", "", "show savings for a specific session ID")
	cmd.Flags().IntVar(&top, "top", 0, "show top N sessions by tokens saved")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
