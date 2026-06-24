package ach

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sentinel/backend"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var svc = &Service{}

func TestParseAch_ValidFile(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	appID := "123456"
	achievementsDir := filepath.Join(tempDir, appID)
	if err := os.MkdirAll(achievementsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test achievements.json
	achievements := map[string]Achievement{
		"TROPHY_001": {
			Earned:     true,
			EarnedTime: 1744671648,
		},
		"TROPHY_002": {
			Earned:      false,
			EarnedTime:  0,
			MaxProgress: 100,
			Progress:    50,
		},
	}

	data, err := json.MarshalIndent(achievements, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	achievementsPath := filepath.Join(achievementsDir, "achievements.json")
	if err := os.WriteFile(achievementsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse achievements
	result, err := svc.ParseAch(achievementsPath)
	if err != nil {
		t.Fatalf("ParseAch returned error: %v", err)
	}

	// Verify results
	if len(result.Achievements) != 2 {
		t.Errorf("Expected 2 achievements, got %d", len(result.Achievements))
	}

	ach1, ok := result.Achievements["TROPHY_001"]
	if !ok {
		t.Error("TROPHY_001 not found in results")
	} else {
		if !ach1.Earned {
			t.Error("Expected TROPHY_001 to be earned")
		}
		if ach1.EarnedTime != 1744671648 {
			t.Errorf("Expected EarnedTime 1744671648, got %d", ach1.EarnedTime)
		}
	}

	ach2, ok := result.Achievements["TROPHY_002"]
	if !ok {
		t.Error("TROPHY_002 not found in results")
	} else {
		if ach2.Earned {
			t.Error("Expected TROPHY_002 to not be earned")
		}
		if ach2.MaxProgress != 100 {
			t.Errorf("Expected MaxProgress 100, got %d", ach2.MaxProgress)
		}
	}
}

func TestParseAch_MissingFile(t *testing.T) {
	// Try to parse non-existent file
	_, err := svc.ParseAch("/nonexistent/path/achievements.json")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestParseAch_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	achievementsPath := filepath.Join(tempDir, "achievements.json")
	require.NoError(t, os.WriteFile(achievementsPath, []byte("invalid json"), 0644))

	_, err := svc.ParseAch(achievementsPath)
	require.Error(t, err)
}

func TestParseAch_UnsupportedExtension(t *testing.T) {
	tempDir := t.TempDir()
	achievementsPath := filepath.Join(tempDir, "achievements.txt")
	require.NoError(t, os.WriteFile(achievementsPath, []byte("test"), 0644))

	_, err := svc.ParseAch(achievementsPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported achievement file format")
}

func TestParseAch_CodexINI(t *testing.T) {
	tempDir := t.TempDir()
	achievementsPath := filepath.Join(tempDir, "achievements.ini")
	data := []byte(`[ACH02]
Achieved=1
CurProgress=0
MaxProgress=0
UnlockTime=1721215291

[ACH_PROGRESS]
Achieved=0
CurProgress=4
MaxProgress=10
UnlockTime=0

[SteamAchievements]
00000=ACH02
00001=ACH_PROGRESS
Count=2
`)
	require.NoError(t, os.WriteFile(achievementsPath, data, 0644))

	result, err := svc.ParseAch(achievementsPath)
	require.NoError(t, err)

	require.Len(t, result.Achievements, 2)
	assert.True(t, result.Achievements["ACH02"].Earned)
	assert.Equal(t, int64(1721215291), result.Achievements["ACH02"].EarnedTime)
	assert.False(t, result.Achievements["ACH_PROGRESS"].Earned)
	assert.Equal(t, 4, result.Achievements["ACH_PROGRESS"].Progress)
	assert.Equal(t, 10, result.Achievements["ACH_PROGRESS"].MaxProgress)
	assert.NotContains(t, result.Achievements, "SteamAchievements")
}

func TestParseAch_RuneINI_NumericAchievementIDs(t *testing.T) {
	tempDir := t.TempDir()
	achievementsPath := filepath.Join(tempDir, "achievements.ini")
	data := []byte(`[13]
Achieved=1
CurProgress=0
MaxProgress=0
UnlockTime=1724340111

[23]
Achieved=0
CurProgress=2
MaxProgress=5
UnlockTime=0

[SteamAchievements]
00000=13
00001=23
Count=2
`)
	require.NoError(t, os.WriteFile(achievementsPath, data, 0644))

	result, err := svc.ParseAch(achievementsPath)
	require.NoError(t, err)

	require.Len(t, result.Achievements, 2)
	assert.True(t, result.Achievements["13"].Earned)
	assert.Equal(t, int64(1724340111), result.Achievements["13"].EarnedTime)
	assert.False(t, result.Achievements["23"].Earned)
	assert.Equal(t, 2, result.Achievements["23"].Progress)
	assert.Equal(t, 5, result.Achievements["23"].MaxProgress)
}

func TestParseAch_INIIgnoresMalformedLinesAndDefaultsInvalidNumbers(t *testing.T) {
	tempDir := t.TempDir()
	achievementsPath := filepath.Join(tempDir, "achievements.ini")
	data := []byte(`
ignored-before-section=1
; comment
# comment

[ACH_SAFE]
MalformedLine
Achieved=yes
CurProgress=not-a-number
MaxProgress=12
UnlockTime=bad
UnknownKey=999

[SteamAchievements]
00000=ACH_SAFE
Count=1
`)
	require.NoError(t, os.WriteFile(achievementsPath, data, 0644))

	result, err := svc.ParseAch(achievementsPath)
	require.NoError(t, err)

	require.Len(t, result.Achievements, 1)
	achievement := result.Achievements["ACH_SAFE"]
	assert.False(t, achievement.Earned)
	assert.Equal(t, 0, achievement.Progress)
	assert.Equal(t, 12, achievement.MaxProgress)
	assert.Equal(t, int64(0), achievement.EarnedTime)
	assert.NotContains(t, result.Achievements, "SteamAchievements")
}

func TestParseAch_INIMetadataOnlyDoesNotCrash(t *testing.T) {
	tempDir := t.TempDir()
	achievementsPath := filepath.Join(tempDir, "achievements.ini")
	data := []byte(`[SteamAchievements]
Count=0
`)
	require.NoError(t, os.WriteFile(achievementsPath, data, 0644))

	result, err := svc.ParseAch(achievementsPath)
	require.NoError(t, err)
	assert.Empty(t, result.Achievements)
}

func TestSaveAch(t *testing.T) {
	tempDir := t.TempDir()
	appID := "123456"
	achievementsDir := filepath.Join(tempDir, appID)
	if err := os.MkdirAll(achievementsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create achievements.json file
	testAchievements := map[string]Achievement{
		"TROPHY_001": {
			Earned:     true,
			EarnedTime: 1744671648,
		},
	}
	data, err := json.MarshalIndent(testAchievements, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	achievementsPath := filepath.Join(achievementsDir, "achievements.json")
	if err := os.WriteFile(achievementsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Isolate cache directory
	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = t.TempDir()
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	err = svc.SaveAch(achievementsDir)
	if err != nil {
		t.Fatalf("SaveAch returned error: %v", err)
	}

	// Verify file was created in the actual cache directory
	cachePath := filepath.Join(backend.ACHCacheDataDir, appID+".json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Cache file was not created")
	}

	// Verify file contents - matches format used by LoadCachedAch
	fileData, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var cachedAchievements map[string]Achievement
	if err := json.Unmarshal(fileData, &cachedAchievements); err != nil {
		t.Fatalf("Failed to unmarshal cached data: %v", err)
	}

	if len(cachedAchievements) != 1 {
		t.Errorf("Expected 1 achievement in cache, got %d", len(cachedAchievements))
	}

	ach, ok := cachedAchievements["TROPHY_001"]
	if !ok {
		t.Error("TROPHY_001 not found in cached data")
	} else {
		if !ach.Earned {
			t.Error("Expected TROPHY_001 to be earned in cache")
		}
	}

	// Clean up
	os.Remove(cachePath)
}

func TestSaveAch_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	appID := "123456"
	achievementsDir := filepath.Join(tempDir, appID)
	if err := os.MkdirAll(achievementsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create achievements.json file
	testAchievements := map[string]Achievement{
		"TROPHY_001": {
			Earned:     true,
			EarnedTime: 1744671648,
		},
	}
	data, err := json.MarshalIndent(testAchievements, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	achievementsPath := filepath.Join(achievementsDir, "achievements.json")
	if err := os.WriteFile(achievementsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Isolate cache directory to a non-existent path
	oldCacheDir := backend.ACHCacheDataDir
	newCacheDir := filepath.Join(t.TempDir(), "nested", "cache")
	backend.ACHCacheDataDir = newCacheDir
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	err = svc.SaveAch(achievementsDir)
	require.NoError(t, err)

	// Verify directory was created and file was written
	cachePath := filepath.Join(newCacheDir, appID+".json")
	_, err = os.Stat(cachePath)
	require.NoError(t, err, "Cache file should be created in newly created directory")
}

func TestSaveAch_NormalizesINIToJSONCache(t *testing.T) {
	tempDir := t.TempDir()
	appID := "814380"
	achievementsDir := filepath.Join(tempDir, appID)
	require.NoError(t, os.MkdirAll(achievementsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(achievementsDir, "achievements.ini"), []byte(`[ACH02]
Achieved=1
CurProgress=0
MaxProgress=0
UnlockTime=1721215291
`), 0644))

	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = t.TempDir()
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	require.NoError(t, svc.SaveAch(achievementsDir))

	cachePath := filepath.Join(backend.ACHCacheDataDir, appID+".json")
	fileData, err := os.ReadFile(cachePath)
	require.NoError(t, err)

	var cachedAchievements map[string]Achievement
	require.NoError(t, json.Unmarshal(fileData, &cachedAchievements))
	require.Contains(t, cachedAchievements, "ACH02")
	assert.True(t, cachedAchievements["ACH02"].Earned)
	assert.Equal(t, int64(1721215291), cachedAchievements["ACH02"].EarnedTime)
}

func TestLoadCachedAch_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	appID := "123456"

	// Isolate cache directory
	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = tempDir
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	// Create cache file
	cacheData := map[string]Achievement{
		"TROPHY_001": {Earned: true, EarnedTime: 1744671648},
		"TROPHY_002": {Earned: false, Progress: 50, MaxProgress: 100},
	}
	data, err := json.MarshalIndent(cacheData, "", "  ")
	require.NoError(t, err)

	cachePath := filepath.Join(tempDir, appID+".json")
	require.NoError(t, os.WriteFile(cachePath, data, 0644))

	// Load cached achievements
	result, err := svc.LoadCachedAch(appID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Achievements, 2)
	assert.True(t, result.Achievements["TROPHY_001"].Earned)
	assert.Equal(t, 50, result.Achievements["TROPHY_002"].Progress)
}

func TestLoadCachedAch_MissingFile(t *testing.T) {
	tempDir := t.TempDir()

	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = tempDir
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	_, err := svc.LoadCachedAch("nonexistent")
	assert.Error(t, err)
}

func TestLoadAllCachedAch_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = tempDir
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	// Create multiple cache files
	app1Data := map[string]Achievement{"ACH1": {Earned: true}}
	app2Data := map[string]Achievement{"ACH2": {Earned: false, Progress: 75, MaxProgress: 100}}

	data1, _ := json.Marshal(app1Data)
	data2, _ := json.Marshal(app2Data)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "11111.json"), data1, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "22222.json"), data2, 0644))

	// Create non-JSON file (should be skipped)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("test"), 0644))

	result, err := svc.LoadAllCachedAch()
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "11111")
	assert.Contains(t, result, "22222")
}

func TestLoadAllCachedAch_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = tempDir
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	result, err := svc.LoadAllCachedAch()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDiff_NewlyEarned(t *testing.T) {
	current := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: true, EarnedTime: 1000}, // Transitioned from not earned
			"ACH2": {Earned: true, EarnedTime: 2000}, // New achievement, earned
		},
	}
	old := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: false, Progress: 50, MaxProgress: 100},
			// ACH2 doesn't exist in old
		},
	}

	diff := current.Diff(old)

	// Both ACH1 (transitioned) and ACH2 (new and earned) should be newly earned
	assert.Len(t, diff.NewlyEarned, 2)
	assert.Contains(t, diff.NewlyEarned, "ACH1")
	assert.Contains(t, diff.NewlyEarned, "ACH2")
	assert.Empty(t, diff.ProgressUpdated)
}

