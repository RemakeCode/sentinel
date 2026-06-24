package ach

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"strconv"
	"strings"
)

type Service struct{}

func (s *Service) Start(ctx context.Context) error {
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

// ParseAch reads and parses an achievement file from the given path without saving to cache.
func (s *Service) ParseAch(path string) (*AchievementData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return parseJSONAchievements(data)
	case ".ini":
		return parseSteamEmuINIAchievements(data)
	default:
		return nil, fmt.Errorf("unsupported achievement file format: %s", path)
	}
}

func parseJSONAchievements(data []byte) (*AchievementData, error) {
	var achievements map[string]Achievement
	if err := json.Unmarshal(data, &achievements); err != nil {
		return nil, err
	}

	return &AchievementData{Achievements: achievements}, nil
}

func parseSteamEmuINIAchievements(data []byte) (*AchievementData, error) {
	achievements := make(map[string]Achievement)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if currentSection != "" && !isMetadataSection(currentSection) {
				if _, ok := achievements[currentSection]; !ok {
					achievements[currentSection] = Achievement{}
				}
			}
			continue
		}

		if currentSection == "" || isMetadataSection(currentSection) {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		achievement := achievements[currentSection]
		switch strings.TrimSpace(key) {
		case "Achieved":
			achievement.Earned = strings.TrimSpace(value) == "1"
		case "CurProgress":
			achievement.Progress = parseIntValue(value)
		case "MaxProgress":
			achievement.MaxProgress = parseIntValue(value)
		case "UnlockTime":
			achievement.EarnedTime, _ = strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		}
		achievements[currentSection] = achievement
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &AchievementData{Achievements: achievements}, nil
}

func isMetadataSection(section string) bool {
	return strings.EqualFold(section, "SteamAchievements")
}

func parseIntValue(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
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

	appId := filepath.Base(path)

	achData, err := s.parseFromDirectory(path)
	if err != nil {
		return err
	}

	file, err := json.MarshalIndent(achData.Achievements, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(backend.ACHCacheDataDir, appId+".json")

	return os.WriteFile(cachePath, file, 0644)
}

func (s *Service) parseFromDirectory(path string) (*AchievementData, error) {
	candidates := []string{
		filepath.Join(path, "achievements.json"),
		filepath.Join(path, "achievements.ini"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return s.ParseAch(candidate)
		}
	}

	return nil, os.ErrNotExist
}

// Diff compares two AchievementData and returns newly earned achievements and progress updates
//
//wails:internal
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
