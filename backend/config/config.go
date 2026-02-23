package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend/steam/types"
	"strings"
)

type Emulator struct {
	Path         string `json:"path"`
	ShouldNotify bool   `json:"shouldNotify"`
	IsDefault    bool   `json:"isDefault"`
}

type SteamSource string

const (
	Unknown  SteamSource = ""
	Key      SteamSource = "key"
	External SteamSource = "external"
)

//wails:internal
type File struct {
	Language          types.Language `json:"language"`
	Emulators         []Emulator     `json:"emulators"`
	SteamAPIKey       string         `json:"-"`
	SteamDataSource   SteamSource    `json:"steamDataSource"`
	SteamAPIKeyMasked string         `json:"steamApiKeyMasked"`
}

var p1, _ = os.UserHomeDir()
var p2, _ = os.UserConfigDir()
var p3, _ = os.UserCacheDir()

var configDir = filepath.Join(p3, "sentinel")
var configPath = filepath.Join(configDir, "config.json")

// Embedded encryption key for Steam API key
// This is intentionally weak security.
var encryptionKey = []byte("sentinel-app-secret-key-32bytes!")

// Cache directory paths
var cacheDir = filepath.Join(configDir, "cache")
var cacheDataDir = filepath.Join(cacheDir, "data")
var cacheIconDir = filepath.Join(cacheDir, "icon")
var cacheSchemaDir = filepath.Join(cacheDir, "schema")

var defaultEmulatorPaths = []Emulator{
	{Path: fmt.Sprintf("%s/Public/Documents/Steam/CODEX", p1), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Public/Documents/Steam/RUNE", p1), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Public/Documents/Steam/OnlineFix", p1), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Public/Documents/EMPRESS", p1), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Steam/CODEX", p2), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Goldberg SteamEmu Saves", p2), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/GSE Saves", p2), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/EMPRESS", p2), ShouldNotify: true, IsDefault: true},
}

func init() {
	slog.Info("Starting Config Initialization")
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// Ensure cache directories exist
	if err := os.MkdirAll(cacheDataDir, 0755); err != nil {
		log.Fatalf("Failed to create cache data directory: %v", err)
	}
	if err := os.MkdirAll(cacheIconDir, 0755); err != nil {
		log.Fatalf("Failed to create cache icon directory: %v", err)
	}
	if err := os.MkdirAll(cacheSchemaDir, 0755); err != nil {
		log.Fatalf("Failed to create cache schema directory: %v", err)
	}

	// Create language folders in schema directory based on steam languages
	languages := types.GetSteamLanguages()
	for _, lang := range languages {
		langDir := filepath.Join(cacheSchemaDir, lang.API)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			if err := os.MkdirAll(langDir, 0755); err != nil {
				log.Printf("Warning: Failed to create language directory %s: %v", lang.API, err)
			}
		}
	}

	_, err := os.Stat(configPath)

	if os.IsNotExist(err) {
		// File doesn't exist - initialize default config
		defaultConfig := File{
			Emulators:       defaultEmulatorPaths,
			SteamDataSource: "external",
			Language: types.Language{
				DisplayName: "English", API: "english", WebAPI: "en",
			},
		}
		config, marshalErr := json.MarshalIndent(defaultConfig, "", "  ")
		if marshalErr != nil {
			log.Fatalf("Failed to marshal default config: %v", marshalErr)
		}

		err := os.WriteFile(configPath, config, 0644)
		if err != nil {
			log.Fatalf("Failed to write default config: %v", err)
		}
	} else if err != nil {
		// Handle other errors (e.g., permission issues)
		log.Fatalf("Unexpected error checking config: %v", err)
	}

	slog.Info("Config Initialization Complete")
}

func (c *File) LoadConfig() (*File, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	//delete(c, c.SteamAPIKey)

	return c, nil
}

func (c *File) SaveConfig() error {
	data, err := json.MarshalIndent(c, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// SetSteamAPIKey sets the Steam API key in the configuration
func (c *File) SetSteamAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API Key is empty")
	}

	// Encrypt the API key before storing
	encryptedKey, err := encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	c.SteamAPIKey = encryptedKey

	c.SteamAPIKeyMasked = strings.Repeat("*", len(apiKey)-4) + apiKey[len(apiKey)-4:]

	return c.SaveConfig()
}

// GetSteamAPIKey retrieves and decrypts the Steam API key from the configuration
//
//wails:internal
func (c *File) GetSteamAPIKey() (string, error) {
	if c.SteamAPIKey == "" {
		return "", nil // No API key set
	}

	// Decrypt the API key before returning
	decryptedKey, err := decrypt(c.SteamAPIKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return decryptedKey, nil
}

// GetSteamDataSource retrieves the current Steam data source preference
func (c *File) GetSteamDataSource() SteamSource {
	if c.SteamDataSource == "" {
		return "external" // Default value
	}
	return c.SteamDataSource
}

// SetSteamDataSource sets the Steam data source preference and saves the configuration
func (c *File) SetSteamDataSource(source SteamSource) error {
	c.SteamDataSource = source
	return c.SaveConfig()
}

func (c *File) AddEmulator(path string) error {

	emulator := Emulator{
		Path:         path,
		IsDefault:    false,
		ShouldNotify: true,
	}

	// Check for duplicates
	for _, emu := range c.Emulators {
		if emu.Path == emulator.Path {
			return nil // Already exists
		}
	}

	c.Emulators = append(c.Emulators, emulator)
	return c.SaveConfig()
}

// RemoveEmulator removes an emulator from the configuration by index
func (c *File) RemoveEmulator(index int) error {
	if index < 0 || index >= len(c.Emulators) {
		return nil
	}

	if c.Emulators[index].IsDefault {
		return nil // Cannot remove default emulators
	}

	c.Emulators = append(c.Emulators[:index], c.Emulators[index+1:]...)
	return c.SaveConfig()

}

// encrypt encrypts plaintext using AES-256-GCM
func encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-256-GCM
func decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GetEmulatorPaths returns all emulator paths from the configuration
func (c *File) GetEmulatorPaths() []string {
	var paths []string
	for _, emulator := range c.Emulators {
		paths = append(paths, emulator.Path)
	}
	return paths
}

// ToggleEmulatorNotification toggles the notification setting for an emulator by index
func (c *File) ToggleEmulatorNotification(index int) error {

	if index < 0 || index >= len(c.Emulators) {
		return nil
	}
	c.Emulators[index].ShouldNotify = !c.Emulators[index].ShouldNotify

	return c.SaveConfig()

}
