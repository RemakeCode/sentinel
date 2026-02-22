package watcher

import (
	"os"
	"regexp"
	"time"

	"sentinel/backend/config"
	"sentinel/backend/steam"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	app         *application.App
	watcher     *fsnotify.Watcher
	done        chan struct{}
	failedPaths []string // Tracks paths that failed to watch
	retryTimer  *time.Timer
	config      *config.File
}

var numericRegex = regexp.MustCompile(`^\d+$`)

// Scan scans a path for steam emulator appid folders
func (w *Service) Scan(path string) []string {
	entries, err := os.ReadDir(path)

	if err != nil {
		return []string{}
	}

	var appIDs []string
	for _, entry := range entries {
		if entry.IsDir() && numericRegex.MatchString(entry.Name()) {
			appID := entry.Name()
			appIDs = append(appIDs, appID)
		}
	}

	return appIDs
}

// Start initializes the file system watcher and begins monitoring paths
func (w *Service) Start() error {
	paths := w.config.GetEmulatorPaths()

	if len(paths) == 0 {
		w.app.Logger.Info("No emulator paths configured, watcher not started")
		return nil
	}
	w.app.Logger.Info("Starting watcher", "paths", paths)

	// Create new fsnotify watcher
	fswatcher, err := fsnotify.NewWatcher()

	if err != nil {
		w.app.Logger.Error("Failed to create watcher", "error", err)
		return err
	}
	w.watcher = fswatcher
	w.done = make(chan struct{})
	w.failedPaths = nil

	// Add all paths to the watcher
	for _, path := range paths {
		if err := w.watchPath(path); err != nil {
			w.app.Logger.Warn("Failed to watch path", "path", path, "error", err)
			w.failedPaths = append(w.failedPaths, path)
		}
	}

	// Start retry timer for failed paths
	if len(w.failedPaths) > 0 {
		w.startRetryTimer()
	}

	// Start event processing loop
	go w.processEvents()

	// Perform initial scan of paths to find steam appid folders and fetch metadata
	var allAppIDs []string
	for _, path := range paths {
		appIDs := w.Scan(path)
		allAppIDs = append(allAppIDs, appIDs...)
	}

	// Fetch metadata for all discovered appIds
	if len(allAppIDs) > 0 {
		w.triggerMetadataFetch(allAppIDs)
	}

	w.app.Logger.Info("Watcher started successfully")
	return nil
}

// Stop gracefully shuts down the watcher
func (w *Service) Stop() {
	w.app.Logger.Info("Stopping watcher")

	// Stop retry timer
	if w.retryTimer != nil {
		w.retryTimer.Stop()
		w.retryTimer = nil
	}

	if w.done != nil {
		close(w.done)
	}

	if w.watcher != nil {
		if err := w.watcher.Close(); err != nil {
			w.app.Logger.Error("Error closing watcher", "error", err)
		}
	}

	w.app.Logger.Info("Watcher stopped")
}

// watchPath adds a path to the file system watcher
func (w *Service) watchPath(path string) error {
	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.ErrNotExist
	}

	// Add path to watcher
	if err := w.watcher.Add(path); err != nil {
		return err
	}

	w.app.Logger.Debug("Watching path", "path", path)
	return nil
}

// startRetryTimer starts a timer to retry watching failed paths every 5 minutes
func (w *Service) startRetryTimer() {
	w.retryTimer = time.AfterFunc(5*time.Minute, func() {
		w.retryFailedPaths()
	})
}

// retryFailedPaths attempts to re-watch paths that previously failed
func (w *Service) retryFailedPaths() {
	select {
	case <-w.done:
		return
	default:
	}

	w.app.Logger.Info("Retrying failed paths", "count", len(w.failedPaths))

	var stillFailed []string
	for _, path := range w.failedPaths {
		if err := w.watchPath(path); err != nil {
			w.app.Logger.Warn("Still failed to watch path", "path", path, "error", err)
			stillFailed = append(stillFailed, path)
		} else {
			w.app.Logger.Info("Successfully re-watched path", "path", path)
			// Perform initial scan for newly watched path
			w.Scan(path)
		}
	}

	w.failedPaths = stillFailed

	// Restart timer if there are still failed paths
	if len(w.failedPaths) > 0 {
		w.startRetryTimer()
	}
}

// processEvents handles file system events from the watcher
func (w *Service) processEvents() {
	for {
		select {
		case <-w.done:
			// Stop processing events
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.app.Logger.Debug("File system event", "path", event.Name, "event", event.Op.String())
			// TBD: Handle file system events
			// This function will be called when a file system event occurs
			// Possible actions: update game state, refresh UI, trigger notifications, etc.
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.app.Logger.Error("Watcher error", "error", err)
		}
	}
}

// handleEvent processes a file system event
// TBD: Implement event handling logic
func (w *Service) handleEvent(event fsnotify.Event) {
	// Placeholder for event handling
	// This function will be called when a file system event occurs
	// Possible actions:
	// - Update game state in database
	// - Refresh UI components
	// - Trigger notifications
	// - Sync with external services
	// - Update cache
}

// triggerMetadataFetch fetches Steam metadata for the given appIds in a background goroutine
func (w *Service) triggerMetadataFetch(appIDs []string) {
	if len(appIDs) == 0 {
		return
	}

	// Fetch in background goroutine
	go func() {
		w.app.Logger.Info("Fetching metadata", "appIDs", appIDs)
		games, err := steam.FetchAppDetailsBulk(appIDs)
		if err != nil {
			w.app.Logger.Error("Failed to fetch metadata", "error", err)
			return
		}

		w.app.Logger.Info("Metadata fetched successfully", "count", len(games))
	}()
}
