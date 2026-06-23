package steam

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/ach"
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
	assert.Equal(t, "", achievements[0].Icon)
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

func TestFetchAchievementsFromOfficialAPI_CachesFilenameIconsAsLocalMediaPaths(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)

	svc := &Service{Config: mc}
	appID := "12345"
	apiURL := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=12345&language=english"
	iconURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon.png"
	grayURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon_gray.png"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case apiURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"achievements": [
							{
								"internal_name": "ACH_1",
								"localized_name": "Achievement One",
								"localized_desc": "Do the thing",
								"icon": "icon.png",
								"icon_gray": "icon_gray.png",
								"hidden": false
							}
						]
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case iconURL, grayURL:
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

	achievements, err := svc.fetchAchievementsFromOfficialAPI(appID, "english")

	assert.NoError(t, err)
	assert.Len(t, achievements, 1)
	assert.Equal(t, "/api/media/icon/12345/icon.png", achievements[0].Icon)
	assert.Equal(t, "/api/media/icon/12345/icon_gray.png", achievements[0].IconGray)

	_, err = os.Stat(filepath.Join(backend.ACHCacheIconDir, appID, "icon.png"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(backend.ACHCacheIconDir, appID, "icon_gray.png"))
	assert.NoError(t, err)
}

func TestFetchAchievementsFromOfficialAPI_IconDownloadFailureDoesNotReturnRemoteURL(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)

	svc := &Service{Config: mc}
	appID := "12345"
	apiURL := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=12345&language=english"
	iconURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon.png"
	grayURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon_gray.png"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case apiURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"achievements": [
							{
								"internal_name": "ACH_1",
								"localized_name": "Achievement One",
								"localized_desc": "Do the thing",
								"icon": "icon.png",
								"icon_gray": "icon_gray.png",
								"hidden": false
							}
						]
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case iconURL, grayURL:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	achievements, err := svc.fetchAchievementsFromOfficialAPI(appID, "english")

	assert.NoError(t, err)
	assert.Len(t, achievements, 1)
	assert.Equal(t, "", achievements[0].Icon)
	assert.Equal(t, "", achievements[0].IconGray)
	assert.False(t, strings.HasPrefix(achievements[0].Icon, "http"))
	assert.False(t, strings.HasPrefix(achievements[0].IconGray, "http"))
}

func TestFetchAchievementsFromOfficialAPI_CachesAbsoluteIconURLsAsLocalMediaPaths(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)

	svc := &Service{Config: mc}
	appID := "12345"
	apiURL := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=12345&language=english"
	iconURL := "https://cdn.example.com/assets/full_icon.png"
	grayURL := "https://cdn.example.com/assets/full_icon_gray.png"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case apiURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"achievements": [
							{
								"internal_name": "ACH_1",
								"localized_name": "Achievement One",
								"localized_desc": "Do the thing",
								"icon": "https://cdn.example.com/assets/full_icon.png",
								"icon_gray": "https://cdn.example.com/assets/full_icon_gray.png",
								"hidden": false
							}
						]
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case iconURL, grayURL:
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

	achievements, err := svc.fetchAchievementsFromOfficialAPI(appID, "english")

	assert.NoError(t, err)
	assert.Len(t, achievements, 1)
	assert.Equal(t, "/api/media/icon/12345/full_icon.png", achievements[0].Icon)
	assert.Equal(t, "/api/media/icon/12345/full_icon_gray.png", achievements[0].IconGray)
}

