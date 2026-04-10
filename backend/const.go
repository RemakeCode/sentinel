package backend

import (
	"os"
	"path/filepath"
	"time"
)

const AppName = "sentinel"

var Version = "0.0.0"

const (
	EventPrefix      = AppName
	EventFetchStatus = AppName + "::fetch-status"
	EventDataUpdated = AppName + "::data-updated"
)

var EmuDir = filepath.Join("AppData", "Roaming", "GSE Saves")

var UserCacheDir, _ = os.UserCacheDir()
var UserConfigDir, _ = os.UserConfigDir()

func getUserDataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func getUserStateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state")
}

// Config directory (XDG_CONFIG_HOME)
var ConfigDir = filepath.Join(UserConfigDir, AppName)
var ConfigPath = filepath.Join(ConfigDir, "config.json")

// Data directory (XDG_DATA_HOME)
var DataDir = filepath.Join(getUserDataDir(), AppName)
var MediaDir = filepath.Join(DataDir, "media")
var ACHCacheDataDir = filepath.Join(DataDir, "data")
var ACHCacheIconDir = filepath.Join(DataDir, "icon")
var GameCacheDir = filepath.Join(DataDir, "games")

// State directory (XDG_STATE_HOME)
var StateDir = filepath.Join(getUserStateDir(), AppName)
var LogDir = filepath.Join(StateDir, "logs")
var LogFilePath = filepath.Join(LogDir, "sentinel.log")

var WalkerInterval = 5 * time.Second

// Notification timing
var NotificationExpireTime = 7 * time.Second
var NotificationDelay = NotificationExpireTime + 1*time.Second
