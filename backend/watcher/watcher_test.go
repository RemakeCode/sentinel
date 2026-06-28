package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sentinel/backend/ach"
	"sentinel/backend/config"
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
func (m *mockConfig) CheckShouldNotify(path string) bool  { return true }

type mockSteam struct {
	calledWithAppIDs []string
	done             chan struct{} // Signal for async completion
}

func (m *mockSteam) FetchAppDetailsBulk(appIDs []string, language steamtypes.Language) ([]*steam.GameBasics, error) {
	m.calledWithAppIDs = append(m.calledWithAppIDs, appIDs...)
	if m.done != nil {
		close(m.done)
	}
	return nil, nil
}

type mockNotifier struct {
	calls        int
	lastID       string
	isProgress   bool
	shouldNotify bool
	achievements map[string]ach.Achievement
}

func (m *mockNotifier) SendNotification(appId string, achievements map[string]ach.Achievement, isProgress bool, shouldNotify bool) error {
	m.calls++
	m.lastID = appId
	m.isProgress = isProgress
	m.shouldNotify = shouldNotify
	m.achievements = achievements
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
	// Return the cacheResult as-is (nil means no cached data exists)
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

func writeAchievementJSON(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "achievements.json"), []byte(`{}`), 0644))
}

func writeAchievementINI(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "achievements.ini"), []byte("[ACH1]\nAchieved=1\n"), 0644))
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

func TestStop_AlreadyStopped(t *testing.T) {
	service := &Service{}
	service.watcher = createTestWatcher(t)
	service.done = make(chan struct{})

	require.NotPanics(t, func() {
		service.Stop()
		service.Stop()
	})
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
		Config: &config.File{Prefixes: []config.Prefix{}},
		Steam:  &mockSteam{},
		Ach:    &mockAchManager{},
	}

	err := service.Start()

	assert.NoError(t, err)
	// Watcher is created but with no paths to watch
	assert.NotNil(t, service.watcher)

	// Clean up
	service.Stop()
}

func TestStart_WithEmulatorPaths(t *testing.T) {
	tempDir := t.TempDir()
	emuPath := filepath.Join("AppData", "Roaming", "GSE Saves")
	appIDDir := filepath.Join(tempDir, "drive_c", "users", "steamuser", emuPath, "99999")
	require.NoError(t, os.MkdirAll(appIDDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(appIDDir, "achievements.json"), []byte(`{}`), 0644))

	steamMock := &mockSteam{done: make(chan struct{})}
	service := &Service{
		Config: &config.File{
			Prefixes:  []config.Prefix{{Path: tempDir}},
			Emulators: []config.Emulator{{ID: "gse"}},
		},
		Steam: steamMock,
		Ach:   &mockAchManager{},
	}

	err := service.Start()
	require.NoError(t, err)

	assert.NotNil(t, service.watcher)
	// Wait for async triggerMetadataFetch to complete via channel
	<-steamMock.done
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
		cacheResult: &ach.AchievementData{
			Achievements: map[string]ach.Achievement{}, // empty cache → ach_1 is newly earned
		},
	}

	service := &Service{
		Notifier: notifMock,
		Ach:      achMock,
		Config:   &config.File{},
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
		Notifier: notifMock,
		Ach:      achMock,
		Config:   &config.File{},
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
		Notifier: notifMock,
		Ach:      &mockAchManager{},
		Config:   &config.File{},
	}

	event := fsnotify.Event{
		Name: "/fake/path/12345/other.txt",
		Op:   fsnotify.Write,
	}
	service.handleEvent(event)

	assert.Equal(t, 0, notifMock.calls)
}

func TestHandleEvent_INIAchievementsWrite_CallsNotifier(t *testing.T) {
	notifMock := &mockNotifier{}
	achMock := &mockAchManager{
		parseResult: &ach.AchievementData{
			Achievements: map[string]ach.Achievement{
				"ACH_1": {Earned: true, EarnedTime: 1000},
			},
		},
		cacheResult: &ach.AchievementData{
			Achievements: map[string]ach.Achievement{
				"ACH_1": {Earned: false},
			},
		},
	}

	appDir := "/fake/path/12345"
	service := &Service{
		Notifier: notifMock,
		Ach:      achMock,
		Config:   &config.File{},
		sourceByAppPath: map[string]config.EmulatorSource{
			appDir: {AchievementFile: "achievements.ini"},
		},
	}

	event := fsnotify.Event{
		Name: filepath.Join(appDir, "achievements.ini"),
		Op:   fsnotify.Write,
	}
	service.handleEvent(event)

	assert.Equal(t, 1, notifMock.calls)
	assert.Equal(t, "12345", notifMock.lastID)
	assert.False(t, notifMock.isProgress)
	assert.Contains(t, notifMock.achievements, "ACH_1")
}

