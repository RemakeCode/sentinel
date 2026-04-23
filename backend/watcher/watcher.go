package watcher

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sentinel/backend"
	"sentinel/backend/steam/types"
	"slices"
	"strings"
	"time"

	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/steam"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type SteamService interface {
	FetchAppDetailsBulk(appIDs []string, language types.Language) ([]*steam.GameBasics, error)
}

type AchService interface {
	SaveAch(path string) error
	ParseAch(path string) (*ach.AchievementData, error)
	LoadCachedAch(appId string) (*ach.AchievementData, error)
}

type Notifier interface {
	SendNotification(appId string, achievements map[string]ach.Achievement, isProgress bool, shouldNotify bool) error
}

type Service struct {
	watcher     *fsnotify.Watcher
	done        chan struct{}
	failedPaths []string // Tracks paths that failed to watch
	retryTimer  *time.Timer
	Steam       SteamService
	Ach         AchService
	Config      *config.File
	Notifier    Notifier // Injected dependency for notifications
	prefixPaths []string // Top-level prefix paths from config to watch for new games
	appIDPaths  []string // All appId paths being watched
}

var numericRegex = regexp.MustCompile(`^\d+$`)

// scanResult contains the results of scanning for steam emulator appid folders
type scanResult struct {
	AppIDs     []string // Array of app IDs (numeric strings)
	AppIDPaths []string // Array of full paths to app ID folders
}

func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	if s.Config == nil {
		c, err := config.Get()
		if err != nil {
			return err
		}
		s.Config = c
	}

	prefixPaths, err := s.Config.GetPrefixPaths()

	if err != nil {
		return err
	}

	if len(prefixPaths) == 0 {
		slog.Info("Watcher disabled: no prefix paths configured")
		return nil
	}

	err = s.Start()

	if err != nil {
		return err
	}
	return nil
}

func (s *Service) scan(paths []string) scanResult {
	var appIDs []string
	var appIDPaths []string

	for _, path := range paths {
		entries, err := os.ReadDir(path)

		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && numericRegex.MatchString(entry.Name()) {
				appID := entry.Name()
				appIDs = append(appIDs, appID)
				appIDPaths = append(appIDPaths, filepath.Join(path, appID))
			}
		}
	}

	return scanResult{
		AppIDs:     appIDs,
		AppIDPaths: appIDPaths,
	}
}

//TODO Start method should be refactored into watchPathForNotification
//TODO Create a new method that watches the Emupaths.

// Start initializes the file system watcher and begins monitoring paths
func (s *Service) Start() error {
	prefixPaths, err := s.Config.GetPrefixPaths()
	if err != nil {
		slog.Error("Failed to get prefix paths", "error", err)
		return err
	}

	s.prefixPaths = prefixPaths

	var scanPaths []string

	if len(prefixPaths) > 0 {
		for _, prefix := range prefixPaths {
			fullPaths, err := s.computeFullPath(prefix)

			if err != nil {
				slog.Warn("Failed to compute additional path", "prefix", prefix, "error", err)
				continue
			}
			for _, fullPath := range fullPaths {
				scanPaths = append(scanPaths, fullPath)
			}
		}
		slog.Info("Prefix paths configured, scanning prefix × emulator paths", "prefixes", len(prefixPaths))
	} else {
		slog.Info("No prefix paths configured, scanning emulator paths directly")
	}

	scanResult := s.scan(scanPaths)

	// Fetch metadata for all discovered appIds
	if len(scanResult.AppIDs) > 0 {
		s.triggerMetadataFetch(scanResult.AppIDs)
	}
	//Watch the exact folder with achievements
	slog.Info("Starting watcher", "paths", scanResult.AppIDPaths)

	// Create new fsnotify watcher
	fswatcher, err := fsnotify.NewWatcher()

	if err != nil {
		slog.Error("Failed to create watcher", "error", err)
		return err
	}
	s.watcher = fswatcher
	s.done = make(chan struct{})
	s.failedPaths = nil

	// Add all appId paths to the watcher
	for _, path := range scanResult.AppIDPaths {
		if err := s.Ach.SaveAch(path); err != nil {
			slog.Warn("Could not cache ach from path", "path", path, "error", err)
		}

		if err := s.watchPath(path); err != nil {
			slog.Warn("Failed to watch path", "path", path, "error", err)
			s.failedPaths = append(s.failedPaths, path)
		}
	}

	// Start retry timer for failed paths
	if len(s.failedPaths) > 0 {
		s.startRetryTimer()
	}

	// Start event processing loop
	go s.processEvents()

	// Start path walker for finding new games while app is running
	go s.PathWalker()

	slog.Info("Watcher started successfully")
	return nil
}

