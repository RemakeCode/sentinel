package notifier

import (
	"testing"
	"time"

	"sentinel/backend/ach"
	"sentinel/backend/config"
	steamtypes "sentinel/backend/steam/types"

	"github.com/stretchr/testify/assert"
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
