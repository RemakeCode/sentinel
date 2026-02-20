package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sentinel/backend/steam"
)

type Emulator struct {
	Path         string `json:"path"`
	ShouldNotify bool   `json:"shouldNotify"`
	IsDefault    bool   `json:"isDefault"`
}

//wails:bind
type CfgFile struct {
	Language  steam.Language
	Emulators []Emulator `json:"emulators"`
}

var p1, _ = os.UserHomeDir()
var p2, _ = os.UserConfigDir()
var p3, _ = os.UserCacheDir()

var configDir = filepath.Join(p3, "sentinel")
var configPath = filepath.Join(configDir, "config.json")

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
	log.Print("Starting Config Initialization")

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
	languages := steam.GetSteamLanguages()
	for _, lang := range languages {
		langDir := filepath.Join(cacheSchemaDir, lang.API)
		if err := os.MkdirAll(langDir, 0755); err != nil {
			log.Printf("Warning: Failed to create language directory %s: %v", lang.API, err)
		}
	}

	_, err := os.Stat(configPath)

	if os.IsNotExist(err) {
		// File doesn't exist - initialize default config
		defaultConfig := CfgFile{Emulators: defaultEmulatorPaths}
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

	log.Print("Config Initialization Complete")
}

func (c *CfgFile) LoadConfig() (*CfgFile, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *CfgFile) SaveConfig() error {
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

func (c *CfgFile) AddEmulator(path string) error {

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
func (c *CfgFile) RemoveEmulator(index int) error {
	if index < 0 || index >= len(c.Emulators) {
		return nil
	}

	if c.Emulators[index].IsDefault {
		return nil // Cannot remove default emulators
	}

	c.Emulators = append(c.Emulators[:index], c.Emulators[index+1:]...)
	return c.SaveConfig()
}

// ToggleEmulatorNotification toggles the notification setting for an emulator by index
func (c *CfgFile) ToggleEmulatorNotification(index int) error {

	if index < 0 || index >= len(c.Emulators) {
		// Wails 3: Runtime logging handled differently
		return nil
	}
	c.Emulators[index].ShouldNotify = !c.Emulators[index].ShouldNotify

	return c.SaveConfig()

}
