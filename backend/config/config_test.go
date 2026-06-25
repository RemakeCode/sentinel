package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sentinel/backend"
	"sentinel/backend/steam/types"
)

func setupTestConfig(t *testing.T) (*File, string) {
	t.Helper()
	tempDir := t.TempDir()

	// Override config path for testing
	originalPath := backend.ConfigPath
	backend.ConfigPath = filepath.Join(tempDir, "config.json")

	// Reset singleton for testing
	instance = nil
	instanceOnce = sync.Once{}
	instanceErr = nil

	t.Cleanup(func() {
		backend.ConfigPath = originalPath
	})

	return &File{}, tempDir
}

func TestLoadConfig_ValidFile(t *testing.T) {
	_, _ = setupTestConfig(t)

	configData := File{
		Emulators: []Emulator{
			{ID: "gse", ShouldNotify: true},
		},
		Language: types.Language{
			DisplayName: "English",
			API:         "english",
			WebAPI:      "en",
		},
		SteamDataSource: "external",
	}

	// Write test config
	data, err := json.MarshalIndent(configData, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(backend.ConfigPath, data, 0644))

	// Load config
	cfg := &File{}
	result, err := cfg.LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "external", string(result.SteamDataSource))
	assert.Contains(t, emulatorIDs(result.Emulators), "gse")
	assert.Contains(t, emulatorIDs(result.Emulators), "goldberg-steamemu")
	assert.Contains(t, emulatorIDs(result.Emulators), "codex")
	assert.Contains(t, emulatorIDs(result.Emulators), "rune")
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, _ = setupTestConfig(t)

	cfg := &File{}
	_, err := cfg.LoadConfig()
	assert.Error(t, err)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	_, _ = setupTestConfig(t)

	require.NoError(t, os.WriteFile(backend.ConfigPath, []byte("invalid json"), 0644))

	cfg := &File{}
	_, err := cfg.LoadConfig()
	assert.Error(t, err)
}

func TestSaveConfig(t *testing.T) {
	_, _ = setupTestConfig(t)

	cfg := &File{
		Emulators: []Emulator{
			{ID: "codex", ShouldNotify: true},
		},
		Language: types.Language{
			DisplayName: "English",
			API:         "english",
		},
		SteamDataSource: "key",
	}

	err := cfg.SaveConfig()
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(backend.ConfigPath)
	assert.NoError(t, err)

	// Verify content
	loaded := &File{}
	_, err = loaded.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "key", string(loaded.SteamDataSource))
	assert.Contains(t, emulatorIDs(loaded.Emulators), "codex")

	data, err := os.ReadFile(backend.ConfigPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"id": "codex"`)
	assert.NotContains(t, string(data), `"path"`)
}

func TestLoadConfig_AchievementProgressUpdateModeDefaults(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "missing field defaults",
			json: `{"language":{"displayName":"English","api":"english","webapi":"en"},"steamDataSource":"external"}`,
		},
		{
			name: "empty string defaults",
			json: `{"AchievementProgressUpdateMode":"","steamDataSource":"external"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = setupTestConfig(t)
			require.NoError(t, os.WriteFile(backend.ConfigPath, []byte(tt.json), 0644))

			cfg := &File{}
			result, err := cfg.LoadConfig()
			require.NoError(t, err)

			assert.Equal(t, AchievementProgressUpdateModeDefault, result.AchievementProgressUpdateMode)
			assert.Equal(t, AchievementProgressUpdateModeDefault, result.GetAchievementProgressUpdateMode())
		})
	}
}

func TestSaveConfig_DefaultsAchievementProgressUpdateMode(t *testing.T) {
	_, _ = setupTestConfig(t)

	cfg := &File{}
	err := cfg.SaveConfig()
	require.NoError(t, err)

	var raw map[string]any
	data, err := os.ReadFile(backend.ConfigPath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, string(AchievementProgressUpdateModeDefault), raw["achievementProgressUpdateMode"])
}

func TestSetSteamAPIKey(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetSteamAPIKey("TEST_API_KEY_12345")
	require.NoError(t, err)

	// API key should be encrypted (not plain text)
	assert.NotEqual(t, "TEST_API_KEY_12345", cfg.SteamAPIKey)
	// Masked key should show last 4 chars
	assert.Contains(t, cfg.SteamAPIKeyMasked, "2345")
	assert.Contains(t, cfg.SteamAPIKeyMasked, "***")
}

