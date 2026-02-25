package backend

import (
	"os"
	"path/filepath"
)

var UserHomeDir, _ = os.UserHomeDir()
var UserCacheDir, _ = os.UserCacheDir()

var ConfigDir = filepath.Join(UserCacheDir, "sentinel")
var ConfigPath = filepath.Join(ConfigDir, "config.json")

// Cache directory paths
var ACHCacheDir = filepath.Join(ConfigDir, "cache")
var ACHCacheDataDir = filepath.Join(ACHCacheDir, "data")
var ACHCacheIconDir = filepath.Join(ACHCacheDir, "icon")
var GameCacheDir = filepath.Join(ACHCacheDir, "games")
