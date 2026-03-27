package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
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
