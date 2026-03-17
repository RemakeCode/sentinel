package ach

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sentinel/backend"
)

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
	result, err := ParseAch(achievementsPath)
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
		if ach2.Progress != 50 {
			t.Errorf("Expected Progress 50, got %d", ach2.Progress)
		}
	}
}

func TestParseAch_MissingFile(t *testing.T) {
	// Try to parse non-existent file
	_, err := ParseAch("/nonexistent/path/achievements.json")
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

	testAchievements := map[string]Achievement{
		"TROPHY_001": {
			Earned:     true,
			EarnedTime: 1744671648,
		},
	}
	data, _ := json.MarshalIndent(testAchievements, "", "  ")
	achievementsPath := filepath.Join(achievementsDir, "achievements.json")
	if err := os.WriteFile(achievementsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := SaveAch(achievementsDir)
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

	emptyData, _ := json.MarshalIndent(map[string]Achievement{}, "", "  ")
	achievementsPath := filepath.Join(achievementsDir, "achievements.json")
	if err := os.WriteFile(achievementsPath, emptyData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := SaveAch(achievementsDir)
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
