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
	require.NoError(t, os.Mkdir(filepath.Join(dir, steamSettingsDirectoryName), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, steamSettingsDirectoryName, "old.txt"), []byte("old"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.bak"), []byte("keep"), 0644))
	stagedDLL := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("new-dll"), 0644))
	generated := filepath.Join(t.TempDir(), steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(generated, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(generated, "new.txt"), []byte("new"), 0644))
	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)

	require.NoError(t, InstallGBESetup(target, stagedDLL, generated))
	require.FileExists(t, targetPath+sentinelBackupSuffix)
	require.DirExists(t, filepath.Join(dir, steamSettingsDirectoryName+sentinelBackupSuffix))
	require.NoFileExists(t, targetPath+sentinelTemporaryBackupSuffix)
	require.NoDirExists(t, filepath.Join(dir, steamSettingsDirectoryName+sentinelTemporaryBackupSuffix))
	restored, err := RestoreSentinelBackups(dir)
	require.NoError(t, err)
	require.True(t, restored)
	require.FileExists(t, filepath.Join(dir, "unrelated.bak"))
	require.FileExists(t, filepath.Join(dir, steamSettingsDirectoryName, "old.txt"))
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

func TestRegenerationPreservesExistingBackups(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetPath, []byte("current"), 0644))
	require.NoError(t, os.WriteFile(targetPath+sentinelBackupSuffix, []byte("historical"), 0644))
	settingsPath := filepath.Join(dir, steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(settingsPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsPath, "current.txt"), []byte("current"), 0644))
	settingsBackup := filepath.Join(dir, steamSettingsDirectoryName+sentinelBackupSuffix)
	require.NoError(t, os.Mkdir(settingsBackup, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsBackup, "historical.txt"), []byte("historical"), 0644))
	staged := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))
	generated := filepath.Join(t.TempDir(), steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(generated, 0755))
	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)
	require.NoError(t, InstallGBESetup(target, staged, generated))
	dllBackup, err := os.ReadFile(targetPath + sentinelBackupSuffix)
	require.NoError(t, err)
	require.Equal(t, "historical", string(dllBackup))
	require.FileExists(t, filepath.Join(settingsBackup, "historical.txt"))
	require.NoFileExists(t, targetPath+sentinelTemporaryBackupSuffix)
	require.NoDirExists(t, filepath.Join(dir, steamSettingsDirectoryName+sentinelTemporaryBackupSuffix))
}

func TestFailedRegenerationRestoresCurrentStateAndPreservesOriginalBackups(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetPath, []byte("current-dll"), 0644))
	require.NoError(t, os.WriteFile(targetPath+sentinelBackupSuffix, []byte("original-dll"), 0644))

	settingsPath := filepath.Join(directory, steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(settingsPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsPath, "current.txt"), []byte("current-settings"), 0644))
	settingsBackup := filepath.Join(directory, steamSettingsDirectoryName+sentinelBackupSuffix)
	require.NoError(t, os.Mkdir(settingsBackup, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsBackup, "original.txt"), []byte("original-settings"), 0644))

	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)
	stagedDLL := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("next-dll"), 0644))

	err = InstallGBESetup(target, stagedDLL, filepath.Join(t.TempDir(), "missing-steam_settings"))
	require.ErrorContains(t, err, "install generated steam_settings")
	require.Equal(t, []byte("current-dll"), mustReadFile(t, targetPath))
	require.Equal(t, []byte("current-settings"), mustReadFile(t, filepath.Join(settingsPath, "current.txt")))
	require.Equal(t, []byte("original-dll"), mustReadFile(t, targetPath+sentinelBackupSuffix))
	require.Equal(t, []byte("original-settings"), mustReadFile(t, filepath.Join(settingsBackup, "original.txt")))
	require.NoFileExists(t, targetPath+sentinelTemporaryBackupSuffix)
	require.NoDirExists(t, filepath.Join(directory, steamSettingsDirectoryName+sentinelTemporaryBackupSuffix))
}

