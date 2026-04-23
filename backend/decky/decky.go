package decky

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func GetPort() int {
	if port := os.Getenv("DECKY_PORT"); port != "" {
		p, _ := strconv.Atoi(port)
		return p
	}
	return 48211 // default deployment port
}

func IsSteamOS() bool {
	return os.Getenv("STEAMOS") == "1"
}

func IsSteamInBPM() bool {
	cmd := exec.Command("/bin/cat", "/proc/self/cmdline")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, arg := range filepath.SplitList(string(output)) {
		if arg == "-gamepadui" {
			return true
		}
	}
	return false
}

func IsDeckyInstalled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	homebrewPath := filepath.Join(homeDir, "homebrew")
	_, err = os.Stat(homebrewPath)
	return err == nil
}
