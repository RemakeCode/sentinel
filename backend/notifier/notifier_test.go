package notifier

import (
	"testing"

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
	// Should have about 12 filled and 13 empty bars (25 total)
	assert.Contains(t, result, "█")
	assert.Contains(t, result, "░")
}

func TestProgressBar_Complete(t *testing.T) {
	result := progressBar(100, 100, 25)
	assert.Contains(t, result, "100/100")
	assert.Contains(t, result, "100.0%")
	// All bars should be filled
	assert.NotContains(t, result, "░")
}

func TestProgressBar_ProgressExceedsMax(t *testing.T) {
	result := progressBar(150, 100, 25)
	assert.Contains(t, result, "150/100")
	// Progress should be capped at width
	assert.NotContains(t, result, "░")
}

func TestProgressBar_DifferentWidth(t *testing.T) {
	result := progressBar(25, 100, 50)
	assert.Contains(t, result, "25/100")
	assert.Contains(t, result, "25.0%")
	// Should have about 12 filled and 38 empty bars (50 total)
}

func TestIsAvailable_WithNotifySend(t *testing.T) {
	// This test depends on the system having notify-send
	// Just verify the function doesn't panic
	result := isAvailable()
	// Result could be true or false depending on the system
	_ = result
}
