package steam

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"sentinel/backend/steam/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type mockConfig struct {
	mock.Mock
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (m *mockConfig) GetSteamAPIKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *mockConfig) GetSteamDataSource() config.SteamSource {
	args := m.Called()
	return args.Get(0).(config.SteamSource)
}

func (m *mockConfig) GetLanguage() types.Language {
	args := m.Called()
	return args.Get(0).(types.Language)
}

func TestPathHelpers(t *testing.T) {
	svc := &Service{}
	appID := "12345"
	lang := "english"

	backend.GameCacheDir = "/tmp/games"
	backend.ACHCacheIconDir = "/tmp/icons"

	assert.Equal(t, "/tmp/games/english/12345.json", svc.getGameCachePath(appID, lang))
	assert.Equal(t, "/tmp/icons/12345/icon.png", svc.getIconCachePath(appID, "icon.png"))
	assert.Equal(t, "/tmp/icons/12345/header.jpg", svc.getGameImageCachePath(appID, "header.jpg"))
}

func TestResponseParsing_GameDetails(t *testing.T) {
	rawJSON := `{
		"12345": {
			"success": true,
			"data": {
				"name": "Test Game",
				"header_image": "http://example.com/header.jpg"
			}
		}
	}`

	var data map[string]gameBasicsResponse
	err := json.Unmarshal([]byte(rawJSON), &data)
	assert.NoError(t, err)

	appData, ok := data["12345"]
	assert.True(t, ok)
	assert.Equal(t, "Test Game", appData.Data.Name)
}

func TestMergeAchievements(t *testing.T) {
	svc := &Service{}
	appID := "12345"

	shItems := []struct {
		ApiName     string `json:"apiName"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}{
		{ApiName: "ACH_1", Name: "Trophy 1", Description: "Desc 1"},
		{ApiName: "ACH_2", Name: "Trophy 2", Description: "Desc 2"},
	}

	communityMap := map[string]communityData{
		"Trophy 1": {Icon: "http://example.com/icon1.png", Hidden: 0},
		"Trophy 2": {Icon: "http://example.com/icon2.png", Hidden: 1},
	}

	achievements := svc.mergeAchievements(shItems, communityMap, appID)

	assert.Len(t, achievements, 2)
	assert.Equal(t, "ACH_1", achievements[0].Name)
	assert.Equal(t, "http://example.com/icon1.png", achievements[0].Icon)
	assert.Equal(t, 0, achievements[0].Hidden)

	assert.Equal(t, "ACH_2", achievements[1].Name)
	assert.Equal(t, 1, achievements[1].Hidden)
}

func TestCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	backend.GameCacheDir = filepath.Join(tmpDir, "games")

	svc := &Service{}
	appID := "12345"
	lang := "english"
	game := &GameBasics{
		AppID: appID,
		Name:  "Test Game",
	}

	// Test caching
	err := svc.cacheGameData(appID, lang, game)
	assert.NoError(t, err)

	// Test loading
	loaded, err := svc.loadCachedGameData(appID, lang)
	assert.NoError(t, err)
	assert.Equal(t, "Test Game", loaded.Name)
	assert.Equal(t, appID, loaded.AppID)
}

func TestFetchAppDetailsBulk_Cached(t *testing.T) {
	tmpDir := t.TempDir()
	backend.GameCacheDir = filepath.Join(tmpDir, "games")

	mc := new(mockConfig)
	svc := &Service{Config: mc}

	appID := "12345"
	lang := types.Language{API: "english"}
	gameData := `{"AppID": "12345", "Name": "Test Game"}`

	cachePath := filepath.Join(backend.GameCacheDir, lang.API)
	os.MkdirAll(cachePath, 0755)
	os.WriteFile(filepath.Join(cachePath, appID+".json"), []byte(gameData), 0644)

	results, err := svc.FetchAppDetailsBulk([]string{appID}, lang)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Test Game", results[0].Name)
}

func TestLoadCachedGameData_SelfHealsRemotePortraitImage(t *testing.T) {
	tmpDir := t.TempDir()
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")

	svc := &Service{}
	appID := "12345"
	lang := "english"
	primaryPortraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	fallbackAPIURL := "https://steam-asset-proxy.steampoacher.workers.dev/?appid=12345"
	fallbackPortraitURL := "https://shared.steamstatic.com/store_item_assets/steam/apps/12345/fallback-capsule.jpg"

	game := &GameBasics{
		AppID:         appID,
		Name:          "Test Game",
		PortraitImage: primaryPortraitURL,
	}

	err := svc.cacheGameData(appID, lang, game)
	assert.NoError(t, err)

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case primaryPortraitURL:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		case fallbackAPIURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"assets": {
						"library_capsule": "fallback-capsule.jpg"
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case fallbackPortraitURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	loaded, err := svc.loadCachedGameData(appID, lang)

	assert.NoError(t, err)
	assert.Equal(t, "/api/media/icon/12345/portraitImage.jpg", loaded.PortraitImage)

	portraitCachePath := filepath.Join(backend.ACHCacheIconDir, appID, "portraitImage.jpg")
	_, err = os.Stat(portraitCachePath)
	assert.NoError(t, err)

	reloaded, err := svc.loadCachedGameData(appID, lang)
	assert.NoError(t, err)
	assert.Equal(t, "/api/media/icon/12345/portraitImage.jpg", reloaded.PortraitImage)
}

func TestLoadCachedGameData_SelfHealsMissingLocalPortraitImage(t *testing.T) {
	tmpDir := t.TempDir()
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")

	svc := &Service{}
	appID := "12345"
	lang := "english"
	stalePortraitPath := "/api/media/icon/12345/portraitImage.jpg"
	primaryPortraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	fallbackAPIURL := "https://steam-asset-proxy.steampoacher.workers.dev/?appid=12345"
	fallbackPortraitURL := "https://shared.steamstatic.com/store_item_assets/steam/apps/12345/fallback-capsule.jpg"

	game := &GameBasics{
		AppID:         appID,
		Name:          "Test Game",
		PortraitImage: stalePortraitPath,
	}

	err := svc.cacheGameData(appID, lang, game)
	assert.NoError(t, err)

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case primaryPortraitURL:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		case fallbackAPIURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"assets": {
						"library_capsule": "fallback-capsule.jpg"
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case fallbackPortraitURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	loaded, err := svc.loadCachedGameData(appID, lang)

	assert.NoError(t, err)
	assert.Equal(t, stalePortraitPath, loaded.PortraitImage)

	portraitCachePath := filepath.Join(backend.ACHCacheIconDir, appID, "portraitImage.jpg")
	_, err = os.Stat(portraitCachePath)
	assert.NoError(t, err)
}

func TestFallbackPortraitURL_UsesNestedStoreItemsAssets(t *testing.T) {
	svc := &Service{}
	appID := "3764200"
	fallbackAPIURL := "https://steam-asset-proxy.steampoacher.workers.dev/?appid=3764200"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case fallbackAPIURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"store_items": [
							{
								"assets": {
									"library_capsule": "ed3b2cae7d15f598f41006f5f1e605ec5517b5e4/library_capsule.jpg"
								}
							}
						]
					}
				}`)),
				Header: make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	fallbackURL := svc.fallbackPortraitURL(appID)

	assert.Equal(
		t,
		"https://shared.steamstatic.com/store_item_assets/steam/apps/3764200/ed3b2cae7d15f598f41006f5f1e605ec5517b5e4/library_capsule.jpg",
		fallbackURL,
	)
}
