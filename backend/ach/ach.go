package ach

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sentinel/backend"
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
func LoadCachedAch(appId string) (*AchievementData, error) {
	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var achData AchievementData
	if err := json.Unmarshal(data, &achData); err != nil {
		return nil, err
	}

	return &achData, nil
}

// SaveAch saves the given achievements to the cache
func SaveAch(appId string, data *AchievementData) error {
	if err := os.MkdirAll(backend.ACHCacheDataDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, jsonData, 0644)
}

// Diff compares two AchievementData and returns a map of newly earned achievements
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
