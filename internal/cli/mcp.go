package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"go-sigil/internal/mcpserver"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP stdio server",
		Long: `Start the Sigil MCP stdio server.

Configure in your MCP client (e.g. Claude Code):
  {
    "mcpServers": {
      "sigil": {
        "command": "sigil",
        "args": ["mcp"],
        "env": {"SIGIL_LOG_FILE": "/tmp/sigil-mcp.log"}
      }
    }
  }

IMPORTANT: Set SIGIL_LOG_FILE — logs must never reach stdout in MCP mode.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cfg.LogFile == "" {
				fmt.Fprintln(os.Stderr, "sigil mcp: warning: SIGIL_LOG_FILE is not set — logs will be written to stderr")
			}

			srv, err := mcpserver.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return srv.Run(ctx)
		},
		// Silence usage on error: MCP errors go to stderr, not help text.
		SilenceUsage: true,
	}
}
