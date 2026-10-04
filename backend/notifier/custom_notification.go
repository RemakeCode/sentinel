package notifier

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"sentinel/backend"
	"sentinel/backend/config"
)

type customNotificationData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	GameName    string `json:"gameName"`
	IconPath    string `json:"iconPath"`
	Progress    int    `json:"progress"`
	MaxProgress int    `json:"maxProgress"`
	IsProgress  bool   `json:"isProgress"`
	IsRare      bool   `json:"isRare"`
	DurationMS  int64  `json:"durationMs"`
}

var customNotificationBinaries embed.FS
var gnomeExtensionFiles embed.FS

const gnomeExtensionUUID = "io.github.remakecode.sentinel"
const gnomeNotificationInterface = "io.github.remakecode.Sentinel.Notifications"

// SetupCustomNotifications installs the bundled GNOME extension and returns user guidance.
func (s *Service) SetupCustomNotifications() (string, error) {
	if s.deliveryMode != DeliveryDesktop || s.Config.GetDesktopNotificationRenderer() != config.DesktopNotificationRendererCustom || !isGNOMEDesktop() {
		return "", nil
	}

	if err := installGNOMEExtension(); err != nil {
		return "", fmt.Errorf("could not install the Sentinel GNOME extension: %w", err)
	}

	return "The Sentinel GNOME extension is installed. Enable sentinel custom notification in the Extensions app, or run: gnome-extensions enable " + gnomeExtensionUUID + ". If it is not listed, log out and back in first. After extension updates, log out and back in to load the new code. Ordinary enable/disable needs no logout. Once enabled, use Test Notification below.", nil
}

func (s *Service) sendNotificationCustom(payload *NotificationPayload) {
	if isGNOMEDesktop() {
		s.sendNotificationCustomGNOME(payload)
		return
	}

	s.sendNotificationCustomQt(payload)
}

func (s *Service) sendNotificationCustomGNOME(payload *NotificationPayload) {
	data, err := encodeCustomNotificationData(payload)
	if err != nil {
		slog.Warn("Invalid custom notification data", "error", err)
		return
	}

	conn, err := s.getDesktopConnection()
	if err != nil {
		slog.Warn("GNOME custom notification unavailable", "error", err)
		return
	}

	object := conn.Object("org.gnome.Shell", "/io/github/remakecode/Sentinel/Notifications")
	if err := object.CallWithContext(s.ctx, gnomeNotificationInterface+".Show", 0, string(data)).Err; err != nil {
		slog.Warn("GNOME custom notification delivery failed; enable sentinel custom notification in GNOME Extensions", "error", err)
		return
	}

	if payload.SoundFile != "" {
		_ = s.PlaySound(payload.SoundFile)
	}
}

func (s *Service) sendNotificationCustomQt(payload *NotificationPayload) {
	executable, err := customNotificationExecutable()
	if err != nil {
		slog.Warn("Custom notification unavailable", "error", err)
		return
	}
	data, err := encodeCustomNotificationData(payload)
	if err != nil {
		slog.Warn("Invalid custom notification data", "error", err)
		return
	}
	cmd := exec.CommandContext(s.ctx, executable)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		slog.Warn("Custom notification delivery failed", "error", err)
		return
	}

	if payload.SoundFile != "" {
		_ = s.PlaySound(payload.SoundFile)
	}

	go func() {
		if err := cmd.Wait(); err != nil && s.ctx.Err() == nil {
			slog.Warn("Custom notification process exited", "error", err)
		}
	}()
}

func isGNOMEDesktop() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Getenv("XDG_SESSION_TYPE") != "wayland" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return false
	}

	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP") + ":" + os.Getenv("XDG_SESSION_DESKTOP"))
	return strings.Contains(desktop, "gnome") || strings.Contains(desktop, "mutter")
}

func installGNOMEExtension() error {
	// Flatpak's XDG_DATA_HOME is app-private; host installation needs packaging support.
	if os.Getenv("FLATPAK_ID") != "" {
		return errors.New("installing into the host GNOME extension directory from Flatpak is not supported yet")
	}

	directory := os.Getenv("XDG_DATA_HOME")
	if directory == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		directory = filepath.Join(home, ".local", "share")
	}

	directory = filepath.Join(directory, "gnome-shell", "extensions", gnomeExtensionUUID)
	entries, err := gnomeExtensionFiles.ReadDir("out/" + gnomeExtensionUUID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		data, err := gnomeExtensionFiles.ReadFile("out/" + gnomeExtensionUUID + "/" + entry.Name())
		if err != nil {
			return err
		}

		path := filepath.Join(directory, entry.Name())
		if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
			continue
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

func customNotificationExecutable() (string, error) {
	if path := os.Getenv("SENTINEL_CUSTOM_NOTIFICATION_EXECUTABLE"); path != "" {
		if !filepath.IsAbs(path) {
			return "", errors.New("SENTINEL_CUSTOM_NOTIFICATION_EXECUTABLE must be an absolute path")
		}
		return path, nil
	}

	data, err := customNotificationBinaries.ReadFile("out/sentinel-custom-notification-" + runtime.GOOS + "-" + runtime.GOARCH)
	if err != nil {
		return "", err
	}

	directory := filepath.Join(backend.DataDir, "bin")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}

	path := filepath.Join(directory, "sentinel-custom-notification")
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return path, os.Chmod(path, 0700)
	}

	file, err := os.CreateTemp(directory, ".custom-notification-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())

	if _, err := file.Write(data); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Chmod(0700); err != nil {
		file.Close()
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(file.Name(), path); err != nil {
		return "", err
	}

	return path, nil
}

func encodeCustomNotificationData(payload *NotificationPayload) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("missing notification")
	}
	data, err := json.Marshal(customNotificationData{
		Title: payload.Title, Description: payload.Message, GameName: payload.GameName,
		IconPath: payload.IconPath, Progress: payload.Progress, MaxProgress: payload.MaxProgress,
		IsProgress: payload.IsProgress, IsRare: payload.IsRare && !payload.IsProgress,
		DurationMS: notificationExpireTime(payload.IsProgress).Milliseconds(),
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}
