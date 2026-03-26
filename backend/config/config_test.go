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
			{Path: "/test/path", ShouldNotify: true},
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
	assert.Len(t, result.Emulators, 1)
	assert.Equal(t, "/test/path", result.Emulators[0].Path)
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
			{Path: "/save/test", ShouldNotify: true},
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
	assert.Len(t, loaded.Emulators, 1)
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

func TestAddEmulator(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{},
	}

	err := cfg.AddEmulator("/new/emulator/path")
	require.NoError(t, err)
	assert.Len(t, cfg.Emulators, 1)
	assert.Equal(t, "/new/emulator/path", cfg.Emulators[0].Path)
	assert.True(t, cfg.Emulators[0].ShouldNotify)
}

func TestAddEmulator_Duplicate(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{Path: "/existing/path"},
		},
	}

	// Adding same path should not create duplicate
	err := cfg.AddEmulator("/existing/path")
	require.NoError(t, err)
	assert.Len(t, cfg.Emulators, 1)
}

func TestRemoveEmulator(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{Path: "/path/1"},
			{Path: "/path/2"},
		},
	}

	err := cfg.RemoveEmulator(0)
	require.NoError(t, err)
	assert.Len(t, cfg.Emulators, 1)
	assert.Equal(t, "/path/2", cfg.Emulators[0].Path)
}

func TestRemoveEmulator_Default(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{Path: "/default/path"},
		},
	}

	// We can now remove any emulator since IsDefault was removed
	err := cfg.RemoveEmulator(0)
	require.NoError(t, err)
	assert.Len(t, cfg.Emulators, 0)
}

func TestRemoveEmulator_InvalidIndex(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{Path: "/path/1"},
		},
	}

	// Invalid indices should not error
	err := cfg.RemoveEmulator(-1)
	require.NoError(t, err)
	err = cfg.RemoveEmulator(100)
	require.NoError(t, err)
	assert.Len(t, cfg.Emulators, 1)
}

func TestGetEmulatorPaths(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{Path: "/path/1"},
			{Path: "/path/2"},
			{Path: "/path/3"},
		},
	}

	paths, err := cfg.GetEmulatorPaths()
	require.NoError(t, err)
	assert.Len(t, paths, 3)
	assert.Contains(t, paths, "/path/1")
	assert.Contains(t, paths, "/path/2")
	assert.Contains(t, paths, "/path/3")
}

func TestToggleEmulatorNotification(t *testing.T) {
	_, tempDir := setupTestConfig(t)
	_ = tempDir

	cfg := &File{
		Emulators: []Emulator{
			{Path: "/path/1", ShouldNotify: true},
			{Path: "/path/2", ShouldNotify: false},
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
			{Path: "/path/1", ShouldNotify: true},
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

func TestDefaultEmulatorPaths(t *testing.T) {
	// Verify default paths use backend constants
	assert.NotEmpty(t, defaultEmulatorPaths)

	for _, emu := range defaultEmulatorPaths {
		assert.True(t, emu.ShouldNotify)
		assert.NotEmpty(t, emu.Path)
	}
}
