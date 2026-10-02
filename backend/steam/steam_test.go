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
	"github.com/stretchr/testify/require"
)

func TestMergeAchievements(t *testing.T) {
	svc := &Service{}
	appID := "12345"

	var shItems []steamHuntersAchievement
	require.NoError(t, json.Unmarshal([]byte(`[
		{"apiName":"ACH_1","name":"INTENSE PRECISION ","description":"Desc 1","steamPercentage":9.99,"localPercentage":2},
		{"apiName":"ACH_2","name":"Paranatural Agility ","description":"Desc 2","steamPercentage":10,"localPercentage":1}
	]`), &shItems))

	communityMap := map[string]communityData{
		"intense precision":   {Icon: "http://example.com/icon1.png", Hidden: 0},
		"paranatural agility": {Icon: "http://example.com/icon2.png", Hidden: 1},
	}

	achievements := svc.mergeAchievements(shItems, communityMap, appID)
	svc.applyCommunityIcons(achievements, communityMap)

	assert.Len(t, achievements, 2)
	assert.Equal(t, "ACH_1", achievements[0].Name)
	assert.Equal(t, "INTENSE PRECISION", achievements[0].DisplayName)
	require.NotNil(t, achievements[0].GlobalPercentage)
	assert.Equal(t, 9.99, *achievements[0].GlobalPercentage)
	assert.True(t, achievements[0].IsRare)
	assert.Equal(t, "http://example.com/icon1.png", achievements[0].Icon)
	assert.Equal(t, 0, achievements[0].Hidden)

	assert.Equal(t, "ACH_2", achievements[1].Name)
	assert.Equal(t, "Paranatural Agility", achievements[1].DisplayName)
	require.NotNil(t, achievements[1].GlobalPercentage)
	assert.Equal(t, 10.0, *achievements[1].GlobalPercentage)
	assert.False(t, achievements[1].IsRare)
	assert.Equal(t, "http://example.com/icon2.png", achievements[1].Icon)
	assert.Equal(t, 1, achievements[1].Hidden)
}

func TestCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	originalGameCacheDir := backend.GameCacheDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	t.Cleanup(func() { backend.GameCacheDir = originalGameCacheDir })

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

func TestLoadAllCachedGameData_MissingCacheDirectoryReturnsEmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	originalGameCacheDir := backend.GameCacheDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	t.Cleanup(func() {
		backend.GameCacheDir = originalGameCacheDir
	})

	svc := &Service{Config: &config.File{Language: types.Language{API: "english"}}}

	games, err := svc.LoadAllCachedGameData()

	assert.NoError(t, err)
	assert.NotNil(t, games)
	assert.Empty(t, games)
}

func TestLoadAllCachedGameData_EmptyCacheDirectoryReturnsEmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	originalGameCacheDir := backend.GameCacheDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	t.Cleanup(func() {
		backend.GameCacheDir = originalGameCacheDir
	})

	lang := types.Language{API: "english"}
	assert.NoError(t, os.MkdirAll(filepath.Join(backend.GameCacheDir, lang.API), 0755))

	svc := &Service{Config: &config.File{Language: lang}}

	games, err := svc.LoadAllCachedGameData()

	assert.NoError(t, err)
	assert.NotNil(t, games)
	assert.Empty(t, games)
}

func TestFetchAppDetailsBulk_Cached(t *testing.T) {
	tmpDir := t.TempDir()
	originalGameCacheDir := backend.GameCacheDir
	backend.GameCacheDir = filepath.Join(tmpDir, "games")
	t.Cleanup(func() { backend.GameCacheDir = originalGameCacheDir })

	appID := "12345"
	lang := types.Language{API: "english"}
	svc := &Service{Config: &config.File{Language: lang}}
	gameData := `{"AppID": "12345", "Name": "Test Game"}`

	cachePath := filepath.Join(backend.GameCacheDir, lang.API)
	os.MkdirAll(cachePath, 0755)
	os.WriteFile(filepath.Join(cachePath, appID+".json"), []byte(gameData), 0644)

	results, err := svc.FetchAppDetailsBulk([]string{appID}, lang)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Test Game", results[0].Name)
	assert.Equal(t, LibrarySyncStatus{State: "done", Current: 1, Total: 1}, svc.GetLibrarySyncStatus())
}

