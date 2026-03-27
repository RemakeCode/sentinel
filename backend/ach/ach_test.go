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
	// Test JSON parsing directly
	invalidJSON := []byte("invalid json")
	var parsedAchievements map[string]Achievement
	err := json.Unmarshal(invalidJSON, &parsedAchievements)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
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

	// Isolate cache directory
	oldCacheDir := backend.ACHCacheDataDir
	backend.ACHCacheDataDir = t.TempDir()
	defer func() { backend.ACHCacheDataDir = oldCacheDir }()

	err = svc.SaveAch(achievementsDir)
	if err != nil {
		t.Fatalf("SaveAch returned error: %v", err)
	}

	// Verify directory was created in the actual cache directory
	if _, err := os.Stat(backend.ACHCacheDataDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}

	// Clean up
	cachePath := filepath.Join(backend.ACHCacheDataDir, appID+".json")
	os.Remove(cachePath)
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
