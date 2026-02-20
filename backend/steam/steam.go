package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
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
	CoverImage    string
	PortraitImage string
	Achievement   struct {
		Total int
		List  []achievement
	}
}

type gameBasicsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Name          string `json:"name"`
		HeaderImage   string `json:"header_image"`
		CoverImage    string `json:"screenshots[0].path_full"`
		PortraitImage string
		Achievement   struct {
			Total int `json:"total"`
		} `json:"achievements"`
	} `json:"data"`
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

// Language represents a language from Steam
type Language struct {
	DisplayName string `json:"displayName"`
	API         string `json:"api"`
	WebAPI      string `json:"webapi"`
}

// steamLanguages is the list of available Steam languages
var steamLanguages = []Language{
	{DisplayName: "Arabic", API: "arabic", WebAPI: "ar"},
	{DisplayName: "Bulgarian", API: "bulgarian", WebAPI: "bg"},
	{DisplayName: "Simplified Chinese", API: "schinese", WebAPI: "zh-CN"},
	{DisplayName: "Traditional Chinese", API: "tchinese", WebAPI: "zh-TW"},
	{DisplayName: "Czech", API: "czech", WebAPI: "cs"},
	{DisplayName: "Danish", API: "danish", WebAPI: "da"},
	{DisplayName: "Dutch", API: "dutch", WebAPI: "nl"},
	{DisplayName: "English", API: "english", WebAPI: "en"},
	{DisplayName: "Finnish", API: "finnish", WebAPI: "fi"},
	{DisplayName: "French", API: "french", WebAPI: "fr"},
	{DisplayName: "German", API: "german", WebAPI: "de"},
	{DisplayName: "Greek", API: "greek", WebAPI: "el"},
	{DisplayName: "Hungarian", API: "hungarian", WebAPI: "hu"},
	{DisplayName: "Italian", API: "italian", WebAPI: "it"},
	{DisplayName: "Japanese", API: "japanese", WebAPI: "ja"},
	{DisplayName: "Korean", API: "koreana", WebAPI: "ko"},
	{DisplayName: "Norwegian", API: "norwegian", WebAPI: "no"},
	{DisplayName: "Polish", API: "polish", WebAPI: "pl"},
	{DisplayName: "Portuguese", API: "portuguese", WebAPI: "pt"},
	{DisplayName: "Portuguese - Brazil", API: "brazilian", WebAPI: "pt-BR"},
	{DisplayName: "Romanian", API: "romanian", WebAPI: "ro"},
	{DisplayName: "Russian", API: "russian", WebAPI: "ru"},
	{DisplayName: "Spanish - Spain", API: "spanish", WebAPI: "es"},
	{DisplayName: "Spanish - Latin America", API: "latam", WebAPI: "es-419"},
	{DisplayName: "Swedish", API: "swedish", WebAPI: "sv"},
	{DisplayName: "Thai", API: "thai", WebAPI: "th"},
	{DisplayName: "Turkish", API: "turkish", WebAPI: "tr"},
	{DisplayName: "Ukrainian", API: "ukrainian", WebAPI: "uk"},
	{DisplayName: "Vietnamese", API: "vietnamese", WebAPI: "vn"},
}

func init() {
	log.Print("Starting GameBasics Init")
	//data := []string{"2358720", "1903340", "3489700"}
	//
	//_, _ = FetchAppDetailsBulk(data)

}

// GetSteamLanguages returns the list of available Steam languages
func GetSteamLanguages() []Language {
	return steamLanguages
}

func FetchAppDetailsBulk(appIDs []string) ([]*GameBasics, error) {
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

			// 1. Fetch App Details
			details, err := fetchGameBasics(id)
			if err != nil {
				// Log error possibly? For now, we just skip this app
				return
			}

			// 2. Fetch Achievements
			// Note: We are already holding a slot in the semaphore, so this is safe
			//achievementsList, err := fetchAchievementsWithKey(id)
			achievementsList, err := fetchAchievementsFromThirdParty(id, "english")

			if err == nil {
				details.Achievement.List = achievementsList
			}

			mu.Lock()
			results = append(results, details)
			mu.Unlock()
		}(id)
	}

	wg.Wait()

	return results, nil
}

func fetchAchievementsWithKey(appID string, language string) []achievement {
	// Check cache first
	if cached, err := loadCachedAchievementData(appID, language); err == nil {
		return cached
	}

	apiKey := os.Getenv("STEAM_API_KEY")

	if apiKey == "" {
		log.Fatal("STEAM_API_KEY variable not set")
		return nil
	}

	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=%s",
		apiKey, appID, language,
	)
	log.Printf(url)
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
		_ = cacheAchievementIcon(appID, a.Icon)

		// Extract filename from URL for local path
		parts := strings.Split(a.Icon, "/")
		filename := parts[len(parts)-1]
		iconLocalPath := filepath.Join("cache", "icon", appID, filename)

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

	// Cache the fetched achievement data
	_ = cacheAchievementData(appID, language, achievements)

	return achievements
}

