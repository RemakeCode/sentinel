package generator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DLLBitness string
type InstallLayout string

const (
	BitnessX64 DLLBitness = "x64"
	BitnessX86 DLLBitness = "x86"

	LayoutRegular    InstallLayout = "regular"
	LayoutColdClient InstallLayout = "coldclient"
)

type InstallTarget struct {
	AppID                   string
	SelectedSteamAPIDLLPath string
	Bitness                 DLLBitness
	Layout                  InstallLayout
	InstallDirectory        string
	ReplacementDLLPath      string
}

func ValidateInstallTarget(appID, dllPath string) (InstallTarget, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" || strings.Trim(appID, "0123456789") != "" {
		return InstallTarget{}, errors.New("a valid Steam app ID is required")
	}

	info, err := os.Lstat(dllPath)
	if err != nil {
		return InstallTarget{}, fmt.Errorf("selected DLL is unavailable: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return InstallTarget{}, errors.New("a direct DLL file must be selected, not a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return InstallTarget{}, errors.New("the selected path is not a regular file")
	}

	if info.Mode().Perm()&0444 == 0 {
		return InstallTarget{}, errors.New("the selected DLL is not readable")
	}

	basename := filepath.Base(dllPath)
	var bitness DLLBitness
	switch basename {
	case "steam_api64.dll":
		bitness = BitnessX64
	case "steam_api.dll":
		bitness = BitnessX86
	default:
		return InstallTarget{}, errors.New("selected file is not a supported Steam API DLL")
	}

	absoluteDLLPath, err := filepath.Abs(dllPath)
	if err != nil {
		return InstallTarget{}, fmt.Errorf("resolve selected DLL path: %w", err)
	}

	target := InstallTarget{
		AppID:                   appID,
		SelectedSteamAPIDLLPath: absoluteDLLPath,
		Bitness:                 bitness,
		Layout:                  LayoutRegular,
		InstallDirectory:        filepath.Dir(absoluteDLLPath),
		ReplacementDLLPath:      absoluteDLLPath,
	}

	return resolveColdClientTarget(target)
}

func resolveColdClientTarget(target InstallTarget) (InstallTarget, error) {
	coldClientDirectory := filepath.Join(filepath.Dir(target.SelectedSteamAPIDLLPath), "coldclient")
	coldClientInfo, err := os.Lstat(coldClientDirectory)
	if os.IsNotExist(err) {
		return target, nil
	}
	if err != nil {
		return InstallTarget{}, fmt.Errorf("inspect ColdClient directory: %w", err)
	}
	if coldClientInfo.Mode()&os.ModeSymlink != 0 || !coldClientInfo.IsDir() {
		return InstallTarget{}, errors.New("ColdClient directory is not a direct directory")
	}

	steamClientName := coldClientDLLName(target.Bitness)
	steamClientPath := filepath.Join(coldClientDirectory, steamClientName)
	steamClientInfo, err := os.Lstat(steamClientPath)
	if os.IsNotExist(err) {
		return target, nil
	}
	if err != nil {
		return InstallTarget{}, fmt.Errorf("inspect ColdClient SteamClient DLL: %w", err)
	}
	if steamClientInfo.Mode()&os.ModeSymlink != 0 || !steamClientInfo.Mode().IsRegular() {
		return InstallTarget{}, errors.New("ColdClient SteamClient DLL is not a direct regular file")
	}
	if steamClientInfo.Mode().Perm()&0444 == 0 {
		return InstallTarget{}, errors.New("ColdClient SteamClient DLL is not readable")
	}

	target.Layout = LayoutColdClient
	target.InstallDirectory = coldClientDirectory
	target.ReplacementDLLPath = steamClientPath
	return target, nil
}

func coldClientDLLName(bitness DLLBitness) string {
	if bitness == BitnessX86 {
		return "steamclient.dll"
	}
	return "steamclient64.dll"
}

