package notifier

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/steam"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed media
var media embed.FS

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
			src, err := media.Open(filepath.Join("media", entry.Name()))
			if err != nil {
				continue
			}

			dest, err := os.Create(destPath)
			if err != nil {
				slog.Warn("Failed to create media file", "file", entry.Name(), "error", err)
				src.Close()
				continue
			}

			if _, err := io.Copy(dest, src); err != nil {
				slog.Warn("Failed to copy media", "file", entry.Name(), "error", err)
				src.Close()
				dest.Close()
				os.Remove(destPath)
				continue
			}

			if err := src.Close(); err != nil {
				slog.Warn("Failed to close source", "file", entry.Name(), "error", err)
			}

			if err := dest.Close(); err != nil {
				slog.Warn("Failed to close destination", "file", entry.Name(), "error", err)
			}
		}
	}
}

var cfg *config.File

// Service provides notification functionality to the Wails frontend
type Service struct {
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
func (s *Service) SendNotification(appId string, earnedAchievement map[string]ach.Achievement) error {

	slog.Info("earnedAchievement", earnedAchievement)
	app := application.Get()
	app.Event.Emit("sentinel::data-updated")

	if !isAvailable() {
		err := fmt.Errorf("notify-send not found in PATH")
		slog.Warn("Failed to send notification", "error", err)
		return err
	}

	for id, a := range earnedAchievement {
		if a.Earned {
			notificationAch, e := s.getAchDataForNotification(appId)
			achievements := notificationAch.Achievement.List
			for _, achievement := range achievements {

				if strings.ToLower(achievement.Name) == strings.ToLower(id) {
					icon := strings.Split(strings.Replace(achievement.Icon, "https://", "", 1), "/")

					title := achievement.DisplayName
					message := achievement.Description
					imagePath := filepath.Join(backend.ACHCacheIconDir, appId, icon[len(icon)-1])

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

			if e != nil {
				return nil
			}

		}

	}

	return nil
}

// isAvailable checks if notify-send is available in the PATH
func isAvailable() bool {
	_, err := exec.LookPath("notify-send")
	return err == nil
}

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

func (s *Service) GetMediaFileBase64(name string) (string, error) {
	data, err := media.ReadFile(filepath.Join("media", name))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
