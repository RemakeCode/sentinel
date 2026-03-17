package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/steam/types"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type achievement struct {
	Name         string
	DisplayName  string
	Description  string
	Icon         string
	IconGray     string
	DefaultValue int
	Hidden       int
	CurrentAch   ach.Achievement // or map[string]ach.Achievement if you strictly wanted a map, but since a single achievement has only one progress object, ach.Achievement is better.
}

type GameBasics struct {
	AppID         string
	Name          string
	HeaderImage   string
	PortraitImage string
	Achievement   struct {
		Total int
		List  []achievement
	}
}

type gameBasicsResponse struct {
	Data struct {
		Name          string `json:"name"`
		HeaderImage   string `json:"header_image"`
		PortraitImage string
	}
}

type Service struct {
}

// Response struct for ISteamUserStats/GetSchemaForGame
type schemaResponse struct {
	Game struct {
		GameName           string `json:"gameName"`
		AvailableGameStats struct {
			Achievements []struct {
				Name         string `json:"name"`
				DefaultValue int    `json:"defaultvalue"`
				DisplayName  string `json:"displayName"`
				Hidden       int    `json:"hidden"`
				Description  string `json:"description"`
				Icon         string `json:"icon"`
				IconGray     string `json:"icongray"`
			} `json:"achievements"`
		} `json:"availableGameStats"`
	} `json:"game"`
}

var cfg *config.File

// ServiceStartup implements the Wails service lifecycle hook.
func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	c, err := config.Get()
	if err != nil {
		return err
	}
	cfg = c
	slog.Info("steam: service startup complete")
	return nil
}

//wails:internal
func (s *Service) FetchAppDetailsBulk(appIDs []string, language types.Language) ([]*GameBasics, error) {
	app := application.Get()

	total := len(appIDs)

	// Emit 0% immediately to signal fetch is starting (even if no appIDs)
	app.Event.Emit("sentinel::fetch-status", backend.FetchStatusEvt{Current: 0, Total: total})

	if len(appIDs) == 0 {
		// Emit 100% for "no games" case so frontend knows to load from cache
		app.Event.Emit("sentinel::fetch-status", backend.FetchStatusEvt{Current: 100, Total: 100})
		return []*GameBasics{}, nil
	}

	var results []*GameBasics
	var wg sync.WaitGroup
	var mu sync.Mutex

	var completed int

	app.Event.Emit("sentinel::fetch-status", backend.FetchStatusEvt{Current: 0, Total: total})

	sem := make(chan struct{}, 5)

	for _, id := range appIDs {
		wg.Add(1)

		go func(id string) {

			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if cached, err := s.loadCachedGameData(id, language.API); err == nil {
				mu.Lock()
				results = append(results, cached)
				completed++
				mu.Unlock()
				app.Event.Emit("sentinel::fetch-status", backend.FetchStatusEvt{Current: completed, Total: total})
				return
			}

			details, err := s.fetchGameDetails(id, language.API)
			if err != nil {
				mu.Lock()
				completed++
				mu.Unlock()
				app.Event.Emit("sentinel::fetch-status", backend.FetchStatusEvt{Current: completed, Total: total})
				return
			}

			achievementsList, err := s.fetchAchievements(id, language.API)

			if err == nil {
				details.Achievement.List = achievementsList
			}

			_ = s.cacheGameData(id, language.API, details)

			mu.Lock()
			results = append(results, details)
			completed++
			mu.Unlock()

			app.Event.Emit("sentinel::fetch-status", backend.FetchStatusEvt{Current: completed, Total: total})
		}(id)
	}

	wg.Wait()

	return results, nil
}

// Used in FE
func (s *Service) LoadAllCachedGameData() ([]*GameBasics, error) {
	var cached []*GameBasics
	language := cfg.Language.API

	schemaPath := filepath.Join(backend.GameCacheDir, language)
	dirs, err := os.ReadDir(schemaPath)

	if err != nil {
		slog.Error("Unable to load cached game data for FE")
		return nil, errors.New("unable to load cached game data for FE")
	}

	if len(dirs) == 0 {
		slog.Info(fmt.Sprintf("Game cache directory is empty for %s", language))
		return nil, err
	}

	// Load all cached current Achievements
	allAch, err := ach.LoadAllCachedAch()

	if err != nil {
		slog.Warn("Couldn't load all cached current ach")
	}

	for _, dir := range dirs {
		cachePath := filepath.Join(schemaPath, dir.Name())
		data, err := os.ReadFile(cachePath)

		if err != nil {
			slog.Error("Unable to read cached game data for FE")
			return nil, errors.New("unable to read cached game data for FE")
		}

		var gb GameBasics

		if err := json.Unmarshal(data, &gb); err != nil {
			slog.Error("Unable to unmarshal cached game data for FE")
			return nil, errors.New("unable to unmarshal cached game data for FE")
		}

		// Map achievement data by appId to each GameBasics.Achievement.List element
		if achData, ok := allAch[gb.AppID]; ok {
			for i, a := range gb.Achievement.List {
				if progress, exists := achData.Achievements[a.Name]; exists {
					a.CurrentAch = progress
					gb.Achievement.List[i] = a
				}
			}
		}

		cached = append(cached, &gb)
	}

	return cached, nil

}

