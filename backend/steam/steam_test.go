package steam

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"sentinel/backend/steam/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type mockConfig struct {
	mock.Mock
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