// Stop gracefully shuts down the watcher
func (s *Service) Stop() {
	slog.Info("Stopping watcher")

	// Stop retry timer
	if s.retryTimer != nil {
		s.retryTimer.Stop()
		s.retryTimer = nil
	}

	if s.done != nil {
		close(s.done)
	}

	if s.watcher != nil {
		if err := s.watcher.Close(); err != nil {
			slog.Error("Error closing watcher", "error", err)
		}
	}

	slog.Info("Watcher stopped")
}

// PathWalker periodically walks prefix directories to find new games
func (s *Service) PathWalker() {
	slog.Info("Path walker started", "interval", backend.WalkerInterval)
	ticker := time.NewTicker(backend.WalkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			for _, prefix := range s.prefixPaths {
				s.scanAndWatchPrefix(prefix)
			}
		}
	}
}

// scanAndWatchPrefix walks a prefix directory and watches any new game directories found
func (s *Service) scanAndWatchPrefix(prefix string) {
	watchlist := s.watcher.WatchList()

	err := filepath.WalkDir(prefix, func(path string, d os.DirEntry, err error) error {
		// Handle walk errors first
		if err != nil {
			// If there's a walk error, log it and skip this directory
			slog.Warn("Error walking path", "path", path, "error", err)
			return filepath.SkipDir
		}

		// Check if d is nil (shouldn't happen with proper error handling, but just in case)
		if d == nil {
			return nil
		}

		if d.IsDir() && strings.EqualFold(d.Name(), "drive_c") && slices.ContainsFunc(watchlist, func(s string) bool { return strings.Contains(s, path) }) {
			return filepath.SkipDir
		}

		if d.IsDir() && numericRegex.MatchString(d.Name()) {
			if err := s.watchPath(path); err != nil {
				return err
			}

			if err := s.Ach.SaveAch(path); err != nil {
				return err
			}

			s.triggerMetadataFetch([]string{filepath.Base(path)})
		}

		return nil
	})

	if err != nil {
		slog.Error("Error scanning prefix", "prefix", prefix, "error", err)
		return
	}
}

// watchPath adds a path to the file system watcher
func (s *Service) watchPath(path string) error {
	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.ErrNotExist
	}

	// Add path to watcher
	if err := s.watcher.Add(path); err != nil {
		return err
	}

	slog.Debug("Watching path", "path", path)
	return nil
}

// startRetryTimer starts a timer to retry watching failed paths every 5 minutes
func (s *Service) startRetryTimer() {
	s.retryTimer = time.AfterFunc(5*time.Minute, func() {
		s.retryFailedPaths()
	})
}

// retryFailedPaths attempts to re-watch paths that previously failed
func (s *Service) retryFailedPaths() {
	select {
	case <-s.done:
		return
	default:
	}

	slog.Info("Retrying failed paths", "count", len(s.failedPaths))

	var stillFailed []string
	for _, path := range s.failedPaths {
		if err := s.watchPath(path); err != nil {
			slog.Warn("Still failed to watch path", "path", path, "error", err)
			stillFailed = append(stillFailed, path)
		} else {
			slog.Info("Successfully re-watched path", "path", path)
		}
	}

	s.failedPaths = stillFailed

	// Restart timer if there are still failed paths
	if len(s.failedPaths) > 0 {
		s.startRetryTimer()
	}
}

