package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const hookScript = "#!/bin/sh\nsigil sync\n"

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage sigil git hooks",
	}
	cmd.AddCommand(newHookInstallCmd())
	cmd.AddCommand(newHookUninstallCmd())
	return cmd
}

func newHookInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [path]",
		Short: "Install pre-commit hook: runs sigil sync before every commit",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")

			// Don't overwrite a hook that doesn't contain sigil
			if data, err := os.ReadFile(hookPath); err == nil {
				if !strings.Contains(string(data), "sigil") {
					return fmt.Errorf("pre-commit hook exists at %s and doesn't contain sigil — refusing to overwrite", hookPath)
				}
			}

			if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
				return fmt.Errorf("write hook: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed pre-commit hook at %s\n", hookPath)
			return nil
		},
	}
}

func newHookUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [path]",
		Short: "Remove sigil pre-commit hook",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRepoRoot(args)
			if err != nil {
				return err
			}
			hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")

			data, err := os.ReadFile(hookPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(cmd.OutOrStdout(), "No pre-commit hook found at %s\n", hookPath)
					return nil
				}
				return fmt.Errorf("read hook: %w", err)
			}

			if !strings.Contains(string(data), "sigil") {
				return fmt.Errorf("hook at %s was not installed by sigil — refusing to remove", hookPath)
			}

			if err := os.Remove(hookPath); err != nil {
				return fmt.Errorf("remove hook: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed pre-commit hook at %s\n", hookPath)
			return nil
		},
	}
}