func TestSetSteamAPIKey_EmptyKey(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}
	err := cfg.SetSteamAPIKey("")
	assert.Error(t, err)
}

func TestGetSteamAPIKey(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}
	originalKey := "MY_SECRET_KEY_123"

	// Set and encrypt
	err := cfg.SetSteamAPIKey(originalKey)
	require.NoError(t, err)

	// Get and decrypt
	decrypted, err := cfg.GetSteamAPIKey()
	require.NoError(t, err)
	assert.Equal(t, originalKey, decrypted)
}

func TestGetSteamAPIKey_EmptyKey(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}
	key, err := cfg.GetSteamAPIKey()
	require.NoError(t, err)
	assert.Empty(t, key)
}

func TestGetSteamDataSource(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	tests := []struct {
		name     string
		setValue SteamSource
		expected SteamSource
	}{
		{"returns key when set", Key, Key},
		{"returns external when set", External, External},
		{"returns external as default", "", External},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &File{SteamDataSource: tt.setValue}
			result := cfg.GetSteamDataSource()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetSteamDataSource(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}
	err := cfg.SetSteamDataSource(Key)
	require.NoError(t, err)
	assert.Equal(t, Key, cfg.SteamDataSource)
}

func TestSetAchievementProgressUpdateMode_Valid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}
	err := cfg.SetAchievementProgressUpdateMode(AchievementProgressUpdateModeSilent)
	require.NoError(t, err)

	assert.Equal(t, AchievementProgressUpdateModeSilent, cfg.AchievementProgressUpdateMode)

	loaded := &File{}
	_, err = loaded.LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, AchievementProgressUpdateModeSilent, loaded.AchievementProgressUpdateMode)
}

func TestSetAchievementProgressUpdateMode_Invalid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{AchievementProgressUpdateMode: AchievementProgressUpdateModeDefault}
	err := cfg.SetAchievementProgressUpdateMode(AchievementProgressUpdateMode("loud"))

	assert.Error(t, err)
	assert.Equal(t, AchievementProgressUpdateModeDefault, cfg.AchievementProgressUpdateMode)
}

func TestGetEmulatorPaths(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{ID: "gse"},
			{ID: "goldberg-steamemu"},
			{ID: "codex"},
			{ID: "unknown"},
		},
	}

	paths, err := cfg.GetEmulatorPaths()
	require.NoError(t, err)
	assert.Len(t, paths, 3)
	assert.Contains(t, paths, backend.EmuDir)
	assert.Contains(t, paths, backend.GoldbergSteamEmuDir)
	assert.Contains(t, paths, backend.CodexEmuDir)
}

func TestToggleEmulatorNotification(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{ID: "gse", ShouldNotify: true},
			{ID: "codex", ShouldNotify: false},
		},
	}

	// Toggle first emulator
	err := cfg.ToggleEmulatorNotification(0)
	require.NoError(t, err)
	assert.False(t, cfg.Emulators[0].ShouldNotify)

	// Toggle second emulator
	err = cfg.ToggleEmulatorNotification(1)
	require.NoError(t, err)
	assert.True(t, cfg.Emulators[1].ShouldNotify)
}

func TestToggleEmulatorNotification_InvalidIndex(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{ID: "gse", ShouldNotify: true},
		},
	}

	// Invalid indices should not error
	err := cfg.ToggleEmulatorNotification(-1)
	require.NoError(t, err)
	err = cfg.ToggleEmulatorNotification(100)
	require.NoError(t, err)
}