func fetchAchievementsFromThirdParty(appID string, language string) ([]achievement, error) {
	// Check cache first
	if cached, err := loadCachedAchievementData(appID, language); err == nil {
		return cached, nil
	}

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
	communityURL := fmt.Sprintf("https://steamcommunity.com/stats/%s/achievements?l=english", appID)
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
			_ = cacheAchievementIcon(appID, data.Icon)

			// Extract filename from URL for local path
			parts := strings.Split(data.Icon, "/")
			filename := parts[len(parts)-1]
			iconLocalPath := filepath.Join("cache", "icon", appID, filename)
			a.IconLocal = iconLocalPath
		}

		achievements = append(achievements, a)
	}

	// Cache the fetched achievement data
	_ = cacheAchievementData(appID, language, achievements)

	return achievements, nil
}

func fetchGameBasics(appID string) (*GameBasics, error) {
	// Check cache for game images
	headerImagePath, _ := loadCachedGameImage(appID, "header-image")
	coverImagePath, _ := loadCachedGameImage(appID, "cover-image")
	portraitImagePath, _ := loadCachedGameImage(appID, "portrait-image")

	// If all images are cached, we can skip the API call for now
	// Note: We still need to fetch achievement count, so we'll make the API call anyway
	// This is a simplified approach - in a full implementation, we'd cache the full GameBasics object

	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%s&l=english", appID)

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

	if !ok || !appData.Success {
		return nil, fmt.Errorf("failed to fetch metadata for appid: %s", appID)
	}

	portraitImageURL := fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)

	// Cache game images
	_ = cacheGameImage(appID, appData.Data.HeaderImage, "headerImage")
	_ = cacheGameImage(appID, appData.Data.CoverImage, "coverImage")
	_ = cacheGameImage(appID, portraitImageURL, "portraitImage")

	// Use cached paths if available, otherwise use URLs
	headerImage := appData.Data.HeaderImage
	coverImage := appData.Data.CoverImage
	portraitImage := portraitImageURL

	if headerImagePath != "" {
		headerImage = headerImagePath
	}
	if coverImagePath != "" {
		coverImage = coverImagePath
	}
	if portraitImagePath != "" {
		portraitImage = portraitImagePath
	}

	return &GameBasics{
		AppID:         appID,
		Name:          appData.Data.Name,
		HeaderImage:   headerImage,
		CoverImage:    coverImage,
		PortraitImage: portraitImage,
		Achievement: struct {
			Total int
			List  []achievement
		}{
			Total: appData.Data.Achievement.Total,
		},
	}, nil
}

// Cache helper functions
func getCacheDir() string {
	p3, _ := os.UserCacheDir()
	return filepath.Join(p3, "sentinel", "cache")
}

func getAchievementCachePath(appID string, language string) string {
	return filepath.Join(getCacheDir(), "schema", language, appID+".json")
}

func getIconCachePath(appID string, filename string) string {
	return filepath.Join(getCacheDir(), "icon", appID, filename)
}

func getGameImageCachePath(appID string, imageType string) string {
	return filepath.Join(getCacheDir(), "icon", appID, imageType)
}

// achievement data caching
func cacheAchievementData(appID string, language string, achievements []achievement) error {
	cachePath := getAchievementCachePath(appID, language)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(achievements, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal achievement data: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write achievement cache: %w", err)
	}

	return nil
}

func loadCachedAchievementData(appID string, language string) ([]achievement, error) {
	cachePath := getAchievementCachePath(appID, language)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var achievements []achievement
	if err := json.Unmarshal(data, &achievements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal achievement data: %w", err)
	}

	return achievements, nil
}

// achievement icon caching
func cacheAchievementIcon(appID string, iconURL string) error {
	if iconURL == "" {
		return nil
	}

	// Extract filename from URL
	parts := strings.Split(iconURL, "/")
	filename := parts[len(parts)-1]

	cachePath := getIconCachePath(appID, filename)

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
func cacheGameImage(appID string, imageURL string, imageType string) error {
	if imageURL == "" {
		return nil
	}

	cachePath := getGameImageCachePath(appID, imageType)

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

func loadCachedGameImage(appID string, imageType string) (string, error) {
	cachePath := getGameImageCachePath(appID, imageType)

	if _, err := os.Stat(cachePath); err != nil {
		return "", err
	}

	return cachePath, nil
}