func TestRegenerationRemovesStaleTemporaryBackups(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetPath, []byte("current-dll"), 0644))
	require.NoError(t, os.WriteFile(targetPath+sentinelBackupSuffix, []byte("original-dll"), 0644))
	dllTemporaryBackup := targetPath + sentinelTemporaryBackupSuffix
	require.NoError(t, os.WriteFile(dllTemporaryBackup, []byte("stale-dll"), 0600))

	settingsPath := filepath.Join(directory, steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(settingsPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsPath, "current.txt"), []byte("current-settings"), 0644))
	settingsBackup := filepath.Join(directory, steamSettingsDirectoryName+sentinelBackupSuffix)
	require.NoError(t, os.Mkdir(settingsBackup, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(settingsBackup, "original.txt"), []byte("original-settings"), 0644))
	settingsTemporaryBackup := filepath.Join(directory, steamSettingsDirectoryName+sentinelTemporaryBackupSuffix)
	require.NoError(t, os.Mkdir(settingsTemporaryBackup, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(settingsTemporaryBackup, "stale.txt"), []byte("stale-settings"), 0600))

	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)
	stagedDLL := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("next-dll"), 0644))
	generated := filepath.Join(t.TempDir(), steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(generated, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(generated, "achievements.json"), []byte("next-settings"), 0644))

	require.NoError(t, InstallGBESetup(target, stagedDLL, generated))
	require.Equal(t, []byte("next-dll"), mustReadFile(t, targetPath))
	require.Equal(t, []byte("next-settings"), mustReadFile(t, filepath.Join(settingsPath, "achievements.json")))
	require.Equal(t, []byte("original-dll"), mustReadFile(t, targetPath+sentinelBackupSuffix))
	require.Equal(t, []byte("original-settings"), mustReadFile(t, filepath.Join(settingsBackup, "original.txt")))
	require.NoFileExists(t, dllTemporaryBackup)
	require.NoDirExists(t, settingsTemporaryBackup)
}

