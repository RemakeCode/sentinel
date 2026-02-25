package config

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/steam/types"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
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
	app               *application.App
	Language          types.Language `json:"language"`
	Emulators         []Emulator     `json:"emulators"`
	SteamAPIKey       string         `json:"-"`
	SteamDataSource   SteamSource    `json:"steamDataSource"`
	SteamAPIKeyMasked string         `json:"steamApiKeyMasked"`
}

var defaultEmulatorPaths = []Emulator{
	{Path: fmt.Sprintf("%s/Public/Documents/Steam/CODEX", backend.UserHomeDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Public/Documents/Steam/RUNE", backend.UserHomeDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Public/Documents/Steam/OnlineFix", backend.UserHomeDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Public/Documents/EMPRESS", backend.UserHomeDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Steam/CODEX", backend.UserHomeDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/Goldberg SteamEmu Saves", backend.UserCacheDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/GSE Saves", backend.UserCacheDir), ShouldNotify: true, IsDefault: true},
	{Path: fmt.Sprintf("%s/EMPRESS", backend.UserCacheDir), ShouldNotify: true, IsDefault: true},
}

// Not a secure Key. Left this way intentionally.
var encryptionKey = []byte("sentinel-app-secret-key-32bytes!")

var (
	instance     *File
	instanceOnce sync.Once
	instanceErr  error
)

// Get returns the package-level singleton *File, loading it on first call.
// Safe to call from multiple goroutines.
func Get() (*File, error) {
	instanceOnce.Do(func() {
		f := &File{}
		if _, err := f.LoadConfig(); err != nil {
			instanceErr = fmt.Errorf("config: failed to load: %w", err)
			return
		}
		instance = f
	})
	return instance, instanceErr
}

func (c *File) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	slog.Info("Starting Config Initialization")

	// Ensure config directory exists
	if err := os.MkdirAll(backend.ConfigDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// Ensure cache directories exist
	if err := os.MkdirAll(backend.ACHCacheDataDir, 0755); err != nil {
		log.Fatalf("Failed to create cache data directory: %v", err)
	}
	if err := os.MkdirAll(backend.ACHCacheIconDir, 0755); err != nil {
		log.Fatalf("Failed to create cache icon directory: %v", err)
	}
	if err := os.MkdirAll(backend.GameCacheDir, 0755); err != nil {
		log.Fatalf("Failed to create game cache directory: %v", err)
	}

	// Create language folders in game cache directory based on steam languages
	languages := types.GetSteamLanguages()
	for _, lang := range languages {
		langDir := filepath.Join(backend.GameCacheDir, lang.API)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			if err := os.MkdirAll(langDir, 0755); err != nil {
				log.Printf("Warning: Failed to create language directory %s: %v", lang.API, err)
			}
		}
	}

	_, err := os.Stat(backend.ConfigPath)

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

		err := os.WriteFile(backend.ConfigPath, config, 0644)
		if err != nil {
			log.Fatalf("Failed to write default config: %v", err)
		}
	} else if err != nil {
		// Handle other errors (e.g., permission issues)
		log.Fatalf("Unexpected error checking config: %v", err)
	}

	slog.Info("Config Initialization Complete")

	_, err = c.LoadConfig()

	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	return nil
}

func (c *File) LoadConfig() (*File, error) {
	data, err := os.ReadFile(backend.ConfigPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return nil, errors.New("unable to unmarshal config")
	}

	return c, nil
}

func (c *File) SaveConfig() error {
	data, err := json.MarshalIndent(c, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(backend.ConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	if err := os.WriteFile(backend.ConfigPath, data, 0644); err != nil {
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
func (c *File) GetEmulatorPaths() ([]string, error) {
	var paths []string
	for _, emulator := range c.Emulators {
		paths = append(paths, emulator.Path)
	}
	return paths, nil
}

// ToggleEmulatorNotification toggles the notification setting for an emulator by index
func (c *File) ToggleEmulatorNotification(index int) error {

	if index < 0 || index >= len(c.Emulators) {
		return nil
	}
	c.Emulators[index].ShouldNotify = !c.Emulators[index].ShouldNotify

	return c.SaveConfig()

}
