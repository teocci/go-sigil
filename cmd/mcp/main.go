// Command sigil-mcp is the MCP stdio server for the Sigil framework.
// It serves symbol queries to AI agents over JSON-RPC via stdin/stdout.
//
// IMPORTANT: Nothing must write to stdout while the server is running —
// it will corrupt the JSON-RPC stream. Set SIGIL_LOG_FILE to redirect logs.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go-sigil/internal/config"
	"go-sigil/internal/logger"
	"go-sigil/internal/mcpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sigil-mcp: load config: %v\n", err)
		os.Exit(1)
	}

	// In MCP stdio mode SIGIL_LOG_FILE must be set — logs must never reach stdout.
	if cfg.LogFile == "" {
		fmt.Fprintln(os.Stderr, "sigil-mcp: warning: SIGIL_LOG_FILE is not set — logs will be written to stderr")
	}
	logger.Setup(cfg.LogLevel, cfg.LogFile)

	srv, err := mcpserver.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sigil-mcp: create server: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "sigil-mcp: %v\n", err)
		os.Exit(1)
	}
}