func TestExistingColdClientTargetUsesSteamClient(t *testing.T) {
	dir := t.TempDir()
	selectedDLL := filepath.Join(dir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(selectedDLL, []byte("steam-api"), 0644))

	coldClientDirectory := filepath.Join(dir, "coldclient")
	require.NoError(t, os.Mkdir(coldClientDirectory, 0755))
	steamClient := filepath.Join(coldClientDirectory, "steamclient64.dll")
	require.NoError(t, os.WriteFile(steamClient, []byte("original-steamclient"), 0644))
	target, err := ValidateInstallTarget("620", selectedDLL)
	require.NoError(t, err)
	require.Equal(t, LayoutColdClient, target.Layout)
	require.Equal(t, selectedDLL, target.SelectedSteamAPIDLLPath)
	require.Equal(t, coldClientDirectory, target.InstallDirectory)
	require.Equal(t, steamClient, target.ReplacementDLLPath)

	stagedDLL := filepath.Join(t.TempDir(), "steamclient64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("replacement-steamclient"), 0644))
	generated := filepath.Join(t.TempDir(), steamSettingsDirectoryName)
	require.NoError(t, os.Mkdir(generated, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(generated, "new.txt"), []byte("new"), 0644))

	require.NoError(t, InstallGBESetup(target, stagedDLL, generated))
	require.FileExists(t, filepath.Join(coldClientDirectory, "steamclient64.dll.sentinel.bak"))
	require.Equal(t, []byte("steam-api"), mustReadFile(t, selectedDLL))
	require.Equal(t, []byte("replacement-steamclient"), mustReadFile(t, steamClient))
	require.DirExists(t, filepath.Join(coldClientDirectory, steamSettingsDirectoryName))

	restored, err := RestoreSentinelBackups(coldClientDirectory)
	require.NoError(t, err)
	require.True(t, restored)
	require.Equal(t, []byte("original-steamclient"), mustReadFile(t, steamClient))
}

func TestUnsafeColdClientSteamClientDoesNotFallBackToRegularTarget(t *testing.T) {
	dir := t.TempDir()
	selectedDLL := filepath.Join(dir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(selectedDLL, []byte("steam-api"), 0644))
	coldClientDirectory := filepath.Join(dir, "coldclient")
	require.NoError(t, os.Mkdir(coldClientDirectory, 0755))
	require.NoError(t, os.Symlink(selectedDLL, filepath.Join(coldClientDirectory, "steamclient64.dll")))

	_, err := ValidateInstallTarget("620", selectedDLL)
	require.ErrorContains(t, err, "ColdClient SteamClient DLL is not a direct regular file")
}

func TestRestoreSentinelBackupsRecognizesEveryDLLName(t *testing.T) {
	for _, dllName := range []string{"steam_api64.dll", "steam_api.dll", "steamclient64.dll", "steamclient.dll"} {
		t.Run(dllName, func(t *testing.T) {
			directory := t.TempDir()
			destination := filepath.Join(directory, dllName)
			backup := destination + sentinelBackupSuffix
			require.NoError(t, os.WriteFile(destination, []byte("replacement"), 0644))
			require.NoError(t, os.WriteFile(backup, []byte("original"), 0644))

			restored, err := RestoreSentinelBackups(directory)
			require.NoError(t, err)
			require.True(t, restored)
			require.Equal(t, []byte("original"), mustReadFile(t, destination))
		})
	}
}

func TestRegularTargetRemainsRegularWithoutMatchingColdClientBackend(t *testing.T) {
	directory := t.TempDir()
	selectedDLL := filepath.Join(directory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(selectedDLL, []byte("steam-api"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(directory, "coldclient"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "coldclient", "steamclient.dll"), []byte("x86-only"), 0644))

	target, err := ValidateInstallTarget("620", selectedDLL)
	require.NoError(t, err)
	require.Equal(t, LayoutRegular, target.Layout)
	require.Equal(t, selectedDLL, target.SelectedSteamAPIDLLPath)
	require.Equal(t, selectedDLL, target.ReplacementDLLPath)
}

func TestFailedSettingsInstallationRestoresOriginalFiles(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetPath, []byte("original-dll"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(directory, steamSettingsDirectoryName), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, steamSettingsDirectoryName, "original.txt"), []byte("original-settings"), 0644))

	target, err := ValidateInstallTarget("620", targetPath)
	require.NoError(t, err)
	stagedDLL := filepath.Join(t.TempDir(), "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("replacement-dll"), 0644))

	err = InstallGBESetup(target, stagedDLL, filepath.Join(t.TempDir(), "missing-steam_settings"))
	require.ErrorContains(t, err, "install generated steam_settings")
	require.Equal(t, []byte("original-dll"), mustReadFile(t, targetPath))
	require.Equal(t, []byte("original-settings"), mustReadFile(t, filepath.Join(directory, steamSettingsDirectoryName, "original.txt")))
	require.FileExists(t, targetPath+sentinelBackupSuffix)
	require.DirExists(t, filepath.Join(directory, steamSettingsDirectoryName+sentinelBackupSuffix))
	require.NoFileExists(t, targetPath+sentinelTemporaryBackupSuffix)
	require.NoDirExists(t, filepath.Join(directory, steamSettingsDirectoryName+sentinelTemporaryBackupSuffix))
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestSettingsInstallationBranchesAndUndo(t *testing.T) {
	for _, layout := range []InstallLayout{LayoutRegular, LayoutColdClient} {
		for _, existing := range []bool{false, true} {
			name := string(layout) + "/new"
			if existing {
				name = string(layout) + "/existing"
			}

			t.Run(name, func(t *testing.T) {
				root := t.TempDir()
				selected := filepath.Join(root, "steam_api64.dll")
				writeInstallFixture(t, selected, "original-api")
				if layout == LayoutColdClient {
					writeInstallFixture(t, filepath.Join(root, "coldclient", "steamclient64.dll"), "original-client")
				}

				target, err := ValidateInstallTarget("620", selected)
				require.NoError(t, err)
				require.Equal(t, layout, target.Layout)
				originalDLL := mustReadFile(t, target.ReplacementDLLPath)
				settings := filepath.Join(target.InstallDirectory, steamSettingsDirectoryName)
				backup := settings + sentinelBackupSuffix
				if existing {
					writeInstallFixture(t, filepath.Join(settings, "achievements.json"), "old-achievements")
					writeInstallFixture(t, filepath.Join(settings, "img", "normal.png"), "old-image")
					writeInstallFixture(t, filepath.Join(settings, "img", "other.png"), "other-image")
					writeInstallFixture(t, filepath.Join(settings, "configs.app.ini"), "game-configuration")
					writeInstallFixture(t, filepath.Join(settings, "load_dlls", "required.dll"), "required-dll")
				}

				generated := t.TempDir()
				writeInstallFixture(t, filepath.Join(generated, "achievements.json"), "new-achievements")
				writeInstallFixture(t, filepath.Join(generated, "stats.json"), "new-stats")
				writeInstallFixture(t, filepath.Join(generated, "img", "normal.png"), "new-image")
				writeInstallFixture(t, filepath.Join(generated, "img", "nested", "gray.png"), "gray-image")
				writeInstallFixture(t, filepath.Join(generated, "configs.app.ini"), "generated-configuration")
				staged := filepath.Join(t.TempDir(), "replacement.dll")
				writeInstallFixture(t, staged, "new-dll")

				require.NoError(t, InstallGBESetup(target, staged, generated))
				require.Equal(t, []byte("new-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))
				require.Equal(t, []byte("new-stats"), mustReadFile(t, filepath.Join(settings, "stats.json")))
				require.Equal(t, []byte("new-image"), mustReadFile(t, filepath.Join(settings, "img", "normal.png")))
				require.Equal(t, []byte("gray-image"), mustReadFile(t, filepath.Join(settings, "img", "nested", "gray.png")))
				if existing {
					require.Equal(t, []byte("game-configuration"), mustReadFile(t, filepath.Join(settings, "configs.app.ini")))
					require.Equal(t, []byte("required-dll"), mustReadFile(t, filepath.Join(settings, "load_dlls", "required.dll")))
					require.NoFileExists(t, filepath.Join(settings, "img", "other.png"))
					require.Equal(t, []byte("old-achievements"), mustReadFile(t, filepath.Join(backup, "achievements.json")))
				} else {
					require.Equal(t, []byte("generated-configuration"), mustReadFile(t, filepath.Join(settings, "configs.app.ini")))
					require.NoDirExists(t, backup)
				}

				// A later result without stats or images only updates the present artifact.
				next := t.TempDir()
				writeInstallFixture(t, filepath.Join(next, "achievements.json"), "next-achievements")
				require.NoError(t, InstallGBESetup(target, staged, next))
				require.Equal(t, []byte("next-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))
				require.Equal(t, []byte("new-stats"), mustReadFile(t, filepath.Join(settings, "stats.json")))
				require.Equal(t, []byte("new-image"), mustReadFile(t, filepath.Join(settings, "img", "normal.png")))
				require.Equal(t, originalDLL, mustReadFile(t, target.ReplacementDLLPath+sentinelBackupSuffix))
				if !existing {
					require.NoDirExists(t, backup)
				}

				restored, err := RestoreSentinelBackups(target.InstallDirectory)
				require.NoError(t, err)
				require.True(t, restored)
				require.Equal(t, originalDLL, mustReadFile(t, target.ReplacementDLLPath))
				if existing {
					require.Equal(t, []byte("old-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))
					require.Equal(t, []byte("old-image"), mustReadFile(t, filepath.Join(settings, "img", "normal.png")))
					require.Equal(t, []byte("other-image"), mustReadFile(t, filepath.Join(settings, "img", "other.png")))
					require.NoFileExists(t, filepath.Join(settings, "stats.json"))
					require.NoDirExists(t, filepath.Join(settings, "img", "nested"))
				} else {
					require.NoDirExists(t, settings)
				}

				if layout == LayoutColdClient {
					require.Equal(t, []byte("original-api"), mustReadFile(t, selected))
				}
			})
		}
	}
}

func TestPartialAchievementCopyRollsBack(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		name := "first"
		if repeated {
			name = "repeated"
		}

		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dll := filepath.Join(root, "steam_api64.dll")
			writeInstallFixture(t, dll, "current-dll")
			settings := filepath.Join(root, steamSettingsDirectoryName)
			writeInstallFixture(t, filepath.Join(settings, "achievements.json"), "current-achievements")
			writeInstallFixture(t, filepath.Join(settings, "img", "original.png"), "original-image")
			// This directory prevents the last artifact copy after achievements and images were copied.
			require.NoError(t, os.Mkdir(filepath.Join(settings, "stats.json"), 0755))
			if repeated {
				writeInstallFixture(t, dll+sentinelBackupSuffix, "original-dll")
				writeInstallFixture(t, filepath.Join(settings+sentinelBackupSuffix, "original.txt"), "original-settings")
			}

			generated := t.TempDir()
			writeInstallFixture(t, filepath.Join(generated, "achievements.json"), "new-achievements")
			writeInstallFixture(t, filepath.Join(generated, "img", "new.png"), "new-image")
			writeInstallFixture(t, filepath.Join(generated, "stats.json"), "new-stats")
			staged := filepath.Join(t.TempDir(), "replacement.dll")
			writeInstallFixture(t, staged, "new-dll")
			target, err := ValidateInstallTarget("620", dll)
			require.NoError(t, err)

			err = InstallGBESetup(target, staged, generated)
			require.ErrorContains(t, err, "install generated steam_settings")
			require.Equal(t, []byte("current-dll"), mustReadFile(t, dll))
			require.Equal(t, []byte("current-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))
			require.Equal(t, []byte("original-image"), mustReadFile(t, filepath.Join(settings, "img", "original.png")))
			require.NoFileExists(t, filepath.Join(settings, "img", "new.png"))
			require.DirExists(t, filepath.Join(settings, "stats.json"))
			if repeated {
				require.Equal(t, []byte("original-dll"), mustReadFile(t, dll+sentinelBackupSuffix))
				require.Equal(t, []byte("original-settings"), mustReadFile(t, filepath.Join(settings+sentinelBackupSuffix, "original.txt")))
			}

			require.NoDirExists(t, settings+sentinelTemporaryBackupSuffix)
			require.NoFileExists(t, dll+sentinelTemporaryBackupSuffix)
		})
	}
}

func TestFailedInitialSettingsBackupCanRetryAndUndo(t *testing.T) {
	root := t.TempDir()
	dll := filepath.Join(root, "steam_api64.dll")
	writeInstallFixture(t, dll, "original-dll")
	settings := filepath.Join(root, steamSettingsDirectoryName)
	writeInstallFixture(t, filepath.Join(settings, "achievements.json"), "original-achievements")
	// An unreadable file fails the copy after the earlier regular file.
	invalid := filepath.Join(settings, "unreadable.json")
	writeInstallFixture(t, invalid, "settings")
	require.NoError(t, os.Chmod(invalid, 0000))
	t.Cleanup(func() { _ = os.Chmod(invalid, 0644) })
	if _, err := os.ReadFile(invalid); err == nil {
		t.Skip("requires unreadable file permissions to be enforced")
	}
	staged := filepath.Join(t.TempDir(), "replacement.dll")
	writeInstallFixture(t, staged, "new-dll")
	generated := t.TempDir()
	writeInstallFixture(t, filepath.Join(generated, "achievements.json"), "new-achievements")
	target, err := ValidateInstallTarget("620", dll)
	require.NoError(t, err)

	err = InstallGBESetup(target, staged, generated)
	require.ErrorContains(t, err, "backup original steam_settings")
	require.NoFileExists(t, dll+sentinelBackupSuffix)
	require.NoDirExists(t, settings+sentinelBackupSuffix)
	require.Equal(t, []byte("original-dll"), mustReadFile(t, dll))
	require.Equal(t, []byte("original-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))

	require.NoError(t, os.Remove(invalid))
	require.NoError(t, InstallGBESetup(target, staged, generated))
	require.Equal(t, []byte("new-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))
	restored, err := RestoreSentinelBackups(root)
	require.NoError(t, err)
	require.True(t, restored)
	require.Equal(t, []byte("original-dll"), mustReadFile(t, dll))
	require.Equal(t, []byte("original-achievements"), mustReadFile(t, filepath.Join(settings, "achievements.json")))
}

func writeInstallFixture(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}