func InstallGBESetup(target InstallTarget, stagedDLL, generatedOutput string) error {
	installDirectory := target.InstallDirectory
	dllName := filepath.Base(target.ReplacementDLLPath)
	dllBackup := filepath.Join(installDirectory, dllName+sentinelBackupSuffix)
	dllTemporaryBackup := filepath.Join(installDirectory, dllName+sentinelTemporaryBackupSuffix)
	settingsPath := filepath.Join(installDirectory, steamSettingsDirectoryName)
	settingsBackup := filepath.Join(installDirectory, steamSettingsDirectoryName+sentinelBackupSuffix)
	settingsTemporaryBackup := filepath.Join(installDirectory, steamSettingsDirectoryName+sentinelTemporaryBackupSuffix)
	if err := removePath(dllTemporaryBackup); err != nil {
		return fmt.Errorf("remove stale temporary DLL backup: %w", err)
	}
	if err := removePath(settingsTemporaryBackup); err != nil {
		return fmt.Errorf("remove stale temporary steam_settings backup: %w", err)
	}

	retainTemporaryBackups := false
	defer func() {
		if !retainTemporaryBackups {
			_ = os.Remove(dllTemporaryBackup)
			_ = os.RemoveAll(settingsTemporaryBackup)
		}
	}()

	dllRollbackSource := dllBackup
	if fileExists(dllBackup) {
		if err := copyFile(target.ReplacementDLLPath, dllTemporaryBackup, 0600); err != nil {
			return fmt.Errorf("snapshot current DLL for redo: %w", err)
		}
		dllRollbackSource = dllTemporaryBackup
	} else {
		if err := copyFile(target.ReplacementDLLPath, dllBackup, 0644); err != nil {
			_ = os.Remove(dllBackup)
			return fmt.Errorf("backup original DLL: %w", err)
		}
	}

	hadSettings := dirExists(settingsPath)
	settingsRollbackSource := ""
	if hadSettings {
		settingsRollbackSource = settingsBackup
		if dirExists(settingsBackup) || dllRollbackSource == dllTemporaryBackup {
			if err := copyDirectory(settingsPath, settingsTemporaryBackup); err != nil {
				return fmt.Errorf("snapshot current steam_settings for redo: %w", err)
			}
			settingsRollbackSource = settingsTemporaryBackup
		} else if err := copyDirectory(settingsPath, settingsBackup); err != nil {
			// Both backups belong to this first attempt. Leaving the DLL backup
			// behind would make a retry treat it as a previous installation.
			if cleanupErr := os.Remove(dllBackup); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("remove new DLL backup: %w", cleanupErr))
			}

			if cleanupErr := os.RemoveAll(settingsBackup); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("remove incomplete steam_settings backup: %w", cleanupErr))
			}

			return fmt.Errorf("backup original steam_settings: %w", err)
		}
	}

	if err := copyFile(stagedDLL, target.ReplacementDLLPath, 0755); err != nil {
		restoreErr := restoreInstallState(target, dllRollbackSource, settingsRollbackSource, hadSettings)
		if restoreErr != nil {
			retainTemporaryBackups = true
			return fmt.Errorf("install GBE DLL: %w (restore failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("install GBE DLL: %w", err)
	}

	if err := installSettings(settingsPath, generatedOutput, hadSettings); err != nil {
		if restoreErr := restoreInstallState(target, dllRollbackSource, settingsRollbackSource, hadSettings); restoreErr != nil {
			retainTemporaryBackups = true
			return fmt.Errorf("install generated steam_settings: %w (restore failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("install generated steam_settings: %w", err)
	}

	return nil
}

func RestoreSentinelBackups(directory string) (bool, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return false, err
	}

	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		return false, nil
	}
	restored := false
	for _, basename := range []string{"steam_api64.dll", "steam_api.dll", "steamclient64.dll", "steamclient.dll"} {
		backup := filepath.Join(directory, basename+sentinelBackupSuffix)
		if !fileExists(backup) {
			continue
		}
		if err := restoreFileBackup(filepath.Join(directory, basename), backup); err != nil {
			return restored, err
		}
		restored = true
	}

	settingsBackup := filepath.Join(directory, steamSettingsDirectoryName+sentinelBackupSuffix)
	settingsPath := filepath.Join(directory, steamSettingsDirectoryName)
	if dirExists(settingsBackup) {
		if err := removePath(settingsPath); err != nil {
			return restored, err
		}
		if err := os.Rename(settingsBackup, settingsPath); err != nil {
			return restored, fmt.Errorf("restore steam_settings backup: %w", err)
		}
		restored = true
	} else if restored {
		if err := removePath(settingsPath); err != nil {
			return restored, err
		}
	}

	return restored, nil
}

func restoreInstallState(target InstallTarget, dllRollbackSource, settingsRollbackSource string, hadSettings bool) error {
	if err := copyFile(dllRollbackSource, target.ReplacementDLLPath, 0755); err != nil {
		return fmt.Errorf("restore pre-install DLL: %w", err)
	}

	settingsPath := filepath.Join(target.InstallDirectory, steamSettingsDirectoryName)
	if err := removePath(settingsPath); err != nil {
		return fmt.Errorf("remove partial steam_settings: %w", err)
	}

	if hadSettings {
		if err := copyDirectory(settingsRollbackSource, settingsPath); err != nil {
			return fmt.Errorf("restore pre-install steam_settings: %w", err)
		}
	}

	return nil
}

func installSettings(destination, source string, existing bool) error {
	if !existing {
		return copyDirectory(source, destination)
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		switch entry.Name() {
		case "achievements.json", "stats.json":
			if err := copyFile(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name()), 0644); err != nil {
				return err
			}
		case "img":
			imagesPath := filepath.Join(destination, "img")
			if err := removePath(imagesPath); err != nil {
				return err
			}

			if err := copyDirectory(filepath.Join(source, "img"), imagesPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func restoreFileBackup(destination, backup string) error {
	if err := removePath(destination); err != nil {
		return err
	}
	return os.Rename(backup, destination)
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func copyDirectory(source, destination string) error {
	return os.CopyFS(destination, os.DirFS(source))
}

func removePath(path string) error {
	return os.RemoveAll(path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
