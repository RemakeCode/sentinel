package notifier

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/steam"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed media
var media embed.FS

type Service struct {
}

var (
	instance     *Service
	instanceOnce sync.Once
)

func Get() *Service {
	instanceOnce.Do(func() {
		instance = &Service{}
	})
	return instance
}

var cfg *config.File

func init() {
	err := os.MkdirAll(backend.MediaDir, 0755)
	if err != nil {
		slog.Error("Failed to create MediaDir", "error", err)
		return
	}

	entries, _ := media.ReadDir("media")
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		destPath := filepath.Join(backend.MediaDir, entry.Name())
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			data, err := media.ReadFile(filepath.Join("media", entry.Name()))
			if err != nil {
				slog.Warn("Failed to read embedded media file", "file", entry.Name(), "error", err)
				continue
			}

			if err := os.WriteFile(destPath, data, 0644); err != nil {
				slog.Warn("Failed to write media file", "file", entry.Name(), "error", err)
				continue
			}
		}
	}
}

// ServiceStartup is called when the Wails application starts
func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	c, err := config.Get()
	if err != nil {
		return err
	}
	cfg = c

	if isAvailable() {
		slog.Info("Notification service initialized and available")
	} else {
		slog.Warn("Notification service initialized but notify-send not found")
	}
	return nil
}

// SendNotification sends a system notification using notify-send.
// It uses fixed urgency "normal" and system default expiration.
// Accessible via Wails bindings.
// wails:internal
func (s *Service) SendNotification(appId string, achievements map[string]ach.Achievement, isProgress bool) error {
	if !isAvailable() {
		err := fmt.Errorf("notify-send not found in PATH")
		slog.Warn("Failed to send notification", "error", err)
		return err
	}

	for id, a := range achievements {
		notificationAch, e := s.getAchDataForNotification(appId)
		if e != nil {
			return nil
		}
		achievementsList := notificationAch.Achievement.List
		for _, achievement := range achievementsList {

			if strings.ToLower(achievement.Name) == strings.ToLower(id) {
				icon := strings.Split(strings.Replace(achievement.Icon, "https://", "", 1), "/")

				title := achievement.DisplayName
				message := achievement.Description
				imagePath := filepath.Join(backend.ACHCacheIconDir, appId, icon[len(icon)-1])

				if isProgress && a.MaxProgress > 0 {
					message = fmt.Sprintf("%s\n%s", message, progressBar(a.Progress, a.MaxProgress, 20))
				}

				args := []string{title, message, "--urgency", "normal", "-t", "10000", "-a", title}

				// Add icon if provided and exists
				if imagePath != "" {
					if _, err := os.Stat(imagePath); err == nil {
						args = append(args, "-h", fmt.Sprintf("%s%s", "string:image-path:", imagePath))
					}
				}

				// Add sound file if configured
				if cfg.NotificationSound != "" {
					soundPath := filepath.Join(backend.MediaDir, cfg.NotificationSound)
					if _, err := os.Stat(soundPath); err == nil {
						args = append(args, "-h", fmt.Sprintf("%s%s", "string:sound-file:", soundPath))
					} else {
						slog.Warn("Sound file not found, skipping sound", "sound", cfg.NotificationSound, "path", soundPath)
					}
				}

				cmd := exec.Command("notify-send", args...)

				if err := cmd.Run(); err != nil {
					err = fmt.Errorf("failed to execute notify-send: %w", err)
					slog.Warn("Failed to send notification", "error", err)
					return err
				}

				slog.Info("Sending notification", "title", title, "message", message, "imagePath", imagePath)

				break
			}
		}
	}

	return nil
}

// progressBar generates a visual progress bar with Unicode characters
// Example: "[████████░░░░░░░░░░░░] 8/100 (8.0%)"
// wails:internal
func progressBar(progress, max, width int) string {
	if max == 0 {
		return ""
	}

	filled := int(float64(progress) / float64(max) * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := "█"
	emptyBar := "░"

	barStr := strings.Repeat(bar, filled) + strings.Repeat(emptyBar, empty)
	percent := float64(progress) / float64(max) * 100.0

	return fmt.Sprintf("[%s] %d/%d (%.1f%%)", barStr, progress, max, percent)
}

// isAvailable checks if notify-send is available in the PATH
// wails:internal
func isAvailable() bool {
	_, err := exec.LookPath("notify-send")
	return err == nil
}

// wails:internal
func (s *Service) getAchDataForNotification(appId string) (*steam.GameBasics, error) {

	language := cfg.Language.API

	schemaPath := filepath.Join(backend.GameCacheDir, language, fmt.Sprintf("%s.json", appId))

	schemaByte, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	gb := steam.GameBasics{}
	err = json.Unmarshal(schemaByte, &gb)

	if err != nil {
		return nil, errors.New("failed to unmarshal steam game")
	}

	return &gb, nil
}