func TestHandleEvent_INIProgressUpdate_CallsProgressNotifier(t *testing.T) {
	notifMock := &mockNotifier{}
	achMock := &mockAchManager{
		parseResult: &ach.AchievementData{
			Achievements: map[string]ach.Achievement{
				"ACH_PROGRESS": {Earned: false, Progress: 4, MaxProgress: 10},
			},
		},
		cacheResult: &ach.AchievementData{
			Achievements: map[string]ach.Achievement{
				"ACH_PROGRESS": {Earned: false, Progress: 3, MaxProgress: 10},
			},
		},
	}

	appDir := "/fake/path/12345"
	service := &Service{
		Notifier: notifMock,
		Ach:      achMock,
		Config:   &config.File{},
		sourceByAppPath: map[string]config.EmulatorSource{
			appDir: {AchievementFile: "achievements.ini"},
		},
	}

	event := fsnotify.Event{
		Name: filepath.Join(appDir, "achievements.ini"),
		Op:   fsnotify.Write,
	}
	service.handleEvent(event)

	assert.Equal(t, 1, notifMock.calls)
	assert.Equal(t, "12345", notifMock.lastID)
	assert.True(t, notifMock.isProgress)
	assert.Contains(t, notifMock.achievements, "ACH_PROGRESS")
}

func TestHandleEvent_SourceMappedAppIgnoresOtherAchievementFile(t *testing.T) {
	notifMock := &mockNotifier{}
	appDir := "/fake/path/12345"
	service := &Service{
		Notifier: notifMock,
		Ach:      &mockAchManager{},
		Config:   &config.File{},
		sourceByAppPath: map[string]config.EmulatorSource{
			appDir: {AchievementFile: "achievements.ini"},
		},
	}

	event := fsnotify.Event{
		Name: filepath.Join(appDir, "achievements.json"),
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
	steamMock := &mockSteam{done: make(chan struct{})}
	service := &Service{
		Config: &config.File{},
		Steam:  steamMock,
	}

	service.triggerMetadataFetch([]string{"111", "222"})

	// Wait for async goroutine to complete via channel
	<-steamMock.done

	assert.Contains(t, steamMock.calledWithAppIDs, "111")
	assert.Contains(t, steamMock.calledWithAppIDs, "222")
}

func TestTriggerMetadataFetch_Empty_DoesNotCallSteam(t *testing.T) {
	steamMock := &mockSteam{}
	service := &Service{
		Config: &config.File{},
		Steam:  steamMock,
	}

	service.triggerMetadataFetch([]string{})

	assert.Empty(t, steamMock.calledWithAppIDs)
}

func TestComputeFullPath_WithDriveC(t *testing.T) {
	tempDir := t.TempDir()

	// Create Wine-style prefix structure
	driveC := filepath.Join(tempDir, "drive_c")
	require.NoError(t, os.MkdirAll(filepath.Join(driveC, "users", "steamuser"), 0755))

	service := &Service{
		Config: &config.File{
			Emulators: []config.Emulator{
				{ID: "gse"},
			},
		},
	}

	paths, err := service.computeFullPath(tempDir)
	require.NoError(t, err)
	assert.Len(t, paths, 1)
	assert.Equal(t, filepath.Join(driveC, "users", "steamuser", "AppData/Roaming/GSE Saves"), paths[0])
}

func TestComputeFullPath_NoDriveC(t *testing.T) {
	tempDir := t.TempDir()

	service := &Service{
		Config: &config.File{
			Emulators: []config.Emulator{
				{ID: "gse"},
			},
		},
	}

	_, err := service.computeFullPath(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find drive_c")
}

// ─── isShortcutAppID tests ────────────────────────────────────────────────────

func TestIsShortcutAppID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0", false},
		{"1493710", false},
		{"1665460", false},
		{"2147483647", false},
		{"2147483648", true},
		{"3237746183", true},
		{"3989101494", true},
		{"abc", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isShortcutAppID(tt.input))
		})
	}
}

