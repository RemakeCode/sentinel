package autostart

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sentinel/backend"
)

//go:embed sentinel-autostart.desktop
var autostartDesktopFile string

const autostartFileName = "sentinel-autostart.desktop"

func GetAutostartPath() string {
	return filepath.Join(backend.UserConfigDir, "autostart", autostartFileName)
}

func SetAutostartEnabled(enabled bool) error {
	path := GetAutostartPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create autostart directory: %w", err)
	}

	if !enabled {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove autostart file: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(path, []byte(autostartDesktopFile), 0644); err != nil {
		return fmt.Errorf("failed to write autostart file: %w", err)
	}

	return nil
}
