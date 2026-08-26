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

const (
	BitnessX64 DLLBitness = "x64"
	BitnessX86 DLLBitness = "x86"
)

type InstallTarget struct {
	AppID   string
	DLLPath string
	Bitness DLLBitness
}

type BackupStatus struct {
	HasDLLBackup      bool `json:"hasDllBackup"`
	HasSettingsBackup bool `json:"hasSettingsBackup"`
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

	if info.IsDir() {
		return InstallTarget{}, errors.New("the selected path is a directory")
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return InstallTarget{}, errors.New("a direct DLL file must be selected, not a symbolic link")
	}

	if info.Mode().Perm()&0444 == 0 {
		return InstallTarget{}, errors.New("the selected DLL is not readable")
	}

	file, err := os.Open(dllPath)
	if err != nil {
		return InstallTarget{}, errors.New("the selected DLL is not readable")
	}
	_ = file.Close()

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

	return InstallTarget{AppID: appID, DLLPath: absoluteDLLPath, Bitness: bitness}, nil
}

func InspectTargetBackups(target InstallTarget) BackupStatus {
	installDirectory := filepath.Dir(target.DLLPath)
	dllName := filepath.Base(target.DLLPath)

	return BackupStatus{
		HasDLLBackup:      fileExists(filepath.Join(installDirectory, dllName+".sentinel.bak")),
		HasSettingsBackup: dirExists(filepath.Join(installDirectory, "steam_settings.sentinel.bak")),
	}
}

func InstallGBESetup(target InstallTarget, stagedDLL, generatedOutput string) error {
	installDirectory := filepath.Dir(target.DLLPath)
	dllName := filepath.Base(target.DLLPath)
	dllBackup := filepath.Join(installDirectory, dllName+".sentinel.bak")
	settingsPath := filepath.Join(installDirectory, "steam_settings")
	settingsBackup := filepath.Join(installDirectory, "steam_settings.sentinel.bak")
	createdDLLBackup := false
	createdSettingsBackup := false
	transactionDir, err := os.MkdirTemp(installDirectory, ".sentinel-install-")
	if err != nil {
		return fmt.Errorf("create installation transaction: %w", err)
	}
	defer os.RemoveAll(transactionDir)

	dllSnapshot := filepath.Join(transactionDir, dllName)
	settingsSnapshot := filepath.Join(transactionDir, "steam_settings")
	if err := copyFile(target.DLLPath, dllSnapshot, 0600); err != nil {
		return fmt.Errorf("snapshot current DLL: %w", err)
	}

	hadSettings := dirExists(settingsPath)
	if hadSettings {
		if err := copyDirectory(settingsPath, settingsSnapshot); err != nil {
			return fmt.Errorf("snapshot current steam_settings: %w", err)
		}
	}

	if !fileExists(dllBackup) {
		if err := copyFile(target.DLLPath, dllBackup, 0644); err != nil {
			return fmt.Errorf("backup original DLL: %w", err)
		}
		createdDLLBackup = true
	}
	if dirExists(settingsPath) && !dirExists(settingsBackup) {
		if err := copyDirectory(settingsPath, settingsBackup); err != nil {
			if createdDLLBackup {
				_ = os.Remove(dllBackup)
			}
			_ = os.RemoveAll(settingsBackup)
			return fmt.Errorf("backup existing steam_settings: %w", err)
		}
		createdSettingsBackup = true
	}

	if err := copyFile(stagedDLL, target.DLLPath, 0755); err != nil {
		restoreErr := restoreInstallSnapshot(target, dllSnapshot, settingsSnapshot, hadSettings)
		removeCreatedBackups(dllBackup, settingsBackup, createdDLLBackup, createdSettingsBackup)
		if restoreErr != nil {
			return fmt.Errorf("install GBE DLL: %w (restore failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("install GBE DLL: %w", err)
	}

	if err := replaceDirectory(settingsPath, generatedOutput); err != nil {
		if restoreErr := restoreInstallSnapshot(target, dllSnapshot, settingsSnapshot, hadSettings); restoreErr != nil {
			return fmt.Errorf("install generated steam_settings: %w (restore failed: %v)", err, restoreErr)
		}
		removeCreatedBackups(dllBackup, settingsBackup, createdDLLBackup, createdSettingsBackup)
		return fmt.Errorf("install generated steam_settings: %w", err)
	}

	return nil
}

func removeCreatedBackups(dllBackup, settingsBackup string, dllCreated, settingsCreated bool) {
	if dllCreated {
		_ = os.Remove(dllBackup)
	}

	if settingsCreated {
		_ = os.RemoveAll(settingsBackup)
	}
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
	for _, basename := range []string{"steam_api64.dll", "steam_api.dll"} {
		backup := filepath.Join(directory, basename+".sentinel.bak")
		if !fileExists(backup) {
			continue
		}
		if err := restoreFileBackup(filepath.Join(directory, basename), backup); err != nil {
			return restored, err
		}
		restored = true
	}

	settingsBackup := filepath.Join(directory, "steam_settings.sentinel.bak")
	settingsPath := filepath.Join(directory, "steam_settings")
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

func restoreInstallSnapshot(target InstallTarget, dllSnapshot, settingsSnapshot string, hadSettings bool) error {
	if err := copyFile(dllSnapshot, target.DLLPath, 0755); err != nil {
		return fmt.Errorf("restore pre-install DLL: %w", err)
	}

	settingsPath := filepath.Join(filepath.Dir(target.DLLPath), "steam_settings")
	if err := removePath(settingsPath); err != nil {
		return fmt.Errorf("remove partial steam_settings: %w", err)
	}

	if hadSettings {
		if err := copyDirectory(settingsSnapshot, settingsPath); err != nil {
			return fmt.Errorf("restore pre-install steam_settings: %w", err)
		}
	}

	return nil
}

func replaceDirectory(destination, source string) error {
	if err := removePath(destination); err != nil {
		return err
	}

	return copyDirectory(source, destination)
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
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
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
