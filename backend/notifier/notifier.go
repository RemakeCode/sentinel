package notifier

import (
	"bytes"
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
	"sentinel/backend/decky"
	"sentinel/backend/steam"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed media
var media embed.FS

type NotificationPayload struct {
	Title       string
	Message     string
	IconPath    string
	SoundFile   string
	GameName    string
	Progress    int
	MaxProgress int
	IsProgress  bool
}

type Service struct {
	notificationQueue chan *NotificationPayload
	ctx               context.Context
	cancel            context.CancelFunc
	Config            *config.File
	clients           map[string]chan string
	mu                sync.RWMutex
}

var queueCap = 100

func init() {
	err := os.MkdirAll(backend.MediaDir, 0755)
	if err != nil {
		slog.Error("Failed to create MediaDir", "error", err)
		return
	}

	slog.Info("Copying embedded media files to config directory")
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

func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	// Config must be injected before startup
	if s.Config == nil {
		slog.Error("Config not injected into notifier service")
		return fmt.Errorf("config not injected into notifier service")
	}

	s.notificationQueue = make(chan *NotificationPayload, queueCap)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.clients = make(map[string]chan string)

	go s.notificationWorker()

	slog.Info("Notification service initialized")
	return nil
}

func (s *Service) notificationWorker() {
	slog.Info("Notification worker started")
	for {
		select {
		case <-s.ctx.Done():
			slog.Info("Notification worker shutting down")
			return
		case payload := <-s.notificationQueue:
			slog.Info("Worker received payload", "title", payload.Title, "game", payload.GameName, "isProgress", payload.IsProgress)
			if decky.IsDecky() {
				s.sendNotificationSSE(payload)
			} else {
				s.sendNotificationSync(payload)
			}
			time.Sleep(backend.NotificationDelay)
		}
	}
}

func (s *Service) sendNotificationSync(payload *NotificationPayload) {
	args := []string{payload.Title, payload.Message, "--urgency", "critical", "--transient", "-p", "-a", payload.GameName}

	if payload.IconPath != "" {
		if _, err := os.Stat(payload.IconPath); err == nil {
			args = append(args, "-h", fmt.Sprintf("%s%s", "string:image-path:", payload.IconPath))
		}
	}

	cmd := exec.Command("notify-send", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Creates a new session, detaching from the terminal
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	go func() {
		if err := cmd.Run(); err != nil {
			slog.Warn("Failed to send notification", "error", err)
			return
		}

		if payload.SoundFile != "" {
			s.PlaySound(payload.SoundFile)
		}

		idStr := strings.TrimSpace(stdout.String())
		id, err := strconv.Atoi(idStr)
		if err != nil {
			slog.Warn("Failed to parse notification ID", "raw", idStr, "error", err)
			return
		}

		time.AfterFunc(backend.NotificationExpireTime, func() {
			closeCmd := exec.Command("busctl", "--user", "call",
				"org.freedesktop.Notifications",
				"/org/freedesktop/Notifications",
				"org.freedesktop.Notifications",
				"CloseNotification", "u", strconv.Itoa(id))
			if err := closeCmd.Run(); err != nil {
				slog.Warn("Failed to close notification", "id", id, "error", err)
			}
		})
	}()

	slog.Info("Sent notification", "title", payload.Title, "game", payload.GameName)
}

func (s *Service) SendNotification(appId string, achievements map[string]ach.Achievement, isProgress bool, shouldNotify bool) error {
	slog.Info("SendNotification called", "appId", appId, "achievementsCount", len(achievements))
	if !isAvailable() {
		err := fmt.Errorf("notify-send not found in PATH")
		slog.Warn("Failed to send notification", "error", err)
		return err
	}

	for id, a := range achievements {
		notificationAch, gameName, e := s.getAchDataForNotification(appId)
		if e != nil {
			return nil
		}
		achievementsList := notificationAch.Achievement.List
		for _, achievement := range achievementsList {

			var title string
			if strings.EqualFold(achievement.Name, id) {
				iconPath := filepath.Join(backend.ACHCacheIconDir, appId, filepath.Base(achievement.Icon))
				title = achievement.DisplayName
				message := achievement.Description

				if isProgress && a.MaxProgress > 0 {
					if !decky.IsDecky() {
						title = achievement.Description
						message = progressBar(a.Progress, a.MaxProgress, 22)
					}
				}

				var soundFile string
				if shouldNotify && s.Config.NotificationSound != "" {
					soundFile = s.Config.NotificationSound
					soundPath := filepath.Join(backend.MediaDir, soundFile)
					if _, err := os.Stat(soundPath); err != nil {
						slog.Warn("Sound file not found, skipping sound", "sound", s.Config.NotificationSound, "path", soundPath)
						soundFile = ""
					}
				}

				payload := &NotificationPayload{
					Title:       title,
					Message:     message,
					IconPath:    iconPath,
					SoundFile:   soundFile,
					GameName:    gameName,
					Progress:    a.Progress,
					MaxProgress: a.MaxProgress,
					IsProgress:  isProgress,
				}

				select {
				case s.notificationQueue <- payload:
					slog.Info("Queued notification", "title", title, "game", gameName)
				default:
					slog.Warn("Notification queue full, dropping notification", "title", title)
				}
				break
			}
		}
	}

	return nil
}

func (s *Service) TestNotification() error {
	slog.Info("TestNotification called")

	payload := &NotificationPayload{
		Title:       "Test Notification",
		Message:     "For those who come after",
		IconPath:    filepath.Join(backend.MediaDir, "sentinel.png"),
		SoundFile:   s.Config.NotificationSound,
		GameName:    "Sentinel",
		Progress:    0,
		MaxProgress: 0,
		IsProgress:  false,
	}

	select {
	case s.notificationQueue <- payload:
		slog.Info("Queued test notification")
	default:
		slog.Warn("Notification queue full, dropping test notification")
	}

	return nil
}

func (s *Service) TestNotificationProgress() error {
	slog.Info("TestNotificationProgress called")

	payload := &NotificationPayload{
		Title:       "For those who come after",
		Message:     progressBar(7, 10, 22),
		IconPath:    filepath.Join(backend.MediaDir, "sentinel.png"),
		SoundFile:   s.Config.NotificationSound,
		GameName:    "Sentinel",
		Progress:    7,
		MaxProgress: 10,
		IsProgress:  true,
	}

	select {
	case s.notificationQueue <- payload:
		slog.Info("Queued test progress notification")
	default:
		slog.Warn("Notification queue full, dropping test progress notification")
	}

	return nil
}

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

	return fmt.Sprintf("%s %d/%d (%.1f%%)", barStr, progress, max, percent)
}

func isAvailable() bool {
	_, err := exec.LookPath("notify-send")
	return err == nil
}

func (s *Service) getAchDataForNotification(appId string) (*steam.GameBasics, string, error) {
	language := s.Config.Language.API

	schemaPath := filepath.Join(backend.GameCacheDir, language, fmt.Sprintf("%s.json", appId))

	schemaByte, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, "", err
	}

	gb := steam.GameBasics{}
	err = json.Unmarshal(schemaByte, &gb)

	if err != nil {
		return nil, "", errors.New("failed to unmarshal steam game")
	}

	return &gb, gb.Name, nil
}