func TestEncryptDecrypt(t *testing.T) {
	original := "sensitive-data-to-encrypt"

	encrypted, err := encrypt(original)
	require.NoError(t, err)
	assert.NotEqual(t, original, encrypted)

	decrypted, err := decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestEncrypt_EmptyString(t *testing.T) {
	encrypted, err := encrypt("")
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := decrypt(encrypted)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	_, err := decrypt("c2hvcnQ=") // "short" in base64
	assert.Error(t, err)
}

func TestDefaultEmulators(t *testing.T) {
	// Verify default emulators store only stable IDs and notification defaults.
	require.Len(t, defaultEmulators, 4)
	assert.Contains(t, emulatorIDs(defaultEmulators), "gse")
	assert.Contains(t, emulatorIDs(defaultEmulators), "goldberg-steamemu")
	assert.Contains(t, emulatorIDs(defaultEmulators), "codex")
	assert.Contains(t, emulatorIDs(defaultEmulators), "rune")

	for _, emu := range defaultEmulators {
		assert.True(t, emu.ShouldNotify)
		assert.NotEmpty(t, emu.ID)
	}
}

func TestDefaultEmulatorSources(t *testing.T) {
	require.Len(t, defaultEmulatorSources, 4)

	sourcesByID := map[string]EmulatorSource{}
	for _, source := range defaultEmulatorSources {
		sourcesByID[source.ID] = source
	}

	assert.Equal(t, backend.EmuDir, sourcesByID["gse"].Path)
	assert.Equal(t, "achievements.json", sourcesByID["gse"].AchievementFile)

	assert.Equal(t, backend.GoldbergSteamEmuDir, sourcesByID["goldberg-steamemu"].Path)
	assert.Equal(t, "achievements.json", sourcesByID["goldberg-steamemu"].AchievementFile)

	assert.Equal(t, backend.CodexEmuDir, sourcesByID["codex"].Path)
	assert.Equal(t, "achievements.ini", sourcesByID["codex"].AchievementFile)

	assert.Equal(t, backend.RuneEmuDir, sourcesByID["rune"].Path)
	assert.Equal(t, "achievements.ini", sourcesByID["rune"].AchievementFile)
}

func TestLoadConfig_MigratesLegacyPathsAndDropsUnknownEmulators(t *testing.T) {
	_, _ = setupTestConfig(t)

	configJSON := `{
		"emulators": [
			{"path": "AppData/Roaming/GSE Saves", "shouldNotify": false},
			{"path": "/custom/path", "shouldNotify": false}
		],
		"steamDataSource": "external"
	}`
	require.NoError(t, os.WriteFile(backend.ConfigPath, []byte(configJSON), 0644))

	cfg := &File{}
	result, err := cfg.LoadConfig()
	require.NoError(t, err)

	ids := emulatorIDs(result.Emulators)
	assert.ElementsMatch(t, []string{"gse", "goldberg-steamemu", "codex", "rune"}, ids)
	assert.Len(t, result.Emulators, 4)
	assert.False(t, emulatorByID(result.Emulators, "gse").ShouldNotify)
	assert.True(t, emulatorByID(result.Emulators, "goldberg-steamemu").ShouldNotify)
}

func TestMigrateLegacyEmulators_MapsKnownPaths(t *testing.T) {
	cfg := &File{}

	changed := cfg.migrateLegacyEmulators([]legacyEmulator{
		{Path: filepath.Join("AppData", "Roaming", "GSE Saves"), ShouldNotify: false},
		{Path: backend.CodexEmuDir, ShouldNotify: true},
	})

	require.True(t, changed)
	assert.ElementsMatch(t, []string{"gse", "codex"}, emulatorIDs(cfg.Emulators))
	assert.False(t, emulatorByID(cfg.Emulators, "gse").ShouldNotify)
	assert.True(t, emulatorByID(cfg.Emulators, "codex").ShouldNotify)
}

func TestMigrateLegacyEmulators_IgnoresCurrentIDShape(t *testing.T) {
	cfg := &File{
		Emulators: []Emulator{
			{ID: "gse", ShouldNotify: true},
		},
	}

	changed := cfg.migrateLegacyEmulators([]legacyEmulator{
		{ID: "gse", ShouldNotify: true},
	})

	require.False(t, changed)
	assert.Equal(t, []string{"gse"}, emulatorIDs(cfg.Emulators))
	assert.True(t, cfg.Emulators[0].ShouldNotify)
}

func TestLoadConfig_DoesNotDuplicateDefaultEmulators(t *testing.T) {
	_, _ = setupTestConfig(t)

	configData := File{
		Emulators: append([]Emulator{}, defaultEmulators...),
	}
	data, err := json.MarshalIndent(configData, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(backend.ConfigPath, data, 0644))

	cfg := &File{}
	result, err := cfg.LoadConfig()
	require.NoError(t, err)

	assert.Len(t, result.Emulators, len(defaultEmulators))
}

func TestGetEmulatorSources_MapsKnownIDsAndIgnoresUnknownIDs(t *testing.T) {
	cfg := &File{
		Emulators: []Emulator{
			{ID: "codex", ShouldNotify: false},
			{ID: "unknown", ShouldNotify: true},
		},
	}

	sources, err := cfg.GetEmulatorSources()
	require.NoError(t, err)
	require.Len(t, sources, 1)

	assert.Equal(t, "codex", sources[0].ID)
	assert.Equal(t, backend.CodexEmuDir, sources[0].Path)
	assert.Equal(t, "achievements.ini", sources[0].AchievementFile)
	assert.False(t, sources[0].ShouldNotify)
}

func TestGetEmulatorSources_EmptyWhenNoKnownIDs(t *testing.T) {
	cfg := &File{
		Emulators: []Emulator{
			{ID: "unknown", ShouldNotify: true},
		},
	}

	sources, err := cfg.GetEmulatorSources()
	require.NoError(t, err)
	assert.Empty(t, sources)
}

func TestCheckShouldNotify_MatchingPath(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{ID: "gse", ShouldNotify: true},
			{ID: "codex", ShouldNotify: false},
		},
	}

	assert.True(t, cfg.CheckShouldNotify(filepath.Join("/prefix", "drive_c", backend.EmuDir, "123", "achievements.json")))
	assert.False(t, cfg.CheckShouldNotify(filepath.Join("/prefix", "drive_c", backend.CodexEmuDir, "123", "achievements.ini")))
}

