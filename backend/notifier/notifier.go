package notifier

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/steam"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
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
}

var queueCap = 100

var (
	speakerMu          sync.Mutex
	speakerInitialized bool
)

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
			s.sendNotificationDesktop(payload)
			time.Sleep(backend.NotificationDelay)
		}
	}
}

func (s *Service) sendNotificationDesktop(payload *NotificationPayload) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		slog.Warn("Failed to connect to session bus", "error", err)
		return
	}
	defer conn.Close()

	hints := map[string]dbus.Variant{
		"urgency":   dbus.MakeVariant(byte(2)),
		"transient": dbus.MakeVariant(true),
	}

	if payload.IconPath != "" {
		if _, err := os.Stat(payload.IconPath); err == nil {
			hints["image-path"] = dbus.MakeVariant(payload.IconPath)
		}
	}

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		payload.GameName,
		uint32(0),
		"",
		payload.Title,
		payload.Message,
		[]string{},
		hints,
		int32(-1),
	)
	if call.Err != nil {
		slog.Warn("Failed to send notification", "error", call.Err)
		return
	}

	var notificationID uint32
	if err := call.Store(&notificationID); err != nil {
		slog.Warn("Failed to read notification ID", "error", err)
		return
	}

	if payload.SoundFile != "" {
		s.PlaySound(payload.SoundFile)
	}

	time.AfterFunc(backend.NotificationExpireTime, func() {
		s.closeNotification(notificationID)
	})

	slog.Info("Sent notification", "title", payload.Title, "game", payload.GameName)
}

func (s *Service) closeNotification(notificationID uint32) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		slog.Warn("Failed to connect to session bus for notification close", "id", notificationID, "error", err)
		return
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	if call := obj.Call("org.freedesktop.Notifications.CloseNotification", 0, notificationID); call.Err != nil {
		slog.Warn("Failed to close notification", "id", notificationID, "error", call.Err)
	}
}

func (s *Service) SendNotification(appId string, achievements map[string]ach.Achievement, isProgress bool, shouldNotify bool) error {
	slog.Info("SendNotification called", "appId", appId, "achievementsCount", len(achievements))

	for id, a := range achievements {
		notificationAch, gameName, e := s.getAchDataForNotification(appId)
		if e != nil {
			return nil
		}
		achievementsList := notificationAch.Achievement.List
		for _, achievement := range achievementsList {

			var title string
			if strings.EqualFold(achievement.Name, id) {
				icon := strings.Split(strings.Replace(achievement.Icon, "https://", "", 1), "/")

				title = achievement.DisplayName
				message := achievement.Description
				imagePath := filepath.Join(backend.ACHCacheIconDir, appId, icon[len(icon)-1])

				if isProgress && a.MaxProgress > 0 {
					title = achievement.Description
					message = progressBar(a.Progress, a.MaxProgress, 22)
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
					IconPath:    imagePath,
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

func initSpeaker() bool {
	speakerMu.Lock()
	defer speakerMu.Unlock()

	if speakerInitialized {
		return true
	}

	sampleRate := beep.SampleRate(44100)
	if err := speaker.Init(sampleRate, sampleRate.N(time.Second/10)); err != nil {
		slog.Warn("Failed to initialize audio speaker", "error", err)
		return false
	}

	speakerInitialized = true
	return true
}

// PlaySound plays a sound file asynchronously.
func (s *Service) PlaySound(filename string) error {
	if filename == "" {
		return nil
	}

	soundPath := filepath.Join(backend.MediaDir, filename)
	if _, err := os.Stat(soundPath); err != nil {
		return nil
	}

	go func() {
		if !initSpeaker() {
			return
		}

		file, err := os.Open(soundPath)
		if err != nil {
			slog.Warn("Failed to open sound file", "filename", filename, "error", err)
			return
		}
		defer file.Close()

		streamer, _, err := wav.Decode(file)
		if err != nil {
			slog.Warn("Failed to decode sound file", "filename", filename, "error", err)
			return
		}
		defer streamer.Close()

		done := make(chan struct{})
		speaker.Play(beep.Seq(streamer, beep.Callback(func() {
			close(done)
		})))
		<-done
	}()

	return nil
}

func (s *Service) GetNotificationExpireTime() int {
	return int(backend.NotificationExpireTime / time.Millisecond)
}

//wails:internal
func (s *Service) ServiceShutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}