func TestRefetchGameData_BypassesExistingCacheAndOverwritesOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)
	mc.On("GetLanguage").Return(types.Language{API: "english"})
	mc.On("GetSteamDataSource").Return(config.SteamSource("key"))

	svc := &Service{Config: mc}
	appID := "12345"
	oldGame := &GameBasics{AppID: appID, Name: "Old Cached Game"}
	assert.NoError(t, svc.cacheGameData(appID, "english", oldGame))

	storeURL := "https://store.steampowered.com/api/appdetails?appids=12345&l=english"
	headerURL := "https://cdn.example.com/header.jpg"
	portraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	apiURL := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=12345&language=english"
	iconURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon.png"
	grayURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon_gray.png"
	storeHit := false

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case storeURL:
			storeHit = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"12345": {
						"data": {
							"name": "Fresh Game",
							"header_image": "https://cdn.example.com/header.jpg"
						}
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case headerURL, portraitURL, iconURL, grayURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		case apiURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"achievements": [
							{
								"internal_name": "ACH_1",
								"localized_name": "Achievement One",
								"localized_desc": "Do the thing",
								"icon": "icon.png",
								"icon_gray": "icon_gray.png",
								"hidden": false
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

	game, err := svc.RefetchGameData(appID)

	assert.NoError(t, err)
	assert.True(t, storeHit)
	assert.Equal(t, "Fresh Game", game.Name)
	assert.Equal(t, 1, game.Achievement.Total)
	assert.Equal(t, "/api/media/icon/12345/headerImage.jpg", game.HeaderImage)
	assert.Equal(t, "/api/media/icon/12345/portraitImage.jpg", game.PortraitImage)
	assert.Equal(t, "/api/media/icon/12345/icon.png", game.Achievement.List[0].Icon)
	assert.Equal(t, "/api/media/icon/12345/icon_gray.png", game.Achievement.List[0].IconGray)

	cached, err := svc.loadCachedGameData(appID, "english")
	assert.NoError(t, err)
	assert.Equal(t, "Fresh Game", cached.Name)
	assert.Equal(t, 1, cached.Achievement.Total)
}

func TestRefetchGameData_GameImageDownloadFailureDoesNotCacheRemoteURLs(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)
	mc.On("GetLanguage").Return(types.Language{API: "english"})
	mc.On("GetSteamDataSource").Return(config.SteamSource("key"))

	svc := &Service{Config: mc}
	appID := "12345"
	storeURL := "https://store.steampowered.com/api/appdetails?appids=12345&l=english"
	headerURL := "https://cdn.example.com/header.jpg"
	portraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	fallbackAPIURL := "https://steam-asset-proxy.steampoacher.workers.dev/?appid=12345"
	apiURL := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=12345&language=english"
	iconURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon.png"
	grayURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon_gray.png"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case storeURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"12345": {
						"data": {
							"name": "Fresh Game",
							"header_image": "https://cdn.example.com/header.jpg"
						}
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case headerURL, portraitURL, fallbackAPIURL:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		case iconURL, grayURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		case apiURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"achievements": [
							{
								"internal_name": "ACH_1",
								"localized_name": "Achievement One",
								"localized_desc": "Do the thing",
								"icon": "icon.png",
								"icon_gray": "icon_gray.png",
								"hidden": false
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

	game, err := svc.RefetchGameData(appID)

	assert.NoError(t, err)
	assert.Equal(t, "", game.HeaderImage)
	assert.Equal(t, "", game.PortraitImage)
	assert.False(t, strings.HasPrefix(game.HeaderImage, "http"))
	assert.False(t, strings.HasPrefix(game.PortraitImage, "http"))

	cached, err := svc.loadCachedGameData(appID, "english")
	assert.NoError(t, err)
	assert.Equal(t, "", cached.HeaderImage)
	assert.Equal(t, "", cached.PortraitImage)
}

func TestRefetchGameData_ReturnsCachedAchievementProgress(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	originalAchDataDir := backend.ACHCacheDataDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	backend.ACHCacheDataDir = filepath.Join(tmpDir, "data")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
		backend.ACHCacheDataDir = originalAchDataDir
	})

	mc := new(mockConfig)
	mc.On("GetLanguage").Return(types.Language{API: "english"})
	mc.On("GetSteamDataSource").Return(config.SteamSource("key"))

	svc := &Service{Config: mc, Ach: &ach.Service{}}
	appID := "12345"

	assert.NoError(t, os.MkdirAll(backend.ACHCacheDataDir, 0755))
	achProgress := map[string]ach.Achievement{
		"ACH_1": {
			Earned:     true,
			EarnedTime: 123,
			Progress:   1,
		},
	}
	progressData, err := json.Marshal(achProgress)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(backend.ACHCacheDataDir, appID+".json"), progressData, 0644))

	storeURL := "https://store.steampowered.com/api/appdetails?appids=12345&l=english"
	headerURL := "https://cdn.example.com/header.jpg"
	portraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	apiURL := "https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=12345&language=english"
	iconURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon.png"
	grayURL := "https://steamcdn-a.akamaihd.net/steamcommunity/public/images/apps/12345/icon_gray.png"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case storeURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"12345": {
						"data": {
							"name": "Fresh Game",
							"header_image": "https://cdn.example.com/header.jpg"
						}
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case headerURL, portraitURL, iconURL, grayURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		case apiURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"achievements": [
							{
								"internal_name": "ACH_1",
								"localized_name": "Achievement One",
								"localized_desc": "Do the thing",
								"icon": "icon.png",
								"icon_gray": "icon_gray.png",
								"hidden": false
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

	game, err := svc.RefetchGameData(appID)

	assert.NoError(t, err)
	assert.True(t, game.Achievement.List[0].CurrentAch.Earned)
	assert.Equal(t, int64(123), game.Achievement.List[0].CurrentAch.EarnedTime)
	assert.Equal(t, 1, game.Achievement.List[0].CurrentAch.Progress)
}

func TestRefetchGameData_FailureLeavesExistingCacheUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)
	mc.On("GetLanguage").Return(types.Language{API: "english"})
	svc := &Service{Config: mc}
	appID := "12345"
	oldGame := &GameBasics{AppID: appID, Name: "Old Cached Game"}
	assert.NoError(t, svc.cacheGameData(appID, "english", oldGame))
	cachePath := svc.getGameCachePath(appID, "english")
	before, err := os.ReadFile(cachePath)
	assert.NoError(t, err)

	storeURL := "https://store.steampowered.com/api/appdetails?appids=12345&l=english"
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case storeURL:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("error")),
				Header:     make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	_, err = svc.RefetchGameData(appID)

	assert.Error(t, err)
	after, readErr := os.ReadFile(cachePath)
	assert.NoError(t, readErr)
	assert.Equal(t, string(before), string(after))
	cached, err := svc.loadCachedGameData(appID, "english")
	assert.NoError(t, err)
	assert.Equal(t, "Old Cached Game", cached.Name)
}

