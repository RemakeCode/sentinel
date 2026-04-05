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
	EventFetchStatus = EventPrefix + "::fetch-status"
	EventDataUpdated = EventPrefix + "::data-updated"
)

var EmuDir = filepath.Join("AppData", "Roaming", "GSE Saves")

var UserCacheDir, _ = os.UserCacheDir()

var ConfigDir = filepath.Join(UserCacheDir, AppName)
var ConfigPath = filepath.Join(ConfigDir, "config.json")

// Media directory paths
var MediaDir = filepath.Join(ConfigDir, "media")

// Cache directory paths
var ACHCacheDir = filepath.Join(ConfigDir, "cache")
var ACHCacheDataDir = filepath.Join(ACHCacheDir, "data")
var ACHCacheIconDir = filepath.Join(ACHCacheDir, "icon")
var GameCacheDir = filepath.Join(ACHCacheDir, "games")

var WalkerInterval = 5 * time.Second

// Notification timing
var NotificationExpireTime = 7 * time.Second
var NotificationDelay = NotificationExpireTime + 1*time.Second
