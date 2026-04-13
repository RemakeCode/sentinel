package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"sentinel/backend"
)

// TestMigration_MediaDirectory verifies media directory migration to XDG_DATA_HOME
func TestMigration_MediaDirectory(t *testing.T) {
	tempHome := t.TempDir()
	oldCacheBase := filepath.Join(tempHome, ".cache")
	newDataBase := filepath.Join(tempHome, ".local", "share")

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer func() { os.Setenv("HOME", oldHome) }()

	oldMediaDir := filepath.Join(oldCacheBase, "sentinel", "media")
	require.NoError(t, os.MkdirAll(oldMediaDir, 0755))

	testSound := []byte("fake wav data")
	require.NoError(t, os.WriteFile(filepath.Join(oldMediaDir, "steam-deck.wav"), testSound, 0644))

	err := MigrateAll()
	require.NoError(t, err)

	newMediaDir := filepath.Join(newDataBase, "sentinel", "media")
	migratedSound, err := os.ReadFile(filepath.Join(newMediaDir, "steam-deck.wav"))
	require.NoError(t, err)
	require.Equal(t, testSound, migratedSound)
}

// TestMigration_LogsDirectory verifies logs directory migration to XDG_STATE_HOME
func TestMigration_LogsDirectory(t *testing.T) {
	tempHome := t.TempDir()
	oldCacheBase := filepath.Join(tempHome, ".cache")
	newStateBase := filepath.Join(tempHome, ".local", "state")

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer func() { os.Setenv("HOME", oldHome) }()

	oldLogsDir := filepath.Join(oldCacheBase, "sentinel", "logs")
	require.NoError(t, os.MkdirAll(oldLogsDir, 0755))

	testLog := []byte("2026-01-01 Starting sentinel\n")
	require.NoError(t, os.WriteFile(filepath.Join(oldLogsDir, "sentinel.log"), testLog, 0644))

	err := MigrateAll()
	require.NoError(t, err)

	newLogsDir := filepath.Join(newStateBase, "sentinel", "logs")
	migratedLog, err := os.ReadFile(filepath.Join(newLogsDir, "sentinel.log"))
	require.NoError(t, err)
	require.Equal(t, testLog, migratedLog)
}

// TestMigration_AchievementData verifies achievement data directory migration to XDG_DATA_HOME
func TestMigration_AchievementData(t *testing.T) {
	tempHome := t.TempDir()
	oldCacheBase := filepath.Join(tempHome, ".cache")
	newDataBase := filepath.Join(tempHome, ".local", "share")

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer func() { os.Setenv("HOME", oldHome) }()

	oldDataDir := filepath.Join(oldCacheBase, "sentinel", "data")
	require.NoError(t, os.MkdirAll(oldDataDir, 0755))

	testAchData := []byte(`{"achievements":[{"id":"ACH_001","unlocked":true}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(oldDataDir, "123456.json"), testAchData, 0644))

	err := MigrateAll()
	require.NoError(t, err)

	newDataDir := filepath.Join(newDataBase, "sentinel", "data")
	migratedAch, err := os.ReadFile(filepath.Join(newDataDir, "123456.json"))
	require.NoError(t, err)
	require.Equal(t, testAchData, migratedAch)
}

// TestMigration_AchievementIcons verifies achievement icons directory migration to XDG_DATA_HOME
func TestMigration_AchievementIcons(t *testing.T) {
	tempHome := t.TempDir()
	oldCacheBase := filepath.Join(tempHome, ".cache")
	newDataBase := filepath.Join(tempHome, ".local", "share")

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer func() { os.Setenv("HOME", oldHome) }()

	oldIconDir := filepath.Join(oldCacheBase, "sentinel", "icon", "123456")
	require.NoError(t, os.MkdirAll(oldIconDir, 0755))

	testIcon := []byte("fake png icon data")
	require.NoError(t, os.WriteFile(filepath.Join(oldIconDir, "icon.png"), testIcon, 0644))

	err := MigrateAll()
	require.NoError(t, err)

	newIconDir := filepath.Join(newDataBase, "sentinel", "icon", "123456")
	migratedIcon, err := os.ReadFile(filepath.Join(newIconDir, "icon.png"))
	require.NoError(t, err)
	require.Equal(t, testIcon, migratedIcon)
}

// TestMigration_GameMetadata verifies games directory migration to XDG_DATA_HOME
func TestMigration_GameMetadata(t *testing.T) {
	tempHome := t.TempDir()
	oldCacheBase := filepath.Join(tempHome, ".cache")
	newDataBase := filepath.Join(tempHome, ".local", "share")

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer func() { os.Setenv("HOME", oldHome) }()

	oldGamesDir := filepath.Join(oldCacheBase, "sentinel", "games", "english")
	require.NoError(t, os.MkdirAll(oldGamesDir, 0755))

	testMeta := []byte(`{"appName":"Test Game","achievements":[]}`)
	require.NoError(t, os.WriteFile(filepath.Join(oldGamesDir, "123456.json"), testMeta, 0644))

	err := MigrateAll()
	require.NoError(t, err)

	newGamesDir := filepath.Join(newDataBase, "sentinel", "games", "english")
	migratedMeta, err := os.ReadFile(filepath.Join(newGamesDir, "123456.json"))
	require.NoError(t, err)
	require.Equal(t, testMeta, migratedMeta)
}

// TestVerification_XDGPaths verifies XDG paths are correctly set on Linux
func TestVerification_XDGPaths(t *testing.T) {
	require.Contains(t, backend.ConfigDir, ".config/sentinel")
	require.Contains(t, backend.DataDir, ".local/share/sentinel")
	require.Contains(t, backend.StateDir, ".local/state/sentinel")
}

// TestVerification_Subdirectories verifies subdirectory paths are correctly formed
func TestVerification_Subdirectories(t *testing.T) {
	require.Equal(t, filepath.Join(backend.DataDir, "media"), backend.MediaDir)
	require.Equal(t, filepath.Join(backend.DataDir, "data"), backend.ACHCacheDataDir)
	require.Equal(t, filepath.Join(backend.DataDir, "icon"), backend.ACHCacheIconDir)
	require.Equal(t, filepath.Join(backend.DataDir, "games"), backend.GameCacheDir)
	require.Equal(t, filepath.Join(backend.StateDir, "logs"), backend.LogDir)
	require.Equal(t, filepath.Join(backend.LogDir, "sentinel.log"), backend.LogFilePath)
}
