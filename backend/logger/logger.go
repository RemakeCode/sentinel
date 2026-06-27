package logger

import (
	"io"
	"log/slog"
	"os"
	"sentinel/backend"
	"strings"

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

// New returns a new slog.Logger instance with sanitization and formatting.
// It writes to both stdout and a log file with automatic rotation.
func New() *slog.Logger {
	return NewWithFile(NewLogFileWriter())
}

// NewWithFile returns a new slog.Logger that writes to both stdout and a log file.
// If fileWriter is nil, it falls back to stdout-only output and logs a warning.
func NewWithFile(fileWriter *lumberjack.Logger) *slog.Logger {
	var output io.Writer = os.Stdout

	if fileWriter != nil {
		output = io.MultiWriter(os.Stdout, fileWriter)
	} else {
		slog.Warn("Log file writer unavailable, falling back to stdout-only output")
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level:       levelVar,
		ReplaceAttr: newReplaceAttr(),
	})

	return slog.New(handler)
}

func newReplaceAttr() func([]string, slog.Attr) slog.Attr {
	homeDir, _ := os.UserHomeDir()

	return func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
			return slog.String(slog.TimeKey, a.Value.Time().Format("15:04:05"))
		}
		if a.Value.Kind() == slog.KindString {
			val := a.Value.String()

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
}
