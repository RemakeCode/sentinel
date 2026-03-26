package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sentinel/backend/ach"
	"sentinel/backend/steam"
	steamtypes "sentinel/backend/steam/types"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Mocks ──────────────────────────────────────────────────────────────────

type mockConfig struct {
	paths []string
}

func (m *mockConfig) GetEmulatorPaths() ([]string, error) { return m.paths, nil }
func (m *mockConfig) GetLanguage() steamtypes.Language    { return steamtypes.Language{API: "english"} }

type mockSteam struct {
	calledWithAppIDs []string
}

func (m *mockSteam) FetchAppDetailsBulk(appIDs []string, language steamtypes.Language) ([]*steam.GameBasics, error) {
	m.calledWithAppIDs = append(m.calledWithAppIDs, appIDs...)
	return nil, nil
}

type mockNotifier struct {
	calls  int
	lastID string
}

func (m *mockNotifier) SendNotification(appId string, diff map[string]ach.Achievement) error {
	m.calls++
	m.lastID = appId
	return nil
}

type mockAchManager struct {
	parseResult *ach.AchievementData
	cacheResult *ach.AchievementData
	saveCalls   int
}

func (m *mockAchManager) SaveAch(path string) error { m.saveCalls++; return nil }
func (m *mockAchManager) ParseAch(path string) (*ach.AchievementData, error) {
	if m.parseResult != nil {
		return m.parseResult, nil
	}
	return &ach.AchievementData{Achievements: map[string]ach.Achievement{}}, nil
}
func (m *mockAchManager) LoadCachedAch(appId string) (*ach.AchievementData, error) {
	return m.cacheResult, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func createTestWatcher(t *testing.T) *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	t.Cleanup(func() {
		watcher.Close()
	})
	return watcher
}

// ─── scan tests ──────────────────────────────────────────────────────────────

func TestScan_NumericDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create test directories
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "12345"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "67890"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "notanumber"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "12abc"), 0755))

	service := &Service{}
	result := service.scan([]string{tempDir})

	assert.Len(t, result.AppIDs, 2)
	assert.Contains(t, result.AppIDs, "12345")
	assert.Contains(t, result.AppIDs, "67890")
	assert.Len(t, result.AppIDPaths, 2)
}

func TestScan_EmptyPath(t *testing.T) {
	service := &Service{}
	result := service.scan([]string{"/nonexistent/path"})

	assert.Empty(t, result.AppIDs)
	assert.Empty(t, result.AppIDPaths)
}

func TestScan_MultiplePaths(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir1, "11111"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir2, "22222"), 0755))

	service := &Service{}
	result := service.scan([]string{tempDir1, tempDir2})

	assert.Len(t, result.AppIDs, 2)
	assert.Contains(t, result.AppIDs, "11111")
	assert.Contains(t, result.AppIDs, "22222")
}

// ─── numericRegex tests ───────────────────────────────────────────────────────

func TestNumericRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"12345", true},
		{"0", true},
		{"99999999", true},
		{"abc", false},
		{"12abc", false},
		{"abc123", false},
		{"12.34", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matched := numericRegex.MatchString(tt.input)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

// ─── watchPath tests ──────────────────────────────────────────────────────────

func TestWatchPath_NonexistentPath(t *testing.T) {
	service := &Service{}
	err := service.watchPath("/nonexistent/path")
	assert.Error(t, err)
}

func TestWatchPath_FileInsteadOfDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	service := &Service{}
	err := service.watchPath(testFile)
	assert.Error(t, err)
}

func TestWatchPath_ValidDirectory(t *testing.T) {
	tempDir := t.TempDir()

	service := &Service{}
	service.watcher = createTestWatcher(t)

	err := service.watchPath(tempDir)
	assert.NoError(t, err)
}

// ─── Stop tests ───────────────────────────────────────────────────────────────

func TestStop_NilWatcher(t *testing.T) {
	// Should not panic when stopping with nil watcher
	service := &Service{}
	service.Stop()
}

func TestStop_WithActiveWatcher(t *testing.T) {
	service := &Service{}
	service.watcher = createTestWatcher(t)
	service.done = make(chan struct{})
	service.retryTimer = time.NewTimer(time.Hour)

	service.Stop()

	assert.NotNil(t, service.watcher)
}

// ─── Start tests ──────────────────────────────────────────────────────────────

