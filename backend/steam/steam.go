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
	"time"

	"github.com/PuerkitoBio/goquery"
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

type LibrarySyncStatus struct {
	State   string
	Current uint32
	Total   uint32
}

type gameBasicsResponse struct {
	Data struct {
		Name          string `json:"name"`
		HeaderImage   string `json:"header_image"`
		PortraitImage string
	} `json:"data"`
}

type communityData struct {
	Icon   string
	Hidden int
}

// GlobalAchievementPercentage represents a single achievement's global unlock rate
type GlobalAchievementPercentage struct {
	Name    string `json:"name"`
	Percent string `json:"percent"`
}

type Config interface {
	GetSteamDataSource() config.SteamSource
	GetLanguage() types.Language
}

type Service struct {
	Config Config
	Ach    *ach.Service

	syncStatusMu sync.RWMutex
	syncStatus   LibrarySyncStatus

	clientOnce       sync.Once
	client           *http.Client
	assetLimiterOnce sync.Once
	assetLimiter     chan struct{}
}

type assetCacheTask struct {
	Kind    string
	AppID   string
	URL     string
	CacheFn func() error
}

// Response struct for ISteamUserStats/GetSchemaForGame
// NOTE: This API returns empty descriptions for hidden achievements
// Kept for reference; use fetchAchievementsWithKeyLegacy as fallback if needed
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

// Response struct for IPlayerService/GetGameAchievements
// Returns full descriptions including for hidden achievements
type gameAchievementsResponse struct {
	Response struct {
		Achievements []struct {
			Apiname     string `json:"internal_name"`
			DisplayName string `json:"localized_name"`
			Description string `json:"localized_desc"`
			Icon        string `json:"icon"`
			IconGray    string `json:"icon_gray"`
			Hidden      bool   `json:"hidden"`
		} `json:"achievements"`
	} `json:"response"`
}

func (s *Service) Start(ctx context.Context) error {
	if s.Config == nil {
		c, err := config.Get()
		if err != nil {
			return err
		}
		s.Config = c
	}
	slog.Info("Steam service startup complete")
	return nil
}

func (s *Service) RefetchGameData(appID string) (*GameBasics, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, errors.New("appID is required")
	}
	for _, r := range appID {
		if r < '0' || r > '9' {
			return nil, fmt.Errorf("invalid appID: %s", appID)
		}
	}

	language := s.Config.GetLanguage().API
	game, err := s.fetchGameDataFresh(appID, language)
	if err != nil {
		return nil, err
	}

	s.applyCachedAchievementProgress(game)
	return game, nil
}

func (s *Service) GetLibrarySyncStatus() LibrarySyncStatus {
	s.syncStatusMu.RLock()
	defer s.syncStatusMu.RUnlock()

	if s.syncStatus.State == "" {
		return LibrarySyncStatus{State: "idle"}
	}

	return s.syncStatus
}

func (s *Service) startLibrarySync(total uint32) {
	s.syncStatusMu.Lock()
	defer s.syncStatusMu.Unlock()

	s.syncStatus = LibrarySyncStatus{
		State:   "running",
		Current: 0,
		Total:   total,
	}
}

func (s *Service) advanceLibrarySync() LibrarySyncStatus {
	s.syncStatusMu.Lock()
	defer s.syncStatusMu.Unlock()

	if s.syncStatus.State == "" || s.syncStatus.State == "idle" {
		s.syncStatus.State = "running"
	}
	if s.syncStatus.Current < s.syncStatus.Total {
		s.syncStatus.Current++
	}

	return s.syncStatus
}

func (s *Service) completeLibrarySync() {
	s.syncStatusMu.Lock()
	defer s.syncStatusMu.Unlock()

	s.syncStatus.State = "done"
	s.syncStatus.Current = s.syncStatus.Total
}

func (s *Service) failLibrarySync() {
	s.syncStatusMu.Lock()
	defer s.syncStatusMu.Unlock()

	s.syncStatus.State = "error"
}

