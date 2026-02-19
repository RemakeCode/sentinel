package watcher

import (
	"log"
	"os"
	"regexp"
)

var numericRegex = regexp.MustCompile(`^\d+$`)

func Scan(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	var appIDs []string
	for _, entry := range entries {
		if entry.IsDir() && numericRegex.MatchString(entry.Name()) {
			appID := entry.Name()
			appIDs = append(appIDs, appID)
		}
	}

	log.Println(appIDs)

	// Create local appId GameCache
	// if err := cache.save(appID,{}); err != nil {
	// 	// Log error but continue scanning? Or return?
	// 	// For now simpler to ignore file creation errors for scanning to proceed
	// 	// or we could log it. Since I don't have a logger passed in, I'll ignore for now
	// 	// but this is where the logic happens.
	// }

}
