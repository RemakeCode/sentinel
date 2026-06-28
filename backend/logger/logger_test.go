package logger

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sentinel/backend"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.NoError(t, writer.Close())

	// Verify file was created
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test message")
}

func TestNewWithFile_Fallback(t *testing.T) {
	logger := NewWithFile(nil)
	assert.NotNil(t, logger)
}

func TestNewWithFile_SanitizesPathsInLogOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "sanitized.log")
	configDir := filepath.Join(tmpDir, "config")
	cacheDir := filepath.Join(tmpDir, "cache")

	originalConfigDir := backend.ConfigDir
	originalCacheDir := backend.UserCacheDir
	backend.ConfigDir = configDir
	backend.UserCacheDir = cacheDir
	defer func() {
		backend.ConfigDir = originalConfigDir
		backend.UserCacheDir = originalCacheDir
	}()

	writer := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1,
		MaxBackups: 1,
	}

	logger := NewWithFile(writer)
	logger.Info(
		"paths",
		"config", filepath.Join(configDir, "config.json"),
		"cache", filepath.Join(cacheDir, "item"),
	)
	require.NoError(t, writer.Close())

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	logOutput := string(content)

	assert.Contains(t, logOutput, "<CONFIG_DIR>")
	assert.Contains(t, logOutput, "<CACHE_DIR>")
	assert.NotContains(t, logOutput, configDir)
	assert.NotContains(t, logOutput, cacheDir)
	assert.Regexp(t, regexp.MustCompile(`time="\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}"`), logOutput)
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "rotation.log")

	writer := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1, // 1 MB for testing
		MaxBackups: 3,
	}

	chunk := append(bytes.Repeat([]byte("x"), 64*1024), '\n')
	for i := 0; i < 20; i++ {
		_, err := writer.Write(chunk)
		assert.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	// Verify rotated files exist
	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 1)
}
