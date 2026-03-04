package ach

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"strings"
)

// Achievement represents a single achievement's progress
type Achievement struct {
	Earned      bool  `json:"earned"`
	EarnedTime  int64 `json:"earned_time"`
	MaxProgress int   `json:"max_progress,omitempty"`
	Progress    int   `json:"progress,omitempty"`
}

// AchievementData contains all achievements for a game
type AchievementData struct {
	Achievements map[string]Achievement `json:"achievements"`
}

// ParseAch reads and parses achievements.json from the given path without saving to cache

// wails:internal
func ParseAch(path string) (*AchievementData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var achievements map[string]Achievement
	if err := json.Unmarshal(data, &achievements); err != nil {
		return nil, err
	}

	return &AchievementData{Achievements: achievements}, nil
}

// LoadCachedAch loads the cached achievement data for a given appId
// wails:internal
func LoadCachedAch(appId string) (*AchievementData, error) {
	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")
	data, err := os.ReadFile(cachePath)

	if err != nil {
		return nil, err
	}

	var achievements map[string]Achievement
	if err := json.Unmarshal(data, &achievements); err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	return &AchievementData{Achievements: achievements}, nil
}

// LoadAllCachedAch loads all cached achievement data from the cache directory
// wails: internal
func LoadAllCachedAch() (map[string]*AchievementData, error) {
	files, err := os.ReadDir(backend.ACHCacheDataDir)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*AchievementData)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		appId := strings.TrimSuffix(file.Name(), ".json")

		achData, err := LoadCachedAch(appId)

		if err != nil {
			log.Printf("failed to load cached ach for %s: %v", appId, err)
			continue
		}

		result[appId] = achData
	}

	return result, nil
}

// SaveAch saves the given achievements to the cache
// wails:internal
func SaveAch(path string) error {
	if err := os.MkdirAll(backend.ACHCacheDataDir, 0755); err != nil {
		return err
	}

	// Extract filename from URL
	parts := strings.Split(path, string(os.PathSeparator))
	appId := parts[len(parts)-1]

	file, err := os.ReadFile(filepath.Join(path, "achievements.json"))
	if err != nil {
		return err
	}

	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")

	if _, err := os.Stat(cachePath); err == nil {
		return nil
	}

	return os.WriteFile(cachePath, file, 0644)
}

// Diff compares two AchievementData and returns a map of newly earned achievements

// wails:internal
func (a *AchievementData) Diff(old *AchievementData) map[string]Achievement {
	newlyEarned := make(map[string]Achievement)
	if old == nil {
		for id, ach := range a.Achievements {
			if ach.Earned {
				newlyEarned[id] = ach
			}
		}
		return newlyEarned
	}

	for id, ach := range a.Achievements {
		oldAch, exists := old.Achievements[id]
		if !exists {
			if ach.Earned {
				newlyEarned[id] = ach
			}
			continue
		}
		if ach.Earned && !oldAch.Earned {
			newlyEarned[id] = ach
		}
	}
	return newlyEarned
}
