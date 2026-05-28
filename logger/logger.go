// Package logger configures the global slog logger with a settable level.
// All packages call slog.Debug/Info/Warn/Error directly; this package only
// needs to be imported for its Init side-effect.
package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// level controls the minimum log level at runtime.
var level = &slog.LevelVar{} // defaults to LevelInfo (0)

func init() {
	// Default to Warn so only Warn and Error are shown out of the box.
	level.Set(slog.LevelWarn)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

// Set parses a level string (debug, info, warn, error — case-insensitive)
// and applies it to the global logger. Returns an error for unknown values.
func Set(s string) error {
	switch strings.ToLower(s) {
	case "debug":
		level.Set(slog.LevelDebug)
	case "info":
		level.Set(slog.LevelInfo)
	case "warn", "warning":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		return fmt.Errorf("unknown log level %q: want debug, info, warn, or error", s)
	}
	return nil
}