func TestDiff_ProgressUpdated(t *testing.T) {
	current := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: false, Progress: 75, MaxProgress: 100},
		},
	}
	old := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: false, Progress: 50, MaxProgress: 100},
		},
	}

	diff := current.Diff(old)

	assert.Empty(t, diff.NewlyEarned)
	assert.Len(t, diff.ProgressUpdated, 1)
	assert.Contains(t, diff.ProgressUpdated, "ACH1")
}

func TestDiff_EarnedTransitionTakesPriorityOverProgress(t *testing.T) {
	current := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: true, EarnedTime: 1000, Progress: 10, MaxProgress: 10},
		},
	}
	old := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: false, Progress: 9, MaxProgress: 10},
		},
	}

	diff := current.Diff(old)

	assert.Len(t, diff.NewlyEarned, 1)
	assert.Contains(t, diff.NewlyEarned, "ACH1")
	assert.Empty(t, diff.ProgressUpdated)
}

func TestDiff_NoChanges(t *testing.T) {
	current := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: true, EarnedTime: 1000},
		},
	}
	old := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: true, EarnedTime: 1000},
		},
	}

	diff := current.Diff(old)

	assert.Empty(t, diff.NewlyEarned)
	assert.Empty(t, diff.ProgressUpdated)
}

func TestDiff_NilOld(t *testing.T) {
	current := &AchievementData{
		Achievements: map[string]Achievement{
			"ACH1": {Earned: true},
			"ACH2": {Earned: false},
		},
	}

	diff := current.Diff(nil)

	assert.Len(t, diff.NewlyEarned, 1)
	assert.Contains(t, diff.NewlyEarned, "ACH1")
	assert.Empty(t, diff.ProgressUpdated)
}
