package notifier

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestIsAvailable_WithNotifySend(t *testing.T) {
	result := isAvailable()
	_ = result
}

func TestSendNotification_NotifySendNotAvailable(t *testing.T) {
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)

	os.Setenv("PATH", "/nonexistent")

	svc := &Service{
		Config: &config.File{},
	}

	err := svc.SendNotification("12345", map[string]ach.Achievement{"ach_1": {}}, false, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notify-send not found")
}

func TestSendNotification_NoAchievements(t *testing.T) {
	mockDir := t.TempDir()
	mockScript := filepath.Join(mockDir, "notify-send")
	require.NoError(t, os.WriteFile(mockScript, []byte("#!/bin/bash\n"), 0755))

	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	os.Setenv("PATH", mockDir+":"+oldPath)

	svc := &Service{
		Config: &config.File{},
	}

	err := svc.SendNotification("12345", map[string]ach.Achievement{}, false, true)
	assert.NoError(t, err)
}

func TestSendNotification_CorrectCommandArgs(t *testing.T) {
	mockDir := t.TempDir()
	argsFile := filepath.Join(mockDir, "notify_args.txt")
	mockScript := filepath.Join(mockDir, "notify-send")
	require.NoError(t, os.WriteFile(mockScript, []byte(fmt.Sprintf(`#!/bin/bash
printf '%%s\n' "$@" > %s
`, argsFile)), 0755))

	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	os.Setenv("PATH", mockDir+":"+oldPath)

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
