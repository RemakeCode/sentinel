package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

func TestParseLevel_Debug(t *testing.T) {
	level := ParseLevel("debug")
	assert.Equal(t, slog.LevelDebug, level)
}

func TestParseLevel_Info(t *testing.T) {
	level := ParseLevel("info")
	assert.Equal(t, slog.LevelInfo, level)
}

func TestParseLevel_Off(t *testing.T) {
	level := ParseLevel("off")
	assert.Equal(t, slog.Level(100), level)
}

func TestParseLevel_Unknown(t *testing.T) {
	level := ParseLevel("unknown")
	assert.Equal(t, slog.LevelInfo, level) // Default to info
}

func TestParseLevel_Empty(t *testing.T) {
	level := ParseLevel("")
	assert.Equal(t, slog.LevelInfo, level) // Default to info
}

func TestSetLevel(t *testing.T) {
	// Save original level
	originalLevel := levelVar.Level()
	defer levelVar.Set(originalLevel)

	SetLevel(slog.LevelDebug)
	assert.Equal(t, slog.LevelDebug, levelVar.Level())

	SetLevel(slog.LevelInfo)
	assert.Equal(t, slog.LevelInfo, levelVar.Level())

	SetLevel(slog.Level(100))
	assert.Equal(t, slog.Level(100), levelVar.Level())
}

func TestNewLogFileWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	writer := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1, // 1 MB for testing
		MaxBackups: 3,
	}

	logger := NewWithFile(writer)
	logger.Info("test message")

	writer.Close()

	// Verify file was created
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test message")
}

func TestNewWithFile_Fallback(t *testing.T) {
	logger := NewWithFile(nil)
	assert.NotNil(t, logger)
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "rotation.log")

	writer := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1, // 1 KB - small for testing
		MaxBackups: 3,
	}

	logger := NewWithFile(writer)

	// Write enough data to trigger rotation (messages are ~60-100 bytes each)
	// 25,000 messages * ~60 bytes = ~1.5 MB, which exceeds the 1MB limit.
	for i := 0; i < 25000; i++ {
		logger.Info("test message for rotation", "index", i)
	}

	writer.Close()

	// Verify rotated files exist
	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 1)
}
