// Package cli defines all Cobra commands for the sigil CLI.
// Commands handle ONLY input validation and output formatting —
// all business logic is delegated to the service layer.
package cli

import (
	"fmt"
	"os"

	"go-sigil/internal/config"
	"go-sigil/internal/logger"

	"github.com/spf13/cobra"
)

var cfg *config.Config

// NewRootCmd creates the root sigil command with all subcommands registered.
func NewRootCmd() *cobra.Command {
	var logLevel string
	var logFile string

	root := &cobra.Command{
		Use:   "sigil",
		Short: "Token-efficient codebase intelligence framework",
		Long:  "Sigil indexes source code into a queryable symbol store for AI agents,\nenabling surgical retrieval with 70-97% token reduction.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// CLI flags override config/env
			if cmd.Flags().Changed("log-level") {
				cfg.LogLevel = logLevel
			}
			if cmd.Flags().Changed("log-file") {
				cfg.LogFile = logFile
			}

			cleanup, err := logger.Setup(cfg.LogLevel, cfg.LogFile)
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			// Store cleanup for deferred call — in practice the process exits
			_ = cleanup

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level: DEBUG, INFO, WARNING, ERROR")
	root.PersistentFlags().StringVar(&logFile, "log-file", "", "log output file path")

	// Register subcommands
	root.AddCommand(newVersionCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newSyncCmd())

	return root
}

// Execute runs the root command. Called from main.go.
func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