// ─── scanAndWatchPrefix tests ──────────────────────────────────────────────────

func TestScanAndWatchPrefix_Compatdata_FiltersShortcuts(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "compatdata")

	emuPath := filepath.Join("AppData", "Roaming", "GSE Saves")
	storeGameDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "1493710")
	shortcutDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "3237746183")
	require.NoError(t, os.MkdirAll(storeGameDir, 0755))
	require.NoError(t, os.MkdirAll(shortcutDir, 0755))
	writeAchievementJSON(t, storeGameDir)
	writeAchievementJSON(t, shortcutDir)

	achMock := &mockAchManager{}
	service := &Service{
		Ach:   achMock,
		Steam: &mockSteam{},
		Config: &config.File{
			Emulators: []config.Emulator{{ID: "gse"}},
		},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix(prefixDir)

	assert.Contains(t, service.watcher.WatchList(), storeGameDir)
	assert.NotContains(t, service.watcher.WatchList(), shortcutDir)
	assert.Equal(t, 1, achMock.saveCalls)
}

func TestScanAndWatchPrefix_Compatdata_DoesNotWalkEntireTree(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "compatdata")

	emuPath := filepath.Join("AppData", "Roaming", "GSE Saves")
	gameDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "1493710")
	require.NoError(t, os.MkdirAll(gameDir, 0755))
	writeAchievementJSON(t, gameDir)

	// Create numeric system dirs that should NOT be found by scan
	systemDir := filepath.Join(prefixDir, "drive_c", "windows", "system32", "spool", "drivers", "w32x86", "3")
	require.NoError(t, os.MkdirAll(systemDir, 0755))
	writeAchievementJSON(t, systemDir)

	achMock := &mockAchManager{}
	service := &Service{
		Ach:   achMock,
		Steam: &mockSteam{},
		Config: &config.File{
			Emulators: []config.Emulator{{ID: "gse"}},
		},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix(prefixDir)

	assert.Contains(t, service.watcher.WatchList(), gameDir)
	assert.NotContains(t, service.watcher.WatchList(), systemDir)
	assert.Equal(t, 1, achMock.saveCalls)
}

func TestScanAndWatchPrefix_NonCompatdata_StillScansEmuPaths(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "customprefix")

	emuPath := filepath.Join("AppData", "Roaming", "GSE Saves")
	gameDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "99999")
	require.NoError(t, os.MkdirAll(gameDir, 0755))
	writeAchievementJSON(t, gameDir)

	// Shortcut IDs should NOT be filtered for non-compatdata prefixes
	shortcutDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "3989101494")
	require.NoError(t, os.MkdirAll(shortcutDir, 0755))
	writeAchievementJSON(t, shortcutDir)

	achMock := &mockAchManager{}
	service := &Service{
		Ach:   achMock,
		Steam: &mockSteam{},
		Config: &config.File{
			Emulators: []config.Emulator{{ID: "gse"}},
		},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix(prefixDir)

	assert.Contains(t, service.watcher.WatchList(), gameDir)
	assert.Contains(t, service.watcher.WatchList(), shortcutDir)
	assert.Equal(t, 2, achMock.saveCalls)
}

func TestScanAndWatchPrefix_PrefixNoLongerExists(t *testing.T) {
	achMock := &mockAchManager{}
	service := &Service{
		Ach:   achMock,
		Steam: &mockSteam{},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix("/nonexistent/path")

	assert.Empty(t, service.watcher.WatchList())
	assert.Equal(t, 0, achMock.saveCalls)
}

func TestScanAndWatchPrefix_NoDriveC(t *testing.T) {
	tempDir := t.TempDir()

	achMock := &mockAchManager{}
	service := &Service{
		Ach:    achMock,
		Steam:  &mockSteam{},
		Config: &config.File{},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix(tempDir)

	assert.Empty(t, service.watcher.WatchList())
	assert.Equal(t, 0, achMock.saveCalls)
}

// ─── Start shortcut filtering test ────────────────────────────────────────────

func TestStart_CompatdataPrefix_FiltersShortcuts(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "compatdata")

	emuPath := filepath.Join("AppData", "Roaming", "GSE Saves")
	storeDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "1493710")
	shortcutDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "3989101494")
	require.NoError(t, os.MkdirAll(storeDir, 0755))
	require.NoError(t, os.MkdirAll(shortcutDir, 0755))
	writeAchievementJSON(t, storeDir)
	writeAchievementJSON(t, shortcutDir)

	steamMock := &mockSteam{done: make(chan struct{})}
	achMock := &mockAchManager{}
	service := &Service{
		Config: &config.File{
			Prefixes:  []config.Prefix{{Path: prefixDir}},
			Emulators: []config.Emulator{{ID: "gse"}},
		},
		Steam: steamMock,
		Ach:   achMock,
	}

	require.NoError(t, service.Start())
	<-steamMock.done

	assert.Contains(t, steamMock.calledWithAppIDs, "1493710")
	assert.NotContains(t, steamMock.calledWithAppIDs, "3989101494")
	assert.Len(t, steamMock.calledWithAppIDs, 1)

	service.Stop()
}