func TestCheckShouldNotify_NoMatch(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{ID: "gse", ShouldNotify: false},
		},
	}

	// Default is true when no match
	assert.True(t, cfg.CheckShouldNotify("/completely/different/path.json"))
}

func emulatorIDs(emulators []Emulator) []string {
	ids := make([]string, 0, len(emulators))
	for _, emulator := range emulators {
		ids = append(ids, emulator.ID)
	}
	return ids
}

func emulatorByID(emulators []Emulator, id string) Emulator {
	for _, emulator := range emulators {
		if emulator.ID == id {
			return emulator
		}
	}
	return Emulator{}
}

func TestSetLanguage_Valid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetLanguage("english")
	require.NoError(t, err)
	assert.Equal(t, "english", cfg.Language.API)
	assert.Equal(t, "English", cfg.Language.DisplayName)
}

func TestSetLanguage_Invalid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetLanguage("invalid-language")
	assert.Error(t, err)
}

func TestGetLanguage(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Language: types.Language{
			DisplayName: "German",
			API:         "german",
			WebAPI:      "de",
		},
	}

	lang := cfg.GetLanguage()
	assert.Equal(t, "german", lang.API)
	assert.Equal(t, "German", lang.DisplayName)
}

func TestGetSteamLanguages(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	languages := cfg.GetSteamLanguages()
	assert.NotEmpty(t, languages)
	// Should contain English
	hasEnglish := false
	for _, lang := range languages {
		if lang.API == "english" {
			hasEnglish = true
			break
		}
	}
	assert.True(t, hasEnglish)
}

func TestSetNotificationSound_Valid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetNotificationSound("steam-deck.wav")
	require.NoError(t, err)
	assert.Equal(t, "steam-deck.wav", cfg.NotificationSound)
}

func TestSetNotificationSound_Disabled(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetNotificationSound("")
	require.NoError(t, err)
	assert.Equal(t, "", cfg.NotificationSound)
}

func TestSetNotificationSound_Invalid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetNotificationSound("nonexistent-sound.wav")
	assert.Error(t, err)
}

func TestGetAvailableSounds(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	sounds := cfg.GetAvailableSounds()
	assert.NotEmpty(t, sounds)
	// Should have "Disabled" option
	hasDisabled := false
	for _, sound := range sounds {
		if sound.Value == "" && sound.Name == "Disabled" {
			hasDisabled = true
			break
		}
	}
	assert.True(t, hasDisabled)
}

func TestSetLogLevel_Valid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetLogLevel("debug")
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestSetLogLevel_Invalid(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	err := cfg.SetLogLevel("invalid-level")
	assert.Error(t, err)
}

func TestGetAvailableLogLevels(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{}

	levels := cfg.GetAvailableLogLevels()
	assert.Len(t, levels, 3) // info, debug, off
}
