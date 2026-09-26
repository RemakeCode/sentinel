//go:build integration

package steam

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sentinel/backend"
)

const (
	integrationAppID         = "620"     // Portal 2
	integrationOfficialAppID = "1245620" // Elden Ring
)

// These tests call the live services. Run them with `task test:steam-integration`.
// Image downloads are replaced with small local responses so API checks do not
// download assets or depend on image CDNs.
type integrationTransport struct{}

func (integrationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch strings.ToLower(filepath.Ext(req.URL.Path)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("integration-image"))),
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Request:    req,
		}, nil
	default:
		return http.DefaultTransport.RoundTrip(req)
	}
}

func newIntegrationService() *Service {
	service := &Service{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: integrationTransport{},
		},
	}
	service.clientOnce.Do(func() {})
	return service
}

func useIntegrationCacheDirs(t *testing.T) {
	t.Helper()

	originalGameCacheDir := backend.GameCacheDir
	originalIconCacheDir := backend.ACHCacheIconDir
	tempDir := t.TempDir()
	backend.GameCacheDir = filepath.Join(tempDir, "games")
	backend.ACHCacheIconDir = filepath.Join(tempDir, "icons")
	t.Cleanup(func() {
		backend.GameCacheDir = originalGameCacheDir
		backend.ACHCacheIconDir = originalIconCacheDir
	})
}

func TestIntegration_SearchApps(t *testing.T) {
	results, err := newIntegrationService().SearchApps("Portal 2")
	if err != nil {
		t.Fatalf("SearchApps failed against Steam: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Steam app search returned no results")
	}

	for _, result := range results {
		if result.AppID == integrationAppID && result.Name != "" {
			return
		}
	}
	t.Fatalf("Steam app search response did not contain Portal 2 (appID %s): %#v", integrationAppID, results)
}

func TestIntegration_AppDetails(t *testing.T) {
	useIntegrationCacheDirs(t)

	game, err := newIntegrationService().fetchGameDetailsFresh(integrationAppID, "english")
	if err != nil {
		t.Fatalf("fetchGameDetailsFresh failed against Steam appdetails: %v", err)
	}
	if strings.TrimSpace(game.Name) == "" {
		t.Fatal("Steam appdetails response decoded without a game name")
	}
	if game.Name != "Portal 2" {
		t.Fatalf("Steam appdetails returned %q for appID %s, expected Portal 2", game.Name, integrationAppID)
	}
}

func TestIntegration_OfficialAchievements(t *testing.T) {
	useIntegrationCacheDirs(t)

	achievements, err := newIntegrationService().fetchAchievementsFromOfficialAPI(integrationOfficialAppID, "english")
	if err != nil {
		t.Fatalf("official achievement API request failed: %v", err)
	}
	if len(achievements) == 0 {
		t.Fatal("official achievement API returned no achievements")
	}
	if achievements[0].Name == "" || achievements[0].DisplayName == "" {
		t.Fatalf("official achievement response is missing fields consumed by the app: %#v", achievements[0])
	}
}

func TestIntegration_ThirdPartyAchievements(t *testing.T) {
	useIntegrationCacheDirs(t)

	achievements, err := newIntegrationService().fetchAchievementsFromThirdParty(integrationAppID, "english")
	if err != nil {
		t.Fatalf("SteamHunters or Steam Community achievement request failed: %v", err)
	}
	if len(achievements) == 0 {
		t.Fatal("SteamHunters and Steam Community returned no usable achievements")
	}
	if achievements[0].Name == "" || achievements[0].DisplayName == "" {
		t.Fatalf("third-party achievement response is missing fields consumed by the app: %#v", achievements[0])
	}
	for _, achievement := range achievements {
		if achievement.Icon != "" {
			return
		}
	}
	t.Fatal("Steam Community achievement page yielded no achievement icons")
}

func TestIntegration_GlobalAchievementPercentages(t *testing.T) {
	percentages, err := newIntegrationService().GetGlobalAchievementPercentages(integrationAppID)
	if err != nil {
		t.Fatalf("global achievement percentages request failed: %v", err)
	}
	if len(percentages) == 0 {
		t.Fatal("global achievement percentages API returned no achievements")
	}
	if percentages[0].Name == "" || percentages[0].Percent == "" {
		t.Fatalf("global percentages response is missing consumed fields: %#v", percentages[0])
	}
}

func TestIntegration_PortraitFallback(t *testing.T) {
	imageURL := newIntegrationService().fallbackPortraitURL(integrationAppID)
	if imageURL == "" {
		t.Fatal("portrait fallback API failed or returned no library_capsule asset")
	}
	if !strings.Contains(imageURL, fmt.Sprintf("/steam/apps/%s/", integrationAppID)) {
		t.Fatalf("portrait fallback returned an unexpected asset URL: %s", imageURL)
	}
}