func TestStart_NonCompatdataPrefix_DoesNotFilterShortcuts(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "customprefix")

	emuPath := filepath.Join("AppData", "Roaming", "GSE Saves")
	storeDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "1493710")
	shortcutDir := filepath.Join(prefixDir, "drive_c", "users", "steamuser", emuPath, "3989101494")
	require.NoError(t, os.MkdirAll(storeDir, 0755))
	require.NoError(t, os.MkdirAll(shortcutDir, 0755))
	writeAchievementJSON(t, storeDir)
	writeAchievementJSON(t, shortcutDir)

	steamMock := &mockSteam{done: make(chan struct{})}
	achMock := &mockAchManager{}
	service := &Service{
		Config: &config.File{
			Prefixes:  []config.Prefix{{Path: prefixDir}},
			Emulators: []config.Emulator{{ID: "gse"}},
		},
		Steam: steamMock,
		Ach:   achMock,
	}

	require.NoError(t, service.Start())
	<-steamMock.done

	assert.Contains(t, steamMock.calledWithAppIDs, "1493710")
	assert.Contains(t, steamMock.calledWithAppIDs, "3989101494")
	assert.Len(t, steamMock.calledWithAppIDs, 2)

	service.Stop()
}

func TestComputeFullPath_WithSteamUserSources(t *testing.T) {
	tempDir := t.TempDir()

	// Create Wine-style prefix structure
	driveC := filepath.Join(tempDir, "drive_c")
	require.NoError(t, os.MkdirAll(filepath.Join(driveC, "users", "steamuser"), 0755))

	service := &Service{
		Config: &config.File{
			Emulators: []config.Emulator{
				{ID: "gse"},
				{ID: "goldberg-steamemu"},
			},
		},
	}

	paths, err := service.computeFullPath(tempDir)
	require.NoError(t, err)
	assert.Len(t, paths, 2)
	assert.Contains(t, paths, filepath.Join(driveC, "users", "steamuser", "AppData/Roaming/GSE Saves"))
	assert.Contains(t, paths, filepath.Join(driveC, "users", "steamuser", "AppData/Roaming/Goldberg SteamEmu Saves"))
}

