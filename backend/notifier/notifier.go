package notifier

import (
	"context"
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

	"github.com/wailsapp/wails/v3/pkg/application"
)

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

	if IsAvailable() {
		slog.Info("Notification service initialized and available")
	} else {
		slog.Warn("Notification service initialized but notify-send not found")
	}
	return nil
}

// SendNotification sends a system notification using notify-send.
// It uses fixed urgency "normal" and system default expiration.
// Accessible via Wails bindings.
func (s *Service) SendNotification(appId string, earnedAchievement map[string]ach.Achievement) error {
	app := application.Get()
	app.Event.Emit("sentinel::data-updated")

	if !IsAvailable() {
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
							args = append(args, "--icon", imagePath)
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

// IsAvailable checks if notify-send is available in the PATH
func IsAvailable() bool {
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