// processEvents handles file system events from the watcher
func (s *Service) processEvents() {
	for {
		select {
		case <-s.done:
			// Stop processing events
			return

		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			slog.Debug("File system event", "path", event.Name, "event", event.Op.String())

			s.handleEvent(event)

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "error", err)
		}
	}
}

// handleEvent processes a file system event
// Currently logs events for future implementation
func (s *Service) handleEvent(event fsnotify.Event) {
	// Extract appId from the event path if it's a numeric directory
	path := event.Name

	appId := filepath.Base(filepath.Dir(path))

	// Log the event with relevant details
	slog.Info("File system event detected",
		"path", path,
		"appId", appId,
		"operation", event.Op.String(),
	)

	// Handle Create/WRITE events for achievements.json files
	if (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) && strings.HasSuffix(path, "achievements.json") {
		s.handleAchievementsWriteEvent(path, appId)
	}
}

// handleAchievementsWriteEvent processes write events on achievements.json files
func (s *Service) handleAchievementsWriteEvent(path, appId string) {
	app := application.Get()

	newAch, err := s.Ach.ParseAch(path)

	if err != nil {
		slog.Error("Failed to parse achievements", "error", err)
		return
	}

	oldAch, err := s.Ach.LoadCachedAch(appId)
	if err != nil {
		return
	}

	diff := newAch.Diff(oldAch)

	if len(diff.NewlyEarned) > 0 || len(diff.ProgressUpdated) > 0 {
		notifierService := s.Notifier
		shouldNotify := s.Config.CheckShouldNotify(path)
		if len(diff.NewlyEarned) > 0 {
			err := notifierService.SendNotification(appId, diff.NewlyEarned, false, shouldNotify)
			if err != nil {
				return
			}
		}
		if len(diff.ProgressUpdated) > 0 {
			err := notifierService.SendNotification(appId, diff.ProgressUpdated, true, shouldNotify)
			if err != nil {
				return
			}
		}

		if err := s.Ach.SaveAch(filepath.Dir(path)); err != nil {
			slog.Error("Failed to save achievements", "error", err)
		}
	}

	// Only emit event if application is running (not in tests)
	if app != nil {
		app.Event.Emit(backend.EventDataUpdated)
	}
}

// triggerMetadataFetch fetches Steam metadata for the given appIds in a background goroutine
func (s *Service) triggerMetadataFetch(appIDs []string) {
	if len(appIDs) == 0 {
		return
	}

	// Fetch in background goroutine
	go func() {
		slog.Info("Fetching metadata", "appIDs", appIDs)

		_, err := s.Steam.FetchAppDetailsBulk(appIDs, s.Config.Language)

		if err != nil {
			slog.Error("Failed to fetch metadata", "error", err)
			return
		}

		slog.Info("Metadata fetched successfully", "count", len(appIDs))
	}()
}

func (s *Service) computeFullPath(prefixPath string) ([]string, error) {
	emuPaths, err := s.Config.GetEmulatorPaths()
	if err != nil {
		return nil, err
	}

	var search func(dir string) []string
	search = func(dir string) []string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		var results []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			if strings.EqualFold(entry.Name(), "drive_c") {
				basePath := filepath.Join(dir, entry.Name(), "users", "steamuser")
				if len(emuPaths) > 0 {
					for _, emuPath := range emuPaths {
						results = append(results, filepath.Join(basePath, emuPath))
					}
				} else {
					results = append(results, basePath)
				}
			}

			subPath := filepath.Join(dir, entry.Name())
			results = append(results, search(subPath)...)
		}

		return results
	}

	result := search(prefixPath)
	if len(result) == 0 {
		return nil, errors.New("could not find drive_c in prefix path")
	}
	return result, nil
}
