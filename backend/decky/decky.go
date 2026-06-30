package decky

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func GetPort() int {
	if port := os.Getenv("DECKY_PORT"); port != "" {
		p, _ := strconv.Atoi(port)
		return p
	}
	return 48211 // default deployment port
}

func IsSteamInBPM() bool {
	cmd := exec.Command("pgrep", "-x", "steam")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	pids := strings.Fields(string(output))
	for _, pid := range pids {
		cmdlinePath := filepath.Join("/proc", pid, "cmdline")
		data, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}
		args := strings.Split(string(data), "\x00")
		if len(args) > 0 && args[len(args)-1] == "" {
			args = args[:len(args)-1]
		}
		for _, arg := range args {
			if arg == "-gamepadui" {
				return true
			}
		}
	}
	return false
}

func IsGamescopeSession() bool {
	cmd := exec.Command("pgrep", "gamescope")
	return cmd.Run() == nil
}

func IsActiveDeckSession() bool {
	return IsSteamInBPM() || IsGamescopeSession()
}

func SteamInstallPath() string {
	home := os.Getenv("DECKY_USER_HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".local", "share", "Steam")
}

func FindSteamGridPortrait(shortcutAppID string) (string, error) {
	filename := shortcutAppID + "p.png"
	userdataPath := filepath.Join(SteamInstallPath(), "userdata")

	userDirs, err := os.ReadDir(userdataPath)
	if err != nil {
		return "", err
	}

	for _, userDir := range userDirs {
		if !userDir.IsDir() {
			continue
		}

		candidate := filepath.Join(userdataPath, userDir.Name(), "config", "grid", filename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", errors.New("steamgrid portrait not found")
}
