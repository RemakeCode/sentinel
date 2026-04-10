package migrate

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
)

var (
	oldCacheDir  = filepath.Join(getUserCacheDir(), backend.AppName)
	newConfigDir = backend.ConfigDir
	newDataDir   = backend.DataDir
	newStateDir  = backend.StateDir
)

func getUserCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache")
}

func MigrateAll() error {
	slog.Info("Starting data migration")

	migrated := false

	if migrateConfig() {
		migrated = true
	}

	if migrateMedia() {
		migrated = true
	}

	if migrateLogs() {
		migrated = true
	}

	if migrateAchievementData() {
		migrated = true
	}

	if migrateAchievementIcons() {
		migrated = true
	}

	if migrateGameMetadata() {
		migrated = true
	}

	if migrated {
		slog.Info("Data migration completed successfully")
	} else {
		slog.Info("No data to migrate, using new locations")
	}

	return nil
}

func detectOldData() bool {
	oldConfigPath := filepath.Join(oldCacheDir, "config.json")
	if _, err := os.Stat(oldConfigPath); err == nil {
		return true
	}

	oldDirs := []string{
		filepath.Join(oldCacheDir, "media"),
		filepath.Join(oldCacheDir, "logs"),
		filepath.Join(oldCacheDir, "cache", "data"),
		filepath.Join(oldCacheDir, "cache", "icon"),
		filepath.Join(oldCacheDir, "cache", "games"),
	}

	for _, dir := range oldDirs {
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}

	return false
}

func detectNewData() bool {
	newConfigPath := filepath.Join(newConfigDir, "config.json")
	if _, err := os.Stat(newConfigPath); err == nil {
		return true
	}
	return false
}

func migrateConfig() bool {
	oldPath := filepath.Join(oldCacheDir, "config.json")
	newPath := filepath.Join(newConfigDir, "config.json")

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(newPath); err == nil {
		slog.Info("Config already exists in new location, skipping migration")
		return false
	}

	slog.Info("Migrating config file", "from", oldPath, "to", newPath)

	if err := os.MkdirAll(newConfigDir, 0755); err != nil {
		slog.Error("Failed to create config directory", "error", err)
		return false
	}

	if err := copyFile(oldPath, newPath); err != nil {
		slog.Error("Failed to migrate config", "error", err)
		return false
	}

	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		slog.Warn("Failed to backup old config", "error", err)
	}

	slog.Info("Config migrated successfully")
	return true
}

func migrateMedia() bool {
	oldPath := filepath.Join(oldCacheDir, "media")
	newPath := filepath.Join(newDataDir, "media")

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(newPath); err == nil {
		slog.Info("Media already exists in new location, skipping migration")
		return false
	}

	slog.Info("Migrating media directory", "from", oldPath, "to", newPath)

	if err := copyDir(oldPath, newPath); err != nil {
		slog.Error("Failed to migrate media", "error", err)
		return false
	}

	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		slog.Warn("Failed to backup old media directory", "error", err)
	}

	slog.Info("Media migrated successfully")
	return true
}

func migrateLogs() bool {
	oldPath := filepath.Join(oldCacheDir, "logs")
	newPath := filepath.Join(newStateDir, "logs")

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(newPath); err == nil {
		slog.Info("Logs already exist in new location, skipping migration")
		return false
	}

	slog.Info("Migrating logs directory", "from", oldPath, "to", newPath)

	if err := copyDir(oldPath, newPath); err != nil {
		slog.Error("Failed to migrate logs", "error", err)
		return false
	}

	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		slog.Warn("Failed to backup old logs directory", "error", err)
	}

	slog.Info("Logs migrated successfully")
	return true
}

func migrateAchievementData() bool {
	oldPath := filepath.Join(oldCacheDir, "cache", "data")
	newPath := filepath.Join(newDataDir, "data")

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(newPath); err == nil {
		slog.Info("Achievement data already exists in new location, skipping migration")
		return false
	}

	slog.Info("Migrating achievement data", "from", oldPath, "to", newPath)

	if err := copyDir(oldPath, newPath); err != nil {
		slog.Error("Failed to migrate achievement data", "error", err)
		return false
	}

	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		slog.Warn("Failed to backup old achievement data", "error", err)
	}

	slog.Info("Achievement data migrated successfully")
	return true
}

func migrateAchievementIcons() bool {
	oldPath := filepath.Join(oldCacheDir, "cache", "icon")
	newPath := filepath.Join(newDataDir, "icon")

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(newPath); err == nil {
		slog.Info("Achievement icons already exist in new location, skipping migration")
		return false
	}

	slog.Info("Migrating achievement icons", "from", oldPath, "to", newPath)

	if err := copyDir(oldPath, newPath); err != nil {
		slog.Error("Failed to migrate achievement icons", "error", err)
		return false
	}

	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		slog.Warn("Failed to backup old achievement icons", "error", err)
	}

	slog.Info("Achievement icons migrated successfully")
	return true
}

func migrateGameMetadata() bool {
	oldPath := filepath.Join(oldCacheDir, "cache", "games")
	newPath := filepath.Join(newDataDir, "games")

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(newPath); err == nil {
		slog.Info("Game metadata already exists in new location, skipping migration")
		return false
	}

	slog.Info("Migrating game metadata", "from", oldPath, "to", newPath)

	if err := copyDir(oldPath, newPath); err != nil {
		slog.Error("Failed to migrate game metadata", "error", err)
		return false
	}

	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		slog.Warn("Failed to backup old game metadata", "error", err)
	}

	slog.Info("Game metadata migrated successfully")
	return true
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func copyDir(src, dst string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, sourceInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