//wails:internal
func (s *Service) FetchAppDetailsBulk(appIDs []string, language types.Language) ([]*GameBasics, error) {
	total := len(appIDs)
	s.startLibrarySync(uint32(total))

	// Emit 0% immediately to signal fetch is starting (even if no appIDs)
	s.emitFetchStatus(0, uint32(total))

	if len(appIDs) == 0 {
		// Emit 100% for "no games" case so frontend knows to load from cache
		s.completeLibrarySync()
		s.emitFetchStatus(100, 100)
		return []*GameBasics{}, nil
	}

	var results []*GameBasics
	var wg sync.WaitGroup
	var mu sync.Mutex

	var completed uint32

	s.emitFetchStatus(0, uint32(total))

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
				s.advanceLibrarySync()
				s.emitFetchStatus(completed, uint32(total))
				return
			}

			details, err := s.fetchGameDataFresh(id, language.API)
			if err != nil {
				mu.Lock()
				completed++
				mu.Unlock()
				s.advanceLibrarySync()
				s.emitFetchStatus(completed, uint32(total))
				return
			}

			mu.Lock()
			results = append(results, details)
			completed++
			mu.Unlock()

			s.advanceLibrarySync()
			s.emitFetchStatus(completed, uint32(total))
		}(id)
	}

	wg.Wait()
	s.completeLibrarySync()

	return results, nil
}

// Used in FE
func (s *Service) LoadAllCachedGameData() ([]*GameBasics, error) {
	var cached []*GameBasics
	language := s.Config.GetLanguage().API

	slog.Info("Loading cached game data for FE", "language", language)
	schemaPath := filepath.Join(backend.GameCacheDir, language)
	dirs, err := os.ReadDir(schemaPath)

	if err != nil {
		slog.Error("Unable to load cached game data for FE")
		return nil, errors.New("unable to load cached game data for FE")
	}

	if len(dirs) == 0 {
		slog.Info("Game cache directory is empty", "language", language)
		return nil, err
	}

	// Load all cached current Achievements
	allAch, err := s.Ach.LoadAllCachedAch()

	if err != nil {
		slog.Warn("Couldn't load all cached current ach")
	}

	for _, dir := range dirs {
		appID := strings.TrimSuffix(dir.Name(), ".json")
		gb, err := s.loadCachedGameData(appID, language)

		if err != nil {
			slog.Error("Unable to load cached game data for FE", "appID", appID, "error", err)
			continue
		}

		s.applyAchievementProgress(gb, allAch[gb.AppID])

		cached = append(cached, gb)
	}

	return cached, nil
}

func (s *Service) applyCachedAchievementProgress(game *GameBasics) {
	if game == nil || s.Ach == nil {
		return
	}

	achData, err := s.Ach.LoadCachedAch(game.AppID)
	if err != nil {
		slog.Warn("Couldn't load cached current ach", "appID", game.AppID, "error", err)
		return
	}

	s.applyAchievementProgress(game, achData)
}

func (s *Service) applyAchievementProgress(game *GameBasics, achData *ach.AchievementData) {
	if game == nil || achData == nil {
		return
	}

	for i, a := range game.Achievement.List {
		if progress, exists := achData.Achievements[a.Name]; exists {
			a.CurrentAch = progress
			game.Achievement.List[i] = a
		}
	}
}

func (s *Service) httpClient() *http.Client {
	s.clientOnce.Do(func() {
		s.client = &http.Client{Timeout: 15 * time.Second}
	})

	return s.client
}

func (s *Service) assetDownloadLimiter() chan struct{} {
	s.assetLimiterOnce.Do(func() {
		s.assetLimiter = make(chan struct{}, 8)
	})

	return s.assetLimiter
}

func (s *Service) runAssetCacheTasks(tasks []assetCacheTask) {
	seen := make(map[string]struct{}, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		if task.URL == "" {
			continue
		}

		key := task.Kind + "|" + task.AppID + "|" + task.URL
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		wg.Add(1)

		go func(task assetCacheTask) {
			defer wg.Done()

			limiter := s.assetDownloadLimiter()
			limiter <- struct{}{}
			defer func() { <-limiter }()

			if err := task.CacheFn(); err != nil {
				slog.Warn("Failed to cache asset", "kind", task.Kind, "appID", task.AppID, "url", task.URL, "error", err)
			}
		}(task)
	}

	wg.Wait()
}

func (s *Service) fetchAchievementsFromOfficialAPI(appID string, language string) ([]achievement, error) {
	url := fmt.Sprintf(
		"https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=%s&language=%s",
		appID, language,
	)

	resp, err := s.httpClient().Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("steam api returned " + resp.Status)
	}

	var schema gameAchievementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		return nil, err
	}

	var achievements []achievement

	for _, a := range schema.Response.Achievements {
		hiddenVal := 0
		if a.Hidden {
			hiddenVal = 1
		}

		iconPath := s.localizeKeySourceAchievementIcon(appID, a.Icon, "icon")
		iconGrayPath := s.localizeKeySourceAchievementIcon(appID, a.IconGray, "iconGray")

		achievement := achievement{
			Name:        a.Apiname,
			DisplayName: a.DisplayName,
			Description: a.Description,
			Icon:        iconPath,
			IconGray:    iconGrayPath,
			Hidden:      hiddenVal,
		}
		achievements = append(achievements, achievement)
	}

	s.runAssetCacheTasks(s.achievementIconTasks(appID, achievements))
	s.localizeAchievementIcons(appID, achievements)

	return achievements, nil
}

