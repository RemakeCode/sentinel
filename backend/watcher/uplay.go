package watcher

import (
	_ "embed"
	"encoding/json"
)

// The mapping is derived from PSerban93/Achievements, licensed under MIT:
// https://github.com/PSerban93/Achievements/blob/main/assets/uplay-steam.json
// Only non-null uplay_id to steam_appid pairs are bundled here.
//
//go:embed uplay-steam.json
var uplaySteamMappingData []byte

var uplaySteamMapping = loadUplaySteamMapping()

func loadUplaySteamMapping() map[string]string {
	var mapping map[string]string
	if err := json.Unmarshal(uplaySteamMappingData, &mapping); err != nil {
		panic(err)
	}
	return mapping
}
