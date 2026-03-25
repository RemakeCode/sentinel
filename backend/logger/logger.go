package logger

import (
	"log/slog"
	"os"
	"sentinel/backend"
	"strings"

	"github.com/lmittmann/tint"
)

var (
	levelVar = new(slog.LevelVar)
)

// SetLevel updates the global log level
func SetLevel(level slog.Level) {
	levelVar.Set(level)
}

// ParseLevel converts a string level to slog.Level
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "off":
		return slog.Level(100)
	default:
		return slog.LevelInfo
	}
}

// New returns a new slog.Logger instance with sanitization and formatting
func New() *slog.Logger {
	// Prepare sanitization prefixes
	homeDir, _ := os.UserHomeDir()

	// ReplaceAttr for path sanitization
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Value.Kind() == slog.KindString {
			val := a.Value.String()

			// Sanitize common paths
			if backend.ConfigDir != "" {
				val = strings.ReplaceAll(val, backend.ConfigDir, "<CONFIG_DIR>")
			}
			if homeDir != "" {
				val = strings.ReplaceAll(val, homeDir, "<HOME>")
			}
			if backend.UserCacheDir != "" {
				val = strings.ReplaceAll(val, backend.UserCacheDir, "<CACHE_DIR>")
			}

			return slog.String(a.Key, val)
		}
		return a
	}

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:       levelVar,
		ReplaceAttr: replace,
		TimeFormat:  "15:04:05",
	})

	return slog.New(handler)
}