func (s *Service) resolveKeyAchievementIconURL(appID string, icon string) string {
	if icon == "" {
		return ""
	}
	if strings.HasPrefix(icon, "http://") || strings.HasPrefix(icon, "https://") {
		return icon
	}
	steamCDN := fmt.Sprintf("https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/%s/", appID)
	return steamCDN + strings.TrimLeft(icon, "/")
}

func (s *Service) localizeKeySourceAchievementIcon(appID string, icon string, _ string) string {
	iconURL := s.resolveKeyAchievementIconURL(appID, icon)
	if iconURL == "" {
		return ""
	}
	return iconURL
}

func (s *Service) achievementIconTasks(appID string, achievements []achievement) []assetCacheTask {
	tasks := make([]assetCacheTask, 0, len(achievements)*2)

	for _, item := range achievements {
		if item.Icon != "" {
			iconURL := item.Icon
			tasks = append(tasks, assetCacheTask{
				Kind:  "icon",
				AppID: appID,
				URL:   iconURL,
				CacheFn: func() error {
					return s.cacheAchievementIcon(appID, iconURL)
				},
			})
		}

		if item.IconGray != "" {
			iconGrayURL := item.IconGray
			tasks = append(tasks, assetCacheTask{
				Kind:  "iconGray",
				AppID: appID,
				URL:   iconGrayURL,
				CacheFn: func() error {
					return s.cacheAchievementIcon(appID, iconGrayURL)
				},
			})
		}
	}

	return tasks
}

func (s *Service) localizeAchievementIcons(appID string, achievements []achievement) {
	for i, item := range achievements {
		if item.Icon != "" {
			if path, err := s.loadCachedAchievementIcon(appID, item.Icon); err == nil {
				item.Icon = path
			} else {
				item.Icon = ""
			}
		}

		if item.IconGray != "" {
			if path, err := s.loadCachedAchievementIcon(appID, item.IconGray); err == nil {
				item.IconGray = path
			} else {
				item.IconGray = ""
			}
		}

		achievements[i] = item
	}
}

// fetchAchievementsWithKeyLegacy is a fallback using GetSchemaForGame API
// NOTE: This API returns empty descriptions for hidden achievements
// Use only as fallback if GetGameAchievements fails
// func (s *Service) fetchAchievementsWithKeyLegacy(appID string, language string) ([]achievement, error) {
// 	apiKey, _ := s.Config.GetSteamAPIKey()
//
// 	url := fmt.Sprintf(
// 		"https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=%s",
// 		apiKey, appID, language,
// 	)
//
// 	resp, err := http.Get(url)
//
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()
//
// 	if resp.StatusCode != http.StatusOK {
// 		return nil, errors.New("steam api returned " + resp.Status)
// 	}
//
// 	var schema schemaResponse
// 	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
// 		return nil, nil
// 	}
//
// 	var achievements []achievement
//
// 	for _, a := range schema.Game.AvailableGameStats.Achievements {
// 		// Cache the achievement icon
// 		_ = s.cacheAchievementIcon(appID, a.Icon)
//
// 		achievement := achievement{
// 			Name:         a.Name,
// 			DisplayName:  a.DisplayName,
// 			Description:  a.Description,
// 			Icon:         a.Icon,
// 			IconGray:     a.IconGray,
// 			DefaultValue: a.DefaultValue,
// 			Hidden:       a.Hidden,
// 		}
// 		achievements = append(achievements, achievement)
// 	}
//
// 	return achievements, nil
// }

func (s *Service) fetchAchievementsFromThirdParty(appID string, language string) ([]achievement, error) {
	// 1. Fetch JSON data from SteamHunters
	shURL := fmt.Sprintf("https://steamhunters.com/api/apps/%s/achievements", appID)
	shReq, err := http.NewRequest("GET", shURL, nil)
	if err != nil {
		return nil, err
	}
	shReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")

	shResp, err := s.httpClient().Do(shReq)
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

	cResp, err := s.httpClient().Do(cReq)
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
	achievements := s.mergeAchievements(shItems, communityMap, appID)
	s.applyCommunityIcons(achievements, communityMap)
	s.runAssetCacheTasks(s.achievementIconTasks(appID, achievements))
	s.localizeAchievementIcons(appID, achievements)

	return achievements, nil
}