// PlaySound plays a sound file asynchronously using paplay or aplay
func (s *Service) PlaySound(filename string) error {
	if filename == "" {
		return nil
	}

	soundPath := filepath.Join(backend.MediaDir, filename)
	if _, err := os.Stat(soundPath); err != nil {
		return nil
	}

	go func() {
		var cmd *exec.Cmd
		if _, err := exec.LookPath("paplay"); err == nil {
			cmd = exec.Command("paplay", soundPath)
		} else if _, err := exec.LookPath("aplay"); err == nil {
			cmd = exec.Command("aplay", soundPath)
		} else {
			slog.Warn("No audio playback utility available (paplay/aplay)")
			return
		}

		if err := cmd.Run(); err != nil {
			slog.Warn("Failed to play sound", "filename", filename, "error", err)
		}
	}()

	return nil
}

func (s *Service) GetNotificationExpireTime() int {
	return int(backend.NotificationExpireTime / time.Millisecond)
}

// RegisterClient registers a new SSE client
func (s *Service) RegisterClient(clientID string, notifications chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[clientID] = notifications
}

// UnregisterClient removes a client from the notifier service
func (s *Service) UnregisterClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, clientID)
}

// sendNotificationSSE sends a notification to all connected SSE clients
func (s *Service) sendNotificationSSE(payload *NotificationPayload) {
	slog.Info("SSE Notification called")

	// Convert local icon path to virtual path for Decky frontend
	if payload.IconPath != "" && filepath.IsAbs(payload.IconPath) {
		if relPath, err := filepath.Rel(backend.DataDir, payload.IconPath); err == nil {
			payload.IconPath = "/api/media/" + filepath.ToSlash(relPath)
		}
	}

	jsonData, _ := json.Marshal(payload)

	// Send to all clients
	s.mu.RLock()
	for _, ch := range s.clients {
		select {
		case ch <- string(jsonData):
		default:
			// Skip if channel is full
		}
	}
	s.mu.RUnlock()
}

//wails:internal
func (s *Service) ServiceShutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}