func (s *Service) fetchAchievementsWithKey(appID string, language string) ([]achievement, error) {
	apiKey, _ := cfg.GetSteamAPIKey()

	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=%s",
		apiKey, appID, language,
	)

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("steam api returned " + resp.Status)
	}

	var schema schemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		return nil, nil
	}

	var achievements []achievement

	for _, a := range schema.Game.AvailableGameStats.Achievements {
		// Cache the achievement icon
		_ = s.cacheAchievementIcon(appID, a.Icon)

		achievement := achievement{
			Name:         a.Name,
			DisplayName:  a.DisplayName,
			Description:  a.Description,
			Icon:         a.Icon,
			IconGray:     a.IconGray,
			DefaultValue: a.DefaultValue,
			Hidden:       a.Hidden,
		}
		achievements = append(achievements, achievement)
	}

	return achievements, nil
}

func (s *Service) fetchAchievementsFromThirdParty(appID string, language string) ([]achievement, error) {
	// 1. Fetch JSON data from SteamHunters
	shURL := fmt.Sprintf("https://steamhunters.com/api/apps/%s/achievements", appID)
	shReq, err := http.NewRequest("GET", shURL, nil)
	if err != nil {
		return nil, err
	}
	shReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")

	shResp, err := http.DefaultClient.Do(shReq)
	if err != nil {
		return nil, err
	}
	defer shResp.Body.Close()

	if shResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamhunters API returned status: %d", shResp.StatusCode)
	}

	var shItems []struct {
		ApiName     string `json:"apiName"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(shResp.Body).Decode(&shItems); err != nil {
		return nil, fmt.Errorf("failed to parse steamhunters api JSON: %v", err)
	}

	// 2. Fetch HTML from Steam Community to get Icons and Hidden status
	communityURL := fmt.Sprintf("https://steamcommunity.com/stats/%s/achievements?l=%s", appID, language)
	cReq, err := http.NewRequest("GET", communityURL, nil)
	if err != nil {
		return nil, err
	}
	cReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")

	cResp, err := http.DefaultClient.Do(cReq)
	if err != nil {
		return nil, err
	}
	defer cResp.Body.Close()

	if cResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam community returned status: %d", cResp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(cResp.Body)
	if err != nil {
		return nil, err
	}

	type communityData struct {
		Icon   string
		Hidden int
	}
	communityMap := make(map[string]communityData)

	doc.Find(".achieveRow").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Find(".achieveTxt h3").First().Text())
		icon := s.Find(".achieveImgHolder img").First().AttrOr("src", "")
		description := strings.TrimSpace(s.Find(".achieveTxt h5").First().Text())

		hidden := 0
		if description == "" {
			hidden = 1
		}

		if name != "" {
			communityMap[name] = communityData{
				Icon:   icon,
				Hidden: hidden,
			}
		}
	})

	// 3. Merge data
	var achievements []achievement
	for _, item := range shItems {
		a := achievement{
			Name:        item.ApiName,
			DisplayName: item.Name,
			Description: item.Description,
		}

		if data, ok := communityMap[item.Name]; ok {
			a.Icon = data.Icon
			a.Hidden = data.Hidden

			// Cache the achievement icon
			_ = s.cacheAchievementIcon(appID, data.Icon)

		}

		achievements = append(achievements, a)
	}

	return achievements, nil
}

// fetchAchievements fetches achievements using the configured data source
// It reads the configuration to determine whether to use Steam Key or External Source
func (s *Service) fetchAchievements(appID string, language string) ([]achievement, error) {
	dataSource := cfg.GetSteamDataSource()

	switch dataSource {
	case "key":
		// Use Steam API key
		return s.fetchAchievementsWithKey(appID, language)

	case "external":
		// Use third-party external source
		return s.fetchAchievementsFromThirdParty(appID, language)
	default:
		// Unknown data source, default to external
		slog.Warn("Unknown data source '%s', using external source", dataSource)
		return s.fetchAchievementsFromThirdParty(appID, language)
	}
}

func (s *Service) fetchGameDetails(appID string, language string) (*GameBasics, error) {
	// 1. Check unified game cache first
	if cached, err := s.loadCachedGameData(appID, language); err == nil {
		return cached, nil
	}

	// 2. Check cache for game images
	headerImagePath, _ := s.loadCachedGameImage(appID, "header-image")
	portraitImagePath, _ := s.loadCachedGameImage(appID, "portrait-image")

	// If all images are cached, we can skip the API call for now
	// Note: We still need to fetch achievement count, so we'll make the API call anyway
	// This is a simplified approach - in a full implementation, we'd cache the full GameBasics object

	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%s&l=%s", appID, language)

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam api returned status: %d", resp.StatusCode)
	}

	var data map[string]gameBasicsResponse

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	appData, ok := data[appID]

	if !ok {
		return nil, fmt.Errorf("failed to fetch metadata for appid: %s", appID)
	}

	portraitImageURL := fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)

	// Cache game images
	_ = s.cacheGameImage(appID, appData.Data.HeaderImage, "headerImage")
	_ = s.cacheGameImage(appID, portraitImageURL, "portraitImage")

	// Use cached paths if available, otherwise use URLs
	headerImage := appData.Data.HeaderImage
	portraitImage := portraitImageURL

	if headerImagePath != "" {
		headerImage = headerImagePath
	}

	if portraitImagePath != "" {
		portraitImage = portraitImagePath
	}

	return &GameBasics{
		AppID:         appID,
		Name:          appData.Data.Name,
		HeaderImage:   headerImage,
		PortraitImage: portraitImage,
	}, nil
}

func (s *Service) getGameCachePath(appID string, language string) string {
	return filepath.Join(backend.GameCacheDir, language, appID+".json")
}

func (s *Service) getIconCachePath(appID string, filename string) string {
	return filepath.Join(backend.ACHCacheIconDir, appID, filename)
}

func (s *Service) getGameImageCachePath(appID string, imageType string) string {
	return filepath.Join(backend.ACHCacheIconDir, appID, imageType)
}

// Game cache persistence
func (s *Service) cacheGameData(appID string, language string, game *GameBasics) error {
	cachePath := s.getGameCachePath(appID, language)

	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(game, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal game data: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write game cache: %w", err)
	}

	return nil
}

func (s *Service) loadCachedGameData(appID string, language string) (*GameBasics, error) {
	cachePath := s.getGameCachePath(appID, language)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var game GameBasics
	if err := json.Unmarshal(data, &game); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game data: %w", err)
	}

	return &game, nil
}

// achievement icon caching
func (s *Service) cacheAchievementIcon(appID string, iconURL string) error {
	if iconURL == "" {
		return nil
	}

	// Extract filename from URL
	filename := filepath.Base(iconURL)

	cachePath := s.getIconCachePath(appID, filename)

	// Check if already cached
	if _, err := os.Stat(cachePath); err == nil {
		return nil // Already cached
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download the image
	resp, err := http.Get(iconURL)
	if err != nil {
		return fmt.Errorf("failed to download icon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download icon: status %d", resp.StatusCode)
	}

	// Save to cache
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read icon data: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write icon cache: %w", err)
	}

	return nil
}

// Game image caching
func (s *Service) cacheGameImage(appID string, imageURL string, imageType string) error {
	if imageURL == "" {
		return nil
	}

	// Extract extension from image URL
	ext := filepath.Ext(imageURL)
	if ext == "" {
		ext = ".jpg" // Default to .jpg if no extension found
	}

	cachePath := s.getGameImageCachePath(appID, imageType+ext)

	// Check if already cached
	if _, err := os.Stat(cachePath); err == nil {
		return nil // Already cached
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Save to cache
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write image cache: %w", err)
	}

	return nil
}

func (s *Service) loadCachedGameImage(appID string, imageType string) (string, error) {
	cacheDir := s.getGameImageCachePath(appID, "")

	// Search for file with matching prefix and any extension
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), imageType) {
			return filepath.Join(cacheDir, entry.Name()), nil
		}
	}

	return "", errors.New("cached image not found")
}

// GlobalAchievementPercentage represents a single achievement's global unlock rate
type GlobalAchievementPercentage struct {
	Name    string `json:"name"`
	Percent string `json:"percent"`
}

// GetGlobalAchievementPercentages fetches global achievement percentages from Steam API
// This method is exposed to the frontend
func (s *Service) GetGlobalAchievementPercentages(appID string) ([]GlobalAchievementPercentage, error) {
	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v0002/?gameid=%s",
		appID,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch global achievement percentages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam api returned status: %d", resp.StatusCode)
	}

	var data struct {
		AchievementPercentages struct {
			Achievements []GlobalAchievementPercentage `json:"achievements"`
		} `json:"achievementpercentages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data.AchievementPercentages.Achievements, nil
}
