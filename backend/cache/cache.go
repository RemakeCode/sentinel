package cache

import (
	"os"
	"path/filepath"
)

type GameCache struct {
	AppID       string `json:"appId"`
	Name        string `json:"name"`
	HeaderImage string `json:"headerImage"`
}

func (g *GameCache) save(appID string, gameDetails *GameCache) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	basePath := filepath.Join(cacheDir, "sentinel", "games")
	appPath := filepath.Join(basePath, appID)

	if err := os.MkdirAll(appPath, 0755); err != nil {
		return err
	}

	jsonPath := filepath.Join(appPath, appID+".json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return os.WriteFile(jsonPath, []byte("{}"), 0644)
	}

	return nil
}

func (g *GameCache) get(appID string) (*GameCache, error) {
	return g, nil
}