func (s *Service) mergeAchievements(shItems []struct {
	ApiName     string `json:"apiName"`
	Name        string `json:"name"`
	Description string `json:"description"`
}, communityMap map[string]communityData, appID string) []achievement {
	var achievements []achievement
	for _, item := range shItems {
		a := achievement{
			Name:        item.ApiName,
			DisplayName: item.Name,
			Description: item.Description,
		}

		if data, ok := communityMap[item.Name]; ok {
			a.Hidden = data.Hidden
		}

		achievements = append(achievements, a)
	}

	return achievements
}

func (s *Service) applyCommunityIcons(achievements []achievement, communityMap map[string]communityData) {
	for i, item := range achievements {
		if data, ok := communityMap[item.DisplayName]; ok {
			item.Icon = data.Icon
			achievements[i] = item
		}
	}
}

// fetchAchievements fetches achievements using the configured data source
// It reads the configuration to determine whether to use Steam Key or External Source
func (s *Service) fetchAchievements(appID string, language string) ([]achievement, error) {
	dataSource := s.Config.GetSteamDataSource()

	switch dataSource {
	case "key":
		// Use official Steam API (key is optional)
		return s.fetchAchievementsFromOfficialAPI(appID, language)

	case "external":
		// Use third-party external source
		return s.fetchAchievementsFromThirdParty(appID, language)
	default:
		// Unknown data source, default to external
		slog.Warn("Unknown data source, defaulting to external", "dataSource", dataSource)
		return s.fetchAchievementsFromThirdParty(appID, language)
	}
}

func (s *Service) fetchGameDataFresh(appID string, language string) (*GameBasics, error) {
	details, err := s.fetchGameDetailsFresh(appID, language)
	if err != nil {
		return nil, err
	}

	achievementsList, err := s.fetchAchievements(appID, language)
	if err != nil {
		return nil, err
	}

	details.Achievement.Total = len(achievementsList)
	details.Achievement.List = achievementsList

	if err := s.cacheGameData(appID, language, details); err != nil {
		return nil, err
	}

	return details, nil
}

