package logger

import (
	"io"
	"log/slog"
	"os"
	"sentinel/backend"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	levelVar = new(slog.LevelVar)
)

// NewLogFileWriter creates a lumberjack logger for file output with rotation
func NewLogFileWriter() *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   backend.LogFilePath,
		MaxSize:    10, // 10 MB
		MaxBackups: 3,
		MaxAge:     0, // No age limit
		Compress:   false,
	}
}

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
// It writes to both stderr and a log file with automatic rotation
func New() *slog.Logger {
	// Prepare sanitization prefixes
	homeDir, _ := os.UserHomeDir()

	// ReplaceAttr for path sanitization and time formatting
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.String(slog.TimeKey, a.Value.Any().(time.Time).Format("15:04:05"))
		}
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

	logWriter := NewLogFileWriter()
	output := io.MultiWriter(os.Stderr, logWriter)

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level:       levelVar,
		ReplaceAttr: replace,
	})

	return slog.New(handler)
}

// NewWithFile returns a new slog.Logger that writes to both stderr and a log file.
// If fileWriter is nil, it falls back to stderr-only output and logs a warning.
func NewWithFile(fileWriter *lumberjack.Logger) *slog.Logger {
	// Prepare sanitization prefixes
	homeDir, _ := os.UserHomeDir()

	// ReplaceAttr for path sanitization and time formatting
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.String(slog.TimeKey, a.Value.Any().(time.Time).Format("15:04:05"))
		}
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

	var output io.Writer = os.Stderr

	if fileWriter != nil {
		output = io.MultiWriter(os.Stderr, fileWriter)
	} else {
		slog.Warn("Log file writer unavailable, falling back to stderr-only output")
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level:       levelVar,
		ReplaceAttr: replace,
	})

	return slog.New(handler)
}
