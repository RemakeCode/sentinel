package notifier

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	steamtypes "sentinel/backend/steam/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressBar_ZeroMax(t *testing.T) {
	result := progressBar(0, 0, 25)
	assert.Empty(t, result)
}

func TestProgressBar_HalfProgress(t *testing.T) {
	result := progressBar(50, 100, 25)
	assert.Contains(t, result, "50/100")
	assert.Contains(t, result, "50.0%")
	assert.Contains(t, result, "█")
	assert.Contains(t, result, "░")
}

func TestProgressBar_Complete(t *testing.T) {
	result := progressBar(100, 100, 25)
	assert.Contains(t, result, "100/100")
	assert.Contains(t, result, "100.0%")
	assert.NotContains(t, result, "░")
}

func TestProgressBar_ProgressExceedsMax(t *testing.T) {
	result := progressBar(150, 100, 25)
	assert.Contains(t, result, "150/100")
	assert.NotContains(t, result, "░")
}

func TestProgressBar_DifferentWidth(t *testing.T) {
	result := progressBar(25, 100, 50)
	assert.Contains(t, result, "25/100")
	assert.Contains(t, result, "25.0%")
}

func TestSendNotification_NoAchievements(t *testing.T) {
	svc := &Service{
		Config: &config.File{},
	}

	err := svc.SendNotification("12345", map[string]ach.Achievement{}, false, true)
	assert.NoError(t, err)
}

func TestSendNotification_QueuesPayload(t *testing.T) {
	svc := &Service{
		Config: &config.File{
			Language:          steamtypes.Language{API: "english"},
			NotificationSound: "",
		},
	}
	svc.notificationQueue = make(chan *NotificationPayload, 10)

	payload := &NotificationPayload{
		Title:      "Test Game",
		Message:    "Achievement Unlocked!",
		IconPath:   "",
		GameName:   "Test Game",
		Progress:   0,
		IsProgress: false,
	}

	svc.notificationQueue <- payload

	select {
	case p := <-svc.notificationQueue:
		assert.Equal(t, "Test Game", p.Title)
		assert.Equal(t, "Achievement Unlocked!", p.Message)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification")
	}
}

func TestSendNotification_ProgressUpdateModes(t *testing.T) {
	appID := setupNotifierCache(t)

	tests := []struct {
		name        string
		mode        config.AchievementProgressUpdateMode
		wantQueued  bool
		wantSound   string
		wantMessage string
	}{
		{
			name:        "default queues progress with sound",
			mode:        config.AchievementProgressUpdateModeDefault,
			wantQueued:  true,
			wantSound:   "steam-deck.wav",
			wantMessage: "4/10",
		},
		{
			name:        "silent queues progress without sound",
			mode:        config.AchievementProgressUpdateModeSilent,
			wantQueued:  true,
			wantSound:   "",
			wantMessage: "4/10",
		},
		{
			name:       "disabled drops progress",
			mode:       config.AchievementProgressUpdateModeDisabled,
			wantQueued: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newNotifierTestService(tt.mode)

			err := svc.SendNotification(appID, map[string]ach.Achievement{
				"ACH_PROGRESS": {Progress: 4, MaxProgress: 10},
			}, true, true)
			require.NoError(t, err)

			if !tt.wantQueued {
				requireNoQueuedPayload(t, svc)
				return
			}

			payload := requireQueuedPayload(t, svc)
			assert.True(t, payload.IsProgress)
			assert.Equal(t, tt.wantSound, payload.SoundFile)
			assert.Contains(t, payload.Message, tt.wantMessage)
		})
	}
}

func TestSendNotification_EarnedIgnoresProgressUpdateMode(t *testing.T) {
	appID := setupNotifierCache(t)
	svc := newNotifierTestService(config.AchievementProgressUpdateModeDisabled)

	err := svc.SendNotification(appID, map[string]ach.Achievement{
		"ACH_PROGRESS": {Earned: true, Progress: 10, MaxProgress: 10},
	}, false, true)
	require.NoError(t, err)

	payload := requireQueuedPayload(t, svc)
	assert.False(t, payload.IsProgress)
	assert.Equal(t, "steam-deck.wav", payload.SoundFile)
	assert.Equal(t, "Progress Achievement", payload.Title)
	assert.Equal(t, "Progress Description", payload.Message)
}

func TestTestNotificationProgress_IgnoresProgressUpdateMode(t *testing.T) {
	tests := []struct {
		name string
		mode config.AchievementProgressUpdateMode
	}{
		{name: "silent", mode: config.AchievementProgressUpdateModeSilent},
		{name: "disabled", mode: config.AchievementProgressUpdateModeDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newNotifierTestService(tt.mode)

			err := svc.TestNotificationProgress()
			require.NoError(t, err)

			payload := requireQueuedPayload(t, svc)
			assert.True(t, payload.IsProgress)
			assert.Equal(t, "steam-deck.wav", payload.SoundFile)
		})
	}
}

func TestProgressNotificationTiming(t *testing.T) {
	assert.Equal(t, backend.ProgressNotificationExpireTime, notificationExpireTime(true))
	assert.Equal(t, backend.NotificationExpireTime, notificationExpireTime(false))
	assert.Equal(t, backend.ProgressNotificationDelay, notificationDelay(true))
	assert.Equal(t, backend.NotificationDelay, notificationDelay(false))
}

func setupNotifierCache(t *testing.T) string {
	t.Helper()

	originalGameCacheDir := backend.GameCacheDir
	originalMediaDir := backend.MediaDir
	originalIconDir := backend.ACHCacheIconDir

	tempDir := t.TempDir()
	backend.GameCacheDir = filepath.Join(tempDir, "games")
	backend.MediaDir = filepath.Join(tempDir, "media")
	backend.ACHCacheIconDir = filepath.Join(tempDir, "icons")

	t.Cleanup(func() {
		backend.GameCacheDir = originalGameCacheDir
		backend.MediaDir = originalMediaDir
		backend.ACHCacheIconDir = originalIconDir
	})

	appID := "12345"
	cacheDir := filepath.Join(backend.GameCacheDir, "english")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))
	require.NoError(t, os.MkdirAll(backend.MediaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backend.MediaDir, "steam-deck.wav"), []byte("sound"), 0644))

	gameData := `{
		"AppID": "12345",
		"Name": "Test Game",
		"Achievement": {
			"Total": 1,
			"List": [
				{
					"Name": "ACH_PROGRESS",
					"DisplayName": "Progress Achievement",
					"Description": "Progress Description",
					"Icon": "https://cdn.example/icon.png"
				}
			]
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, appID+".json"), []byte(gameData), 0644))

	return appID
}

func newNotifierTestService(mode config.AchievementProgressUpdateMode) *Service {
	return &Service{
		Config: &config.File{
			Language:                      steamtypes.Language{API: "english"},
			NotificationSound:             "steam-deck.wav",
			AchievementProgressUpdateMode: mode,
		},
		notificationQueue: make(chan *NotificationPayload, 10),
	}
}

func requireQueuedPayload(t *testing.T, svc *Service) *NotificationPayload {
	t.Helper()

	select {
	case payload := <-svc.notificationQueue:
		return payload
	default:
		t.Fatal("expected queued notification payload")
		return nil
	}
}

func requireNoQueuedPayload(t *testing.T, svc *Service) {
	t.Helper()

	select {
	case payload := <-svc.notificationQueue:
		t.Fatalf("expected no queued notification payload, got %#v", payload)
	default:
	}
}