func TestStart_NoEmulatorPaths(t *testing.T) {
	service := &Service{
		config: &mockConfig{paths: []string{}},
		steam:  &mockSteam{},
		ach:    &mockAchManager{},
	}

	err := service.Start()

	assert.NoError(t, err)
	assert.Nil(t, service.watcher)
}

func TestStart_WithEmulatorPaths(t *testing.T) {
	tempDir := t.TempDir()
	appIDDir := filepath.Join(tempDir, "99999")
	require.NoError(t, os.MkdirAll(appIDDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(appIDDir, "achievements.json"), []byte(`{}`), 0644))

	steamMock := &mockSteam{}
	service := &Service{
		config: &mockConfig{paths: []string{tempDir}},
		steam:  steamMock,
		ach:    &mockAchManager{},
	}

	err := service.Start()
	require.NoError(t, err)

	assert.NotNil(t, service.watcher)
	// triggerMetadataFetch runs async - give it a moment
	time.Sleep(10 * time.Millisecond)
	assert.Contains(t, steamMock.calledWithAppIDs, "99999")

	service.Stop()
}

// ─── handleEvent tests ────────────────────────────────────────────────────────

func TestHandleEvent_AchievementsWrite_CallsNotifier(t *testing.T) {
	notifMock := &mockNotifier{}
	achMock := &mockAchManager{
		parseResult: &ach.AchievementData{
			Achievements: map[string]ach.Achievement{
				"ach_1": {Earned: true},
			},
		},
		cacheResult: nil, // no cached data → diff includes ach_1
	}

	service := &Service{
		notifier: notifMock,
		ach:      achMock,
	}

	event := fsnotify.Event{
		Name: "/fake/path/12345/achievements.json",
		Op:   fsnotify.Write,
	}
	service.handleEvent(event)

	assert.Equal(t, 1, notifMock.calls)
	assert.Equal(t, "12345", notifMock.lastID)
}

func TestHandleEvent_NoDiff_DoesNotCallNotifier(t *testing.T) {
	existing := &ach.AchievementData{
		Achievements: map[string]ach.Achievement{
			"ach_1": {Earned: true},
		},
	}
	notifMock := &mockNotifier{}
	achMock := &mockAchManager{
		// parse returns same data as cache → no diff
		parseResult: existing,
		cacheResult: existing,
	}

	service := &Service{
		notifier: notifMock,
		ach:      achMock,
	}

	event := fsnotify.Event{
		Name: "/fake/path/12345/achievements.json",
		Op:   fsnotify.Write,
	}
	service.handleEvent(event)

	assert.Equal(t, 0, notifMock.calls)
}

func TestHandleEvent_NonAchievementsFile(t *testing.T) {
	notifMock := &mockNotifier{}

	service := &Service{
		notifier: notifMock,
		ach:      &mockAchManager{},
	}

	event := fsnotify.Event{
		Name: "/fake/path/12345/other.txt",
		Op:   fsnotify.Write,
	}
	service.handleEvent(event)

	assert.Equal(t, 0, notifMock.calls)
}

// ─── retryFailedPaths tests ───────────────────────────────────────────────────

func TestRetryFailedPaths(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "99999"), 0755))

	service := &Service{
		failedPaths: []string{filepath.Join(tempDir, "99999")},
		done:        make(chan struct{}),
	}
	service.watcher = createTestWatcher(t)

	service.retryFailedPaths()

	assert.Empty(t, service.failedPaths)
}

func TestRetryTimer_Creation(t *testing.T) {
	service := &Service{}
	service.done = make(chan struct{})

	// Verify timer can be created
	service.startRetryTimer()
	assert.NotNil(t, service.retryTimer)

	// Cleanup
	service.Stop()
}

// ─── triggerMetadataFetch tests ───────────────────────────────────────────────

func TestTriggerMetadataFetch_CallsSteam(t *testing.T) {
	steamMock := &mockSteam{}
	service := &Service{
		config: &mockConfig{},
		steam:  steamMock,
	}

	service.triggerMetadataFetch([]string{"111", "222"})

	// goroutine - wait briefly for it to complete
	time.Sleep(10 * time.Millisecond)

	assert.Contains(t, steamMock.calledWithAppIDs, "111")
	assert.Contains(t, steamMock.calledWithAppIDs, "222")
}

func TestTriggerMetadataFetch_Empty_DoesNotCallSteam(t *testing.T) {
	steamMock := &mockSteam{}
	service := &Service{
		config: &mockConfig{},
		steam:  steamMock,
	}

	service.triggerMetadataFetch([]string{})

	assert.Empty(t, steamMock.calledWithAppIDs)
}
