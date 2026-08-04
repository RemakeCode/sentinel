//go:build !decky

package autostart

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v3/pkg/application"

	"sentinel/backend"
	"sentinel/backend/config"
)

func setupAutostartTest(t *testing.T) *config.File {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	originalPath := backend.ConfigPath
	originalConfigDir := backend.ConfigDir
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	backend.ConfigPath = configPath
	backend.ConfigDir = filepath.Join(tempDir, "config")
	backend.DataDir = filepath.Join(tempDir, "data")
	backend.GameCacheDir = filepath.Join(backend.DataDir, "games")
	t.Cleanup(func() {
		backend.ConfigPath = originalPath
		backend.ConfigDir = originalConfigDir
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
	})

	return &config.File{}
}

func TestServiceSetEnabledPersistsPreference(t *testing.T) {
	cfg := setupAutostartTest(t)
	service := NewService(cfg)

	require.NoError(t, service.SetEnabled(true))
	assert.True(t, cfg.StartOnLogin)

	loaded := &config.File{}
	_, err := loaded.LoadConfig()
	require.NoError(t, err)
	assert.True(t, loaded.StartOnLogin)
}

func TestServiceStartupUsesHydratedEnabledPreference(t *testing.T) {
	cfg := setupAutostartTest(t)
	require.NoError(t, os.WriteFile(backend.ConfigPath, []byte(`{"startOnLogin":true}`), 0644))
	require.NoError(t, cfg.Start(context.Background()))

	service := NewService(cfg)
	require.NoError(t, service.ServiceStartup(context.Background(), application.ServiceOptions{}))
	assert.True(t, cfg.GetStartOnLogin())
}

func TestServiceStartupUsesHydratedDisabledPreference(t *testing.T) {
	cfg := setupAutostartTest(t)
	require.NoError(t, os.WriteFile(backend.ConfigPath, []byte(`{"startOnLogin":false}`), 0644))
	require.NoError(t, cfg.Start(context.Background()))

	service := NewService(cfg)
	require.NoError(t, service.ServiceStartup(context.Background(), application.ServiceOptions{}))
	assert.False(t, cfg.GetStartOnLogin())
}
