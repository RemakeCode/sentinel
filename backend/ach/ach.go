package ach

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct{}

func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

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

// AchievementDiff represents the result of comparing two AchievementData snapshots
type AchievementDiff struct {
	NewlyEarned     map[string]Achievement
	ProgressUpdated map[string]Achievement
}

// ParseAch reads and parses achievements.json from the given path without saving to cache
func (s *Service) ParseAch(path string) (*AchievementData, error) {
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
func (s *Service) LoadCachedAch(appId string) (*AchievementData, error) {
	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")
	data, err := os.ReadFile(cachePath)

	if err != nil {
		return nil, err
	}

	var achievements map[string]Achievement
	if err := json.Unmarshal(data, &achievements); err != nil {
		slog.Error("Failed to unmarshal cached achievements", "error", err)
		return nil, err
	}

	return &AchievementData{Achievements: achievements}, nil
}

// LoadAllCachedAch loads all cached achievement data from the cache directory
func (s *Service) LoadAllCachedAch() (map[string]*AchievementData, error) {
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

		achData, err := s.LoadCachedAch(appId)

		if err != nil {
			slog.Error("Failed to load cached achievements", "appId", appId, "error", err)
			continue
		}

		result[appId] = achData
	}

	return result, nil
}

// SaveAch saves the given achievements to the cache
func (s *Service) SaveAch(path string) error {
	if err := os.MkdirAll(backend.ACHCacheDataDir, 0755); err != nil {
		slog.Error("Failed to create achievement cache directory", "error", err)
		return err
	}

	// Extract appId from path
	appId := filepath.Base(path)

	file, err := os.ReadFile(filepath.Join(path, "achievements.json"))
	if err != nil {
		return err
	}

	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")

	return os.WriteFile(cachePath, file, 0644)
}

// Diff compares two AchievementData and returns newly earned achievements and progress updates
// wails:internal
func (a *AchievementData) Diff(old *AchievementData) *AchievementDiff {
	result := &AchievementDiff{
		NewlyEarned:     make(map[string]Achievement),
		ProgressUpdated: make(map[string]Achievement),
	}

	// Handle nil old - all achievements are new
	if old == nil {
		for id, ach := range a.Achievements {
			if ach.Earned {
				result.NewlyEarned[id] = ach
			}
		}
		return result
	}

	for id, ach := range a.Achievements {
		oldAch, exists := old.Achievements[id]
		if !exists {
			if ach.Earned {
				result.NewlyEarned[id] = ach
			}
			continue
		}

		if ach.Earned && !oldAch.Earned {
			result.NewlyEarned[id] = ach
		} else if !ach.Earned && ach.Progress != oldAch.Progress {
			result.ProgressUpdated[id] = ach
		}
	}

	return result
}
