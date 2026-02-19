package steam

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type Achievement struct {
	Name         string
	DisplayName  string
	Description  string
	Icon         string
	IconGray     string
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
		List  []Achievement
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
			achievementsList, err := fetchAchievementsFromThirdParty(id)

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

func fetchAchievementsWithKey(appID string) ([]Achievement, error) {
	apiKey := os.Getenv("STEAM_API_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf("STEAM_API_KEY environment variable not set")
	}

	//TODO match language to config and system language or default to english

	url := fmt.Sprintf(
		"https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=english",
		apiKey, appID,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API returned status: %d", resp.StatusCode)
	}

	var schema schemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		return nil, err
	}

	var achievements []Achievement
	for _, a := range schema.Game.AvailableGameStats.Achievements {
		achievements = append(achievements, Achievement{
			Name:         a.Name,
			DisplayName:  a.DisplayName,
			Description:  a.Description,
			Icon:         a.Icon,
			IconGray:     a.IconGray,
			DefaultValue: a.DefaultValue,
			Hidden:       a.Hidden,
		})
	}

	return achievements, nil
}

func fetchAchievementsFromThirdParty(appID string) ([]Achievement, error) {
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
	var achievements []Achievement
	for _, item := range shItems {
		a := Achievement{
			Name:        item.ApiName,
			DisplayName: item.Name,
			Description: item.Description,
		}

		if data, ok := communityMap[item.Name]; ok {
			a.Icon = data.Icon
			a.Hidden = data.Hidden
		}

		achievements = append(achievements, a)
	}

	log.Println("Found", len(achievements), "merged achievements for", appID)
	log.Println("Found", achievements[0])
	return achievements, nil
}

func fetchGameBasics(appID string) (*GameBasics, error) {
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

	return &GameBasics{
		AppID:         appID,
		Name:          appData.Data.Name,
		HeaderImage:   appData.Data.HeaderImage,
		CoverImage:    appData.Data.CoverImage,
		PortraitImage: fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID),
		Achievement: struct {
			Total int
			List  []Achievement
		}{
			Total: appData.Data.Achievement.Total,
		},
	}, nil
}

// func cacheIcons(appID string) error {

// }