func (s *Service) fetchGameDetailsFresh(appID string, language string) (*GameBasics, error) {
	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%s&l=%s", appID, language)

	resp, err := s.httpClient().Get(url)

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

	portraitImageURL := s.primaryPortraitImageURL(appID)
	headerImage := ""
	portraitImage := ""

	s.runAssetCacheTasks([]assetCacheTask{
		{
			Kind:  "headerImage",
			AppID: appID,
			URL:   appData.Data.HeaderImage,
			CacheFn: func() error {
				return s.cacheGameImage(appID, appData.Data.HeaderImage, "headerImage")
			},
		},
		{
			Kind:  "portraitImage",
			AppID: appID,
			URL:   portraitImageURL,
			CacheFn: func() error {
				return s.cacheGameImage(appID, portraitImageURL, "portraitImage")
			},
		},
	})

	if headerImagePath, err := s.loadCachedGameImage(appID, "headerImage"); err == nil {
		headerImage = headerImagePath
	}

	if portraitImagePath, err := s.loadCachedGameImage(appID, "portraitImage"); err == nil {
		portraitImage = portraitImagePath
	}

	return &GameBasics{
		AppID:         appID,
		Name:          appData.Data.Name,
		HeaderImage:   s.toVirtualPath(headerImage),
		PortraitImage: s.toVirtualPath(portraitImage),
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

	if game.PortraitImage != "" {
		if localPath, err := s.loadCachedGameImage(appID, "portraitImage"); err == nil {
			game.PortraitImage = localPath
		}
	}

	if strings.HasPrefix(game.HeaderImage, "http") {
		if localPath, err := s.loadCachedGameImage(appID, "headerImage"); err == nil {
			game.HeaderImage = localPath
		}
	}

	for i, a := range game.Achievement.List {
		if strings.HasPrefix(a.Icon, "http") {
			if path, err := s.loadCachedAchievementIcon(appID, a.Icon); err == nil {
				a.Icon = path
			}
		}
		if strings.HasPrefix(a.IconGray, "http") {
			if path, err := s.loadCachedAchievementIcon(appID, a.IconGray); err == nil {
				a.IconGray = path
			}
		}
		game.Achievement.List[i] = a
	}

	game.HeaderImage = s.toVirtualPath(game.HeaderImage)
	game.PortraitImage = s.toVirtualPath(game.PortraitImage)

	for i, a := range game.Achievement.List {
		a.Icon = s.toVirtualPath(a.Icon)
		a.IconGray = s.toVirtualPath(a.IconGray)
		game.Achievement.List[i] = a
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
	resp, err := s.httpClient().Get(iconURL)
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

// Game image caching with fallback support
func (s *Service) cacheGameImage(appID string, imageURL string, imageType string) error {
	if imageURL == "" {
		return nil
	}

	// Strip query parameters from URL to get clean extension
	cleanURL := strings.Split(imageURL, "?")[0]
	ext := filepath.Ext(cleanURL)
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

	// Try to download the image
	err := s.downloadImageToCache(appID, imageURL, cachePath)
	if err != nil {
		// Check if it's a 404 and we should try fallback
		if imageType == "portraitImage" && is404Error(err) {
			slog.Warn("Primary portrait image returned 404, trying fallback", "appID", appID)
			if fallbackURL := s.fallbackPortraitURL(appID); fallbackURL != "" {
				if err := s.downloadImageToCache(appID, fallbackURL, cachePath); err != nil {
					slog.Warn("Failed to cache fallback portrait image", "appID", appID, "error", err)
					return nil // Return nil to avoid blocking; placeholder will be used
				}
				slog.Info("Successfully cached fallback portrait image", "appID", appID)
				return nil
			}
			slog.Warn("No fallback portrait URL available", "appID", appID)
		}
		return err
	}

	return nil
}

func is404Error(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status 404")
}

func (s *Service) primaryPortraitImageURL(appID string) string {
	return fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)
}

func (s *Service) downloadImageToCache(appID string, imageURL string, cachePath string) error {
	// Download the image
	resp, err := s.httpClient().Get(imageURL)
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

func (s *Service) fallbackPortraitURL(appID string) string {
	fallbackAPIURL := fmt.Sprintf("https://steam-asset-proxy.steampoacher.workers.dev/?appid=%s", appID)

	resp, err := s.httpClient().Get(fallbackAPIURL)
	if err != nil {
		slog.Warn("Failed to call fallback API", "appID", appID, "error", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Fallback API returned non-200 status", "appID", appID, "status", resp.StatusCode)
		return ""
	}

	var response struct {
		Assets struct {
			LibraryCapsule string `json:"library_capsule"`
		} `json:"assets"`
		Response struct {
			StoreItems []struct {
				Assets struct {
					LibraryCapsule string `json:"library_capsule"`
				} `json:"assets"`
			} `json:"store_items"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		slog.Warn("Failed to decode fallback API response", "appID", appID, "error", err)
		return ""
	}

	libraryCapsule := response.Assets.LibraryCapsule
	if libraryCapsule == "" && len(response.Response.StoreItems) > 0 {
		libraryCapsule = response.Response.StoreItems[0].Assets.LibraryCapsule
	}

	if libraryCapsule == "" {
		slog.Warn("Fallback API response missing library_capsule asset", "appID", appID)
		return ""
	}

	return fmt.Sprintf("https://shared.steamstatic.com/store_item_assets/steam/apps/%s/%s", appID, libraryCapsule)
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

// GetGlobalAchievementPercentages fetches global achievement percentages from Steam API
// This method is exposed to the frontend
func (s *Service) GetGlobalAchievementPercentages(appID string) ([]GlobalAchievementPercentage, error) {
	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v0002/?gameid=%s",
		appID,
	)

	resp, err := s.httpClient().Get(url)
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

func (s *Service) toVirtualPath(absPath string) string {
	if absPath == "" || !filepath.IsAbs(absPath) || strings.HasPrefix(absPath, "/api/media/") {
		return absPath
	}
	rel, err := filepath.Rel(backend.DataDir, absPath)
	if err != nil {
		return absPath
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return absPath
	}
	return "/api/media/" + filepath.ToSlash(rel)
}

func (s *Service) loadCachedAchievementIcon(appID string, iconURL string) (string, error) {
	if iconURL == "" {
		return "", errors.New("empty icon URL")
	}

	filename := filepath.Base(iconURL)
	cachePath := s.getIconCachePath(appID, filename)

	if _, err := os.Stat(cachePath); err == nil {
		return s.toVirtualPath(cachePath), nil
	}

	return "", errors.New("icon not found")
}
