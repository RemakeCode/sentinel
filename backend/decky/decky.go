package decky

import (
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

func IsDeckyInstalled() bool {
	cmd := exec.Command("systemctl", "is-active", "plugin_loader.service")
	output, _ := cmd.Output()
	status := strings.TrimSpace(string(output))
	return status == "active"
}

func IsGamescopeSession() bool {
	cmd := exec.Command("pgrep", "gamescope")
	return cmd.Run() == nil
}

func IsDecky() bool {
	return IsSteamInBPM() && IsDeckyInstalled() || IsGamescopeSession() && IsDeckyInstalled()
}