func TestLibrarySyncStatus_DefaultsToIdle(t *testing.T) {
	svc := &Service{}

	assert.Equal(t, LibrarySyncStatus{State: "idle", Current: 0, Total: 0}, svc.GetLibrarySyncStatus())
}

func TestLibrarySyncStatus_ProgressAndCompletion(t *testing.T) {
	svc := &Service{}

	svc.startLibrarySync(2)
	assert.Equal(t, LibrarySyncStatus{State: "running", Current: 0, Total: 2}, svc.GetLibrarySyncStatus())

	status := svc.advanceLibrarySync(false)
	assert.Equal(t, LibrarySyncStatus{State: "running", Current: 1, Total: 2}, status)
	assert.Equal(t, LibrarySyncStatus{State: "running", Current: 1, Total: 2}, svc.GetLibrarySyncStatus())

	svc.completeLibrarySync()
	assert.Equal(t, LibrarySyncStatus{State: "done", Current: 2, Total: 2}, svc.GetLibrarySyncStatus())
}

func TestLibrarySyncStatus_Error(t *testing.T) {
	svc := &Service{}

	svc.startLibrarySync(3)
	svc.advanceLibrarySync(false)
	svc.failLibrarySync()

	assert.Equal(t, LibrarySyncStatus{State: "error", Current: 1, Total: 3}, svc.GetLibrarySyncStatus())
}

func TestParseSourcePercentage(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       float64
		wantValid  bool
		wantIsRare bool
	}{
		{name: "valid zero", input: `0`, want: 0, wantValid: true, wantIsRare: true},
		{name: "quoted value", input: `"9.99"`, want: 9.99, wantValid: true, wantIsRare: true},
		{name: "threshold", input: `10`, want: 10, wantValid: true},
		{name: "upper bound", input: `100`, want: 100, wantValid: true},
		{name: "missing"},
		{name: "null", input: `null`},
		{name: "invalid", input: `"invalid"`},
		{name: "below range", input: `-0.1`},
		{name: "above range", input: `100.1`},
		{name: "non-finite", input: `"Inf"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.input != "" {
				raw = json.RawMessage(tt.input)
			}
			percentage := parseSourcePercentage(raw)
			if !tt.wantValid {
				assert.Nil(t, percentage)
				assert.False(t, isRarePercentage(percentage))
				return
			}
			require.NotNil(t, percentage)
			assert.Equal(t, tt.want, *percentage)
			assert.Equal(t, tt.wantIsRare, isRarePercentage(percentage))
		})
	}
}

func TestGetGlobalAchievementPercentagesReadsSavedCache(t *testing.T) {
	originalCacheDir := backend.GameCacheDir
	backend.GameCacheDir = t.TempDir()
	t.Cleanup(func() { backend.GameCacheDir = originalCacheDir })

	zero := 0.0
	rare := 9.99
	normal := 10.0
	svc := &Service{Config: &config.File{Language: types.Language{API: "english"}}}
	game := &GameBasics{
		AppID: "12345",
		Achievement: struct {
			Total int
			List  []achievement
		}{
			List: []achievement{
				{Name: "ZERO", GlobalPercentage: &zero, IsRare: true},
				{Name: "RARE", GlobalPercentage: &rare, IsRare: true},
				{Name: "NORMAL", GlobalPercentage: &normal},
				{Name: "MISSING"},
			},
		},
	}
	require.NoError(t, svc.cacheGameData(game.AppID, "english", game))

	percentages, err := svc.GetGlobalAchievementPercentages(game.AppID)
	require.NoError(t, err)
	require.Len(t, percentages, 3)
	assert.Equal(t, GlobalAchievementPercentage{Name: "ZERO", Percent: "0", IsRare: true}, percentages[0])
	assert.Equal(t, GlobalAchievementPercentage{Name: "RARE", Percent: "9.99", IsRare: true}, percentages[1])
	assert.Equal(t, GlobalAchievementPercentage{Name: "NORMAL", Percent: "10", IsRare: false}, percentages[2])

	legacyPath := svc.getGameCachePath("67890", "english")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0755))
	require.NoError(t, os.WriteFile(legacyPath, []byte(`{"AppID":"67890","Achievement":{"List":[{"Name":"LEGACY"}]}}`), 0644))
	legacy, err := svc.GetGlobalAchievementPercentages("67890")
	require.NoError(t, err)
	assert.Empty(t, legacy)
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
