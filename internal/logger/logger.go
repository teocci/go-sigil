// Package logger provides structured logging setup using log/slog.
//
// In MCP stdio mode, logs MUST go to SIGIL_LOG_FILE to avoid
// corrupting the JSON-RPC stream on stdout.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Setup configures the global slog logger.
//
// logLevel: one of DEBUG, INFO, WARNING, ERROR (default: WARNING).
// logFile: file path for log output. If empty, logs go to stderr.
// Returns a closer function that should be deferred by the caller.
func Setup(logLevel string, logFile string) (cleanup func(), err error) {
	var w io.Writer
	cleanup = func() {} // no-op default

	if logFile != "" {
		f, ferr := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if ferr != nil {
			return nil, ferr
		}
		w = f
		cleanup = func() { f.Close() }
	} else {
		w = os.Stderr
	}

	level := parseLevel(logLevel)

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return cleanup, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARNING", "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
