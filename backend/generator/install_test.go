package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallAndUndoPreserveExactBackups(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetPath, []byte("original-dll"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "steam_settings"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "steam_settings", "old.txt"), []byte("old"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.bak"), []byte("keep"), 0644))
	stagedDLL := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("new-dll"), 0644))
	generated := filepath.Join(t.TempDir(), "steam_settings")
	require.NoError(t, os.Mkdir(generated, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(generated, "new.txt"), []byte("new"), 0644))
	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)

	require.NoError(t, InstallGBESetup(target, stagedDLL, generated))
	require.FileExists(t, targetPath+".sentinel.bak")
	require.DirExists(t, filepath.Join(dir, "steam_settings.sentinel.bak"))
	restored, err := RestoreSentinelBackups(dir)
	require.NoError(t, err)
	require.True(t, restored)
	require.FileExists(t, filepath.Join(dir, "unrelated.bak"))
	require.FileExists(t, filepath.Join(dir, "steam_settings", "old.txt"))
	data, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, "original-dll", string(data))
}

func TestValidateInstallTargetRejectsInvalidInputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "other.dll")
	require.NoError(t, os.WriteFile(path, []byte("dll"), 0644))
	_, err := ValidateInstallTarget("not-an-id", path)
	require.Error(t, err)
	_, err = ValidateInstallTarget("620", path)
	require.Error(t, err)
	link := filepath.Join(dir, "steam_api64.dll")
	require.NoError(t, os.Symlink(path, link))
	_, err = ValidateInstallTarget("620", link)
	require.Error(t, err)
}

func TestConfirmedRegenerationPreservesExistingBackups(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetPath, []byte("current"), 0644))
	require.NoError(t, os.WriteFile(targetPath+".sentinel.bak", []byte("historical"), 0644))
	settingsBackup := filepath.Join(dir, "steam_settings.sentinel.bak")
	require.NoError(t, os.Mkdir(settingsBackup, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsBackup, "historical.txt"), []byte("historical"), 0644))
	staged := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))
	generated := filepath.Join(t.TempDir(), "steam_settings")
	require.NoError(t, os.Mkdir(generated, 0755))
	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)
	require.NoError(t, InstallGBESetup(target, staged, generated))
	dllBackup, err := os.ReadFile(targetPath + ".sentinel.bak")
	require.NoError(t, err)
	require.Equal(t, "historical", string(dllBackup))
	require.FileExists(t, filepath.Join(settingsBackup, "historical.txt"))
}