func TestComputeFullPath_NoKnownEmulatorSources(t *testing.T) {
	service := &Service{
		Config: &config.File{
			Emulators: []config.Emulator{
				{ID: "unknown"},
			},
		},
	}

	paths, err := service.computeFullPath(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no configured emulator sources")
	assert.Empty(t, paths)
}

func TestComputeFullPath_WithBuiltInDriveCSources(t *testing.T) {
	tempDir := t.TempDir()
	driveC := filepath.Join(tempDir, "drive_c")
	require.NoError(t, os.MkdirAll(filepath.Join(driveC, "users", "steamuser"), 0755))

	service := &Service{
		Config: &config.File{
			Emulators: []config.Emulator{
				{ID: "goldberg-steamemu"},
				{ID: "codex"},
				{ID: "rune"},
			},
		},
	}

	paths, err := service.computeFullPath(tempDir)
	require.NoError(t, err)

	assert.Contains(t, paths, filepath.Join(driveC, "users", "steamuser", "AppData/Roaming/Goldberg SteamEmu Saves"))
	assert.Contains(t, paths, filepath.Join(driveC, "users", "Public", "Documents", "Steam", "CODEX"))
	assert.Contains(t, paths, filepath.Join(driveC, "users", "Public", "Documents", "Steam", "RUNE"))
}

func TestScanSources_IncludesNumericAppFoldersWithoutAchievementFile(t *testing.T) {
	tempDir := t.TempDir()
	jsonRoot := filepath.Join(tempDir, "json")
	iniRoot := filepath.Join(tempDir, "ini")
	jsonAppDir := filepath.Join(jsonRoot, "11111")
	iniAppDir := filepath.Join(iniRoot, "22222")
	missingFileDir := filepath.Join(jsonRoot, "33333")
	nonNumericDir := filepath.Join(iniRoot, "notanumber")
	require.NoError(t, os.MkdirAll(jsonAppDir, 0755))
	require.NoError(t, os.MkdirAll(iniAppDir, 0755))
	require.NoError(t, os.MkdirAll(missingFileDir, 0755))
	require.NoError(t, os.MkdirAll(nonNumericDir, 0755))
	writeAchievementJSON(t, jsonAppDir)
	writeAchievementINI(t, iniAppDir)
	writeAchievementINI(t, nonNumericDir)

	service := &Service{}
	result := service.scanSources([]resolvedSource{
		{
			Path: jsonRoot,
			Source: config.EmulatorSource{
				AchievementFile: "achievements.json",
			},
		},
		{
			Path: iniRoot,
			Source: config.EmulatorSource{
				AchievementFile: "achievements.ini",
			},
		},
		{
			Path: filepath.Join(tempDir, "missing"),
			Source: config.EmulatorSource{
				AchievementFile: "achievements.json",
			},
		},
	})

	assert.ElementsMatch(t, []string{"11111", "22222", "33333"}, result.AppIDs)
	assert.Contains(t, result.AppIDPaths, jsonAppDir)
	assert.Contains(t, result.AppIDPaths, iniAppDir)
	assert.Contains(t, result.AppIDPaths, missingFileDir)
	assert.Len(t, result.Sources, 3)
}

func TestScanAndWatchPrefix_WatchesNumericFolderBeforeAchievementFileExists(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "customprefix")

	codexDir := filepath.Join(prefixDir, "drive_c", "users", "Public", "Documents", "Steam", "CODEX", "814380")
	runeDir := filepath.Join(prefixDir, "drive_c", "users", "Public", "Documents", "Steam", "RUNE", "1716740")
	pendingDir := filepath.Join(prefixDir, "drive_c", "users", "Public", "Documents", "Steam", "CODEX", "22222")
	require.NoError(t, os.MkdirAll(codexDir, 0755))
	require.NoError(t, os.MkdirAll(runeDir, 0755))
	require.NoError(t, os.MkdirAll(pendingDir, 0755))
	writeAchievementINI(t, codexDir)
	writeAchievementINI(t, runeDir)

	achMock := &mockAchManager{}
	service := &Service{
		Ach:   achMock,
		Steam: &mockSteam{},
		Config: &config.File{
			Emulators: []config.Emulator{
				{ID: "codex"},
				{ID: "rune"},
			},
		},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix(prefixDir)

	assert.Contains(t, service.watcher.WatchList(), codexDir)
	assert.Contains(t, service.watcher.WatchList(), runeDir)
	assert.Contains(t, service.watcher.WatchList(), pendingDir)
	assert.Equal(t, 2, achMock.saveCalls)
	assert.Equal(t, "achievements.ini", service.sourceByAppPath[codexDir].AchievementFile)
	assert.Equal(t, "achievements.ini", service.sourceByAppPath[runeDir].AchievementFile)
	assert.Equal(t, "achievements.ini", service.sourceByAppPath[pendingDir].AchievementFile)
}

func TestScanAndWatchPrefix_DoesNotAutoDiscoverTenokeStyleGameDirectory(t *testing.T) {
	tempDir := t.TempDir()
	prefixDir := filepath.Join(tempDir, "customprefix")

	tenokeDir := filepath.Join(prefixDir, "drive_c", "Games", "Any Game", "12345")
	require.NoError(t, os.MkdirAll(tenokeDir, 0755))
	writeAchievementINI(t, tenokeDir)

	achMock := &mockAchManager{}
	service := &Service{
		Ach:   achMock,
		Steam: &mockSteam{},
		Config: &config.File{
			Emulators: []config.Emulator{{ID: "codex"}},
		},
	}
	service.watcher = createTestWatcher(t)

	service.scanAndWatchPrefix(prefixDir)

	assert.NotContains(t, service.watcher.WatchList(), tenokeDir)
	assert.Equal(t, 0, achMock.saveCalls)
}