func TestRefetchGameData_RejectsInvalidAppIDBeforeNetwork(t *testing.T) {
	svc := &Service{}
	networkCalled := false

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		networkCalled = true
		return nil, assert.AnError
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	_, err := svc.RefetchGameData("not-an-app")

	assert.Error(t, err)
	assert.False(t, networkCalled)
}

func TestRefetchGameData_UsesConfiguredExternalSourceAndLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)
	mc.On("GetLanguage").Return(types.Language{API: "spanish"})
	mc.On("GetSteamDataSource").Return(config.SteamSource("external"))
	svc := &Service{Config: mc}
	appID := "12345"
	storeURL := "https://store.steampowered.com/api/appdetails?appids=12345&l=spanish"
	headerURL := "https://cdn.example.com/header.jpg"
	portraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	shURL := "https://steamhunters.com/api/apps/12345/achievements"
	communityURL := "https://steamcommunity.com/stats/12345/achievements?l=spanish"
	iconURL := "https://cdn.example.com/external_icon.jpg"
	steamHuntersHit := false

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case storeURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"12345": {
						"data": {
							"name": "Juego Nuevo",
							"header_image": "https://cdn.example.com/header.jpg"
						}
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case headerURL, portraitURL, iconURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		case shURL:
			steamHuntersHit = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{
						"apiName": "EXT_ACH",
						"name": "External Achievement",
						"description": "External description"
					}
				]`)),
				Header: make(http.Header),
			}, nil
		case communityURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`<html><body>
					<div class="achieveRow">
						<div class="achieveImgHolder"><img src="https://cdn.example.com/external_icon.jpg"></div>
						<div class="achieveTxt"><h3>External Achievement</h3><h5>External description</h5></div>
					</div>
				</body></html>`)),
				Header: make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	game, err := svc.RefetchGameData(appID)

	assert.NoError(t, err)
	assert.True(t, steamHuntersHit)
	assert.Equal(t, "Juego Nuevo", game.Name)
	assert.Equal(t, 1, game.Achievement.Total)
	assert.Equal(t, "EXT_ACH", game.Achievement.List[0].Name)
	assert.Equal(t, "/api/media/icon/12345/external_icon.jpg", game.Achievement.List[0].Icon)
	_, err = os.Stat(svc.getGameCachePath(appID, "spanish"))
	assert.NoError(t, err)
}

func TestRefetchGameData_ExternalIconDownloadFailureDoesNotCacheRemoteURL(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	mc := new(mockConfig)
	mc.On("GetLanguage").Return(types.Language{API: "spanish"})
	mc.On("GetSteamDataSource").Return(config.SteamSource("external"))
	svc := &Service{Config: mc}
	appID := "12345"
	storeURL := "https://store.steampowered.com/api/appdetails?appids=12345&l=spanish"
	headerURL := "https://cdn.example.com/header.jpg"
	portraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"
	shURL := "https://steamhunters.com/api/apps/12345/achievements"
	communityURL := "https://steamcommunity.com/stats/12345/achievements?l=spanish"
	iconURL := "https://cdn.example.com/external_icon.jpg"

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case storeURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"12345": {
						"data": {
							"name": "Juego Nuevo",
							"header_image": "https://cdn.example.com/header.jpg"
						}
					}
				}`)),
				Header: make(http.Header),
			}, nil
		case headerURL, portraitURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("image-bytes")),
				Header:     make(http.Header),
			}, nil
		case iconURL:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		case shURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{
						"apiName": "EXT_ACH",
						"name": "External Achievement",
						"description": "External description"
					}
				]`)),
				Header: make(http.Header),
			}, nil
		case communityURL:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`<html><body>
					<div class="achieveRow">
						<div class="achieveImgHolder"><img src="https://cdn.example.com/external_icon.jpg"></div>
						<div class="achieveTxt"><h3>External Achievement</h3><h5>External description</h5></div>
					</div>
				</body></html>`)),
				Header: make(http.Header),
			}, nil
		default:
			return nil, assert.AnError
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	game, err := svc.RefetchGameData(appID)

	assert.NoError(t, err)
	assert.Len(t, game.Achievement.List, 1)
	assert.Equal(t, "", game.Achievement.List[0].Icon)
	assert.False(t, strings.HasPrefix(game.Achievement.List[0].Icon, "http"))

	cached, err := svc.loadCachedGameData(appID, "spanish")
	assert.NoError(t, err)
	assert.Len(t, cached.Achievement.List, 1)
	assert.Equal(t, "", cached.Achievement.List[0].Icon)
}

func TestLoadCachedGameData_DoesNotSelfHealRemotePortraitImage(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	svc := &Service{}
	appID := "12345"
	lang := "english"
	primaryPortraitURL := "https://cdn.akamai.steamstatic.com/steam/apps/12345/library_600x900.jpg"

	game := &GameBasics{
		AppID:         appID,
		Name:          "Test Game",
		PortraitImage: primaryPortraitURL,
	}

	err := svc.cacheGameData(appID, lang, game)
	assert.NoError(t, err)
	cachePath := svc.getGameCachePath(appID, lang)
	before, err := os.ReadFile(cachePath)
	assert.NoError(t, err)

	networkCalled := false
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		networkCalled = true
		return nil, assert.AnError
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	loaded, err := svc.loadCachedGameData(appID, lang)

	assert.NoError(t, err)
	assert.False(t, networkCalled)
	assert.Equal(t, primaryPortraitURL, loaded.PortraitImage)
	_, err = os.Stat(filepath.Join(backend.ACHCacheIconDir, appID, "portraitImage.jpg"))
	assert.Error(t, err)
	after, err := os.ReadFile(cachePath)
	assert.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestLoadCachedGameData_DoesNotSelfHealMissingLocalPortraitImage(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	svc := &Service{}
	appID := "12345"
	lang := "english"
	stalePortraitPath := "/api/media/icon/12345/portraitImage.jpg"

	game := &GameBasics{
		AppID:         appID,
		Name:          "Test Game",
		PortraitImage: stalePortraitPath,
	}

	err := svc.cacheGameData(appID, lang, game)
	assert.NoError(t, err)
	cachePath := svc.getGameCachePath(appID, lang)
	before, err := os.ReadFile(cachePath)
	assert.NoError(t, err)

	networkCalled := false
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		networkCalled = true
		return nil, assert.AnError
	})
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	loaded, err := svc.loadCachedGameData(appID, lang)

	assert.NoError(t, err)
	assert.False(t, networkCalled)
	assert.Equal(t, stalePortraitPath, loaded.PortraitImage)

	portraitCachePath := filepath.Join(backend.ACHCacheIconDir, appID, "portraitImage.jpg")
	_, err = os.Stat(portraitCachePath)
	assert.Error(t, err)
	after, err := os.ReadFile(cachePath)
	assert.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestLoadCachedGameData_NormalizesLocalMediaPathsInMemoryOnly(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := backend.DataDir
	originalGameCacheDir := backend.GameCacheDir
	originalIconDir := backend.ACHCacheIconDir
	backend.DataDir = tmpDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tmpDir, "icon")
	t.Cleanup(func() {
		backend.DataDir = originalDataDir
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconDir
	})

	svc := &Service{}
	appID := "12345"
	lang := "english"
	headerPath := filepath.Join(tmpDir, "icon", appID, "headerImage.jpg")
	portraitPath := filepath.Join(tmpDir, "icon", appID, "portraitImage.jpg")
	externalPath := filepath.Join(string(filepath.Separator), "outside", "icon.png")
	game := &GameBasics{
		AppID:         appID,
		Name:          "Test Game",
		HeaderImage:   headerPath,
		PortraitImage: portraitPath,
		Achievement: struct {
			Total int
			List  []achievement
		}{
			List: []achievement{
				{Icon: filepath.Join(tmpDir, "icon", appID, "icon.png"), IconGray: externalPath},
			},
		},
	}

	assert.NoError(t, svc.cacheGameData(appID, lang, game))
	cachePath := svc.getGameCachePath(appID, lang)
	before, err := os.ReadFile(cachePath)
	assert.NoError(t, err)

	loaded, err := svc.loadCachedGameData(appID, lang)

	assert.NoError(t, err)
	assert.Equal(t, "/api/media/icon/12345/headerImage.jpg", loaded.HeaderImage)
	assert.Equal(t, "/api/media/icon/12345/portraitImage.jpg", loaded.PortraitImage)
	assert.Equal(t, "/api/media/icon/12345/icon.png", loaded.Achievement.List[0].Icon)
	assert.Equal(t, externalPath, loaded.Achievement.List[0].IconGray)
	after, err := os.ReadFile(cachePath)
	assert.NoError(t, err)
	assert.Equal(t, string(before), string(after))
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
