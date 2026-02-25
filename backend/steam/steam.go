package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sentinel/backend"
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
	IconLocal    string
	DefaultValue int `default:"0"`
	Hidden       int `default:"0"`
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
func (s *GameBasics) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	c, err := config.Get()
	if err != nil {
		return err
	}
	cfg = c
	log.Println("steam: service startup complete")
	return nil
}

//wails:internal
func (s *GameBasics) FetchAppDetailsBulk(appIDs []string, language types.Language) ([]*GameBasics, error) {
	if len(appIDs) == 0 {
		return []*GameBasics{}, nil
	}

	var results []*GameBasics
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Use a semaphore to limit total concurrent requests (both app details and achievements)
	sem := make(chan struct{}, 5)

	for _, id := range appIDs {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release

			// Check unified cache first
			if cached, err := s.loadCachedGameData(id, language.API); err == nil {
				mu.Lock()
				results = append(results, cached)
				mu.Unlock()
				return
			}

			// 1. Fetch App Details
			details, err := s.fetchGameDetails(id, language.API)
			if err != nil {
				// Log error possibly? For now, we just skip this app
				return
			}

			// 2. Fetch Achievements
			achievementsList, err := s.fetchAchievements(id, language.API)

			if err == nil {
				details.Achievement.List = achievementsList
			}

			// Cache the unified complete GameBasics record
			_ = s.cacheGameData(id, language.API, details)

			mu.Lock()
			results = append(results, details)
			mu.Unlock()
		}(id)
	}

	wg.Wait()

	return results, nil
}

func (s *GameBasics) LoadAllCachedGameData() ([]*GameBasics, error) {
	var cached []*GameBasics
	language := cfg.Language.API

	schemaPath := filepath.Join(backend.GameCacheDir, language)
	dirs, err := os.ReadDir(schemaPath)

	if err != nil {
		slog.Error("Unable to load cached game data for FE")
		return nil, errors.New("unable to load cached game data for FE")
	}

	if len(dirs) == 0 {
		slog.Error(fmt.Sprintf("Game cache directory is empty for %s", language))
		return nil, err
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

		cached = append(cached, &gb)
	}

	return cached, nil

}

func (s *GameBasics) fetchAchievementsWithKey(appID string, language string) []achievement {
	apiKey := cfg.SteamAPIKey

	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=%s",
		apiKey, appID, language,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var schema schemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		return nil
	}

	var achievements []achievement
	for _, a := range schema.Game.AvailableGameStats.Achievements {
		// Cache the achievement icon
		_ = s.cacheAchievementIcon(appID, a.Icon)

		// Extract filename from URL for local path
		parts := strings.Split(a.Icon, "/")
		filename := parts[len(parts)-1]
		iconLocalPath := filename

		achievement := achievement{
			Name:         a.Name,
			DisplayName:  a.DisplayName,
			Description:  a.Description,
			Icon:         a.Icon,
			IconGray:     a.IconGray,
			IconLocal:    iconLocalPath,
			DefaultValue: a.DefaultValue,
			Hidden:       a.Hidden,
		}
		achievements = append(achievements, achievement)
	}

	return achievements
}

func (s *GameBasics) fetchAchievementsFromThirdParty(appID string, language string) ([]achievement, error) {
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

			// Extract filename from URL for local path
			parts := strings.Split(data.Icon, "/")
			filename := parts[len(parts)-1]
			iconLocalPath := filepath.Join("cache", "icon", appID, filename)
			a.IconLocal = iconLocalPath
		}

		achievements = append(achievements, a)
	}

	return achievements, nil
}

// fetchAchievements fetches achievements using the configured data source
// It reads the configuration to determine whether to use Steam Key or External Source
func (s *GameBasics) fetchAchievements(appID string, language string) ([]achievement, error) {
	dataSource := cfg.GetSteamDataSource()

	switch dataSource {
	case "key":
		// Use Steam API key
		achievements := s.fetchAchievementsWithKey(appID, language)
		return achievements, nil
	case "external":
		// Use third-party external source
		return s.fetchAchievementsFromThirdParty(appID, language)
	default:
		// Unknown data source, default to external
		log.Printf("Unknown data source '%s', using external source", dataSource)
		return s.fetchAchievementsFromThirdParty(appID, language)
	}
}

func (s *GameBasics) fetchGameDetails(appID string, language string) (*GameBasics, error) {
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

func (s *GameBasics) getGameCachePath(appID string, language string) string {
	return filepath.Join(backend.GameCacheDir, language, appID+".json")
}

func (s *GameBasics) getIconCachePath(appID string, filename string) string {
	return filepath.Join(backend.ACHCacheIconDir, appID, filename)
}

func (s *GameBasics) getGameImageCachePath(appID string, imageType string) string {
	return filepath.Join(backend.ACHCacheIconDir, appID, imageType)
}

// Game cache persistence
func (s *GameBasics) cacheGameData(appID string, language string, game *GameBasics) error {
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

func (s *GameBasics) loadCachedGameData(appID string, language string) (*GameBasics, error) {
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
func (s *GameBasics) cacheAchievementIcon(appID string, iconURL string) error {
	if iconURL == "" {
		return nil
	}

	// Extract filename from URL
	parts := strings.Split(iconURL, "/")
	filename := parts[len(parts)-1]

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
func (s *GameBasics) cacheGameImage(appID string, imageURL string, imageType string) error {
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

func (s *GameBasics) loadCachedGameImage(appID string, imageType string) (string, error) {
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
