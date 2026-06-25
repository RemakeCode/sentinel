package steam

import (
	"fmt"
	"net/http"
	"testing"
)

func TestOfficialAPI_WorksWithoutKey(t *testing.T) {
	url := fmt.Sprintf(
		"https://api.steampowered.com/IPlayerService/GetGameAchievements/v1/?appid=%s&language=%s",
		"1245620", "english",
	)
	resp, err := http.Get(url)
	if err != nil {
		t.Skipf("skipping network integration test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Official API returned %d without key — API may now require authentication", resp.StatusCode)
	}
}
