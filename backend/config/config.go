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
	"os/exec"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/steam/types"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Prefix struct {
	Path string `json:"path"`
}
type Emulator struct {
	Path         string `json:"path"`
	ShouldNotify bool   `json:"shouldNotify"`
}

type SteamSource string

const (
	Unknown  SteamSource = ""
	Key      SteamSource = "key"
	External SteamSource = "external"
)

type SoundOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Company     string `json:"company"`
	Year        string `json:"year"`
	Description string `json:"description"`
	GitHub      string `json:"github"`
}

//wails:internal
type File struct {
	app               *application.App
	Language          types.Language `json:"language"`
	Emulators         []Emulator     `json:"emulators"`
	Prefixes          []Prefix       `json:"prefixes"`
	SteamAPIKey       string         `json:"SteamAPIKey"`
	SteamDataSource   SteamSource    `json:"steamDataSource"`
	SteamAPIKeyMasked string         `json:"steamApiKeyMasked"`
	NotificationSound string         `json:"notificationSound"`
}

var defaultEmulatorPaths = []Emulator{
	{Path: backend.EmuDir, ShouldNotify: true},
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

	// Ensure cache directory exists (subdirectories are created automatically)
	if err := os.MkdirAll(backend.ACHCacheDir, 0755); err != nil {
		log.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create language folders in game cache directory based on steam languages
	languages := types.GetSteamLanguages()
	for _, lang := range languages {
		langDir := filepath.Join(backend.GameCacheDir, lang.API)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			if err := os.MkdirAll(langDir, 0755); err != nil {
				slog.Info("Warning: Failed to create language directory %s: %v", lang.API, err)
			}
		}
	}

	_, err := os.Stat(backend.ConfigPath)

	if os.IsNotExist(err) {
		// File doesn't exist - initialize default config
		defaultConfig := File{
			Emulators:         defaultEmulatorPaths,
			SteamDataSource:   "external",
			NotificationSound: "steam-deck.wav",
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

	cfg, err := Get()
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	_, err = cfg.LoadConfig()

	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	return nil
}

func (c *File) getConfig() *File {
	cfg, _ := Get()
	return cfg
}

//wails:internal
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

func (c *File) GetConfig() (*File, error) {
	return Get()
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
	cfg := c.getConfig()
	if apiKey == "" {
		return fmt.Errorf("API Key is empty")
	}

	// Encrypt the API key before storing
	encryptedKey, err := encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	cfg.SteamAPIKey = encryptedKey

	cfg.SteamAPIKeyMasked = strings.Repeat("*", len(apiKey)-4) + apiKey[len(apiKey)-4:]

	return cfg.SaveConfig()
}

// GetSteamAPIKey retrieves and decrypts the Steam API key from the configuration
//
//wails:internal
func (c *File) GetSteamAPIKey() (string, error) {
	cfg := c.getConfig()
	if cfg.SteamAPIKey == "" {
		return "", errors.New("no API Key set") // No API key set
	}

	// Decrypt the API key before returning
	decryptedKey, err := decrypt(cfg.SteamAPIKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return decryptedKey, nil
}

// GetSteamDataSource retrieves the current Steam data source preference
func (c *File) GetSteamDataSource() SteamSource {
	cfg := c.getConfig()
	if cfg.SteamDataSource == "" {
		return "external" // Default value
	}
	return cfg.SteamDataSource
}

// SetSteamDataSource sets the Steam data source preference and saves the configuration
func (c *File) SetSteamDataSource(source SteamSource) error {
	cfg := c.getConfig()
	cfg.SteamDataSource = source
	return cfg.SaveConfig()
}

func (c *File) AddPrefix(path string) error {
	cfg := c.getConfig()
	prefix := Prefix{Path: path}

	for _, p := range cfg.Prefixes {
		if p.Path == prefix.Path {
			return nil
		}
	}

	cfg.Prefixes = append(cfg.Prefixes, prefix)
	return cfg.SaveConfig()
}

func (c *File) RemovePrefix(index int) error {
	cfg := c.getConfig()
	if index < 0 || index >= len(cfg.Prefixes) {
		return nil
	}

	cfg.Prefixes = append(cfg.Prefixes[:index], cfg.Prefixes[index+1:]...)
	return cfg.SaveConfig()
}

func (c *File) GetPrefixPaths() ([]string, error) {
	cfg := c.getConfig()
	var paths []string
	for _, prefix := range cfg.Prefixes {
		paths = append(paths, prefix.Path)
	}
	return paths, nil
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
	cfg := c.getConfig()
	var paths []string
	for _, emulator := range cfg.Emulators {
		paths = append(paths, emulator.Path)
	}
	return paths, nil
}

// ToggleEmulatorNotification toggles the notification setting for an emulator by index
func (c *File) ToggleEmulatorNotification(index int) error {
	cfg := c.getConfig()

	if index < 0 || index >= len(cfg.Emulators) {
		return nil
	}
	cfg.Emulators[index].ShouldNotify = !cfg.Emulators[index].ShouldNotify

	return cfg.SaveConfig()

}

// SetLanguage sets the language preference
func (c *File) SetLanguage(api string) error {
	cfg := c.getConfig()

	languages := types.GetSteamLanguages()
	for _, lang := range languages {
		if lang.API == api {
			cfg.Language = lang
			return cfg.SaveConfig()
		}
	}

	return fmt.Errorf("language not found: %s", api)
}

// GetSteamLanguages returns the list of available Steam languages
func (c *File) GetSteamLanguages() []types.Language {
	return types.GetSteamLanguages()
}

// GetAvailableSounds returns the list of available notification sound files
func (c *File) GetAvailableSounds() []SoundOption {
	return []SoundOption{
		{Name: "Disabled", Value: ""},
		{Name: "GOG Galaxy", Value: "gog-galaxy.wav"},
		{Name: "PlayStation", Value: "playstation.wav"},
		{Name: "PlayStation 5 Platinum", Value: "playstation5-platinum.wav"},
		{Name: "PlayStation 5", Value: "playstation5.wav"},
		{Name: "Steam Deck", Value: "steam-deck.wav"},
		{Name: "Windows 10", Value: "windows-10.wav"},
		{Name: "Windows 11", Value: "windows-11.wav"},
		{Name: "Windows 8", Value: "windows-8.wav"},
		{Name: "Xbox Rare", Value: "xbox-rare.wav"},
		{Name: "Xbox", Value: "xbox.wav"},
	}
}

// SetNotificationSound sets the notification sound preference
func (c *File) SetNotificationSound(sound string) error {
	cfg := c.getConfig()
	availableSounds := cfg.GetAvailableSounds()

	// Validate sound exists or is empty string (no sound)
	valid := false
	for _, s := range availableSounds {
		if s.Value == sound {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid sound: %s", sound)
	}

	cfg.NotificationSound = sound
	return cfg.SaveConfig()
}

// CheckShouldNotify checks if the path matches any emulator path and returns the ShouldNotify setting
func (c *File) CheckShouldNotify(path string) bool {
	cfg := c.getConfig()
	for _, emulator := range cfg.Emulators {
		if emulator.Path != "" && strings.Contains(path, emulator.Path) {
			return emulator.ShouldNotify
		}
	}
	return true
}

// PlaySound plays a sound file asynchronously using paplay or aplay
func (c *File) PlaySound(filename string) error {
	if filename == "" {
		return nil
	}

	soundPath := filepath.Join(backend.MediaDir, filename)
	if _, err := os.Stat(soundPath); err != nil {
		return nil
	}

	go func() {
		var cmd *exec.Cmd
		if _, err := exec.LookPath("paplay"); err == nil {
			cmd = exec.Command("paplay", soundPath)
		} else if _, err := exec.LookPath("aplay"); err == nil {
			cmd = exec.Command("aplay", soundPath)
		} else {
			slog.Warn("No audio playback utility available (paplay/aplay)")
			return
		}

		if err := cmd.Run(); err != nil {
			slog.Warn("Failed to play sound", "filename", filename, "error", err)
		}
	}()

	return nil
}

func (c *File) GetAppInfo() AppInfo {
	return AppInfo{
		Name:        cases.Title(language.Und).String(backend.AppName),
		Version:     backend.Version,
		Company:     "Remake Code",
		Year:        "2026",
		Description: "An Achievement Watcher for Linux",
		GitHub:      "https://github.com/RemakeCode/sentinel",
	}
}
