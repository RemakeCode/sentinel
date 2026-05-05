package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"sentinel/backend/notifier"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// AppError represents an application-level error with HTTP status code
type AppError struct {
	Status  int
	Message string
}

func (e AppError) Error() string {
	return e.Message
}

// HandlerFunc defines a handler that can return errors
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// Wrap converts our HandlerFunc into a standard http.HandlerFunc
func Wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			handleError(w, r, err)
		}
	}
}

// handleError logs and responds with a JSON error
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	// Log the error
	slog.Error(err.Error(), "method", r.Method, "path", r.URL.Path)

	// Default values
	status := http.StatusInternalServerError
	message := "An internal server error occurred"

	// Check if it's a specific AppError
	if appErr, ok := errors.AsType[AppError](err); ok {
		status = appErr.Status
		message = appErr.Message
	}

	// Send JSON error response (matches success format)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err = json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": message,
	})
	if err != nil {
		return
	}
}

// JSON responds with a JSON-encoded value and status code
func JSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type Router struct {
	Config   *config.File
	Steam    *steam.Service
	Watcher  *watcher.Service
	Notifier *notifier.Service
}

func NewRouter(c *config.File, s *steam.Service, w *watcher.Service, n *notifier.Service) *Router {
	return &Router{Config: c, Steam: s, Watcher: w, Notifier: n}
}

// Handler returns a fully configured chi router as an http.Handler
func (r *Router) Handler() http.Handler {
	router := chi.NewRouter()

	// 1. Setup CORS (Essential for Decky Loader plugins)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 2. Standard Middlewares
	router.Use(middleware.Logger)    // Logs requests to terminal
	router.Use(middleware.Recoverer) // Prevents app from crashing on API panics

	// 3. Define Routes
	router.Route("/api", func(api chi.Router) {
		// Config endpoints
		api.Get("/config", Wrap(r.handleGetConfig))
		api.Get("/config/available-sounds", Wrap(r.handleGetAvailableSounds))
		api.Put("/config/steam-api-key", Wrap(r.handleSetSteamAPIKey))
		api.Put("/config/steam-data-source", Wrap(r.handleSetSteamDataSource))
		api.Put("/config/logging", Wrap(r.handleSetLogging))
		api.Post("/config/notification-sound", Wrap(r.handleSetSound))
		api.Patch("/config/emulator-notification/{index}", Wrap(r.handleToggleEmulatorNotification))
		api.Post("/config/prefix", Wrap(r.handleAddPrefix))
		api.Delete("/config/prefix/{index}", Wrap(r.handleRemovePrefix))

		// Steam service endpoints
		api.Get("/steam/games", Wrap(r.handleGetAllGames))
		api.Get("/steam/games/{id}/global-achievement-percentages", Wrap(r.handleGetGlobalAchievementPercentages))

		// Notifier service endpoints
		api.Post("/notifier/play-sound", Wrap(r.handlePlaySound))
		api.Post("/notifier/test", Wrap(r.handleTestNotification))
		api.Post("/notifier/test-progress", Wrap(r.handleTestNotificationProgress))
		api.Get("/notifications", Wrap(r.handleNotifications))

		// Serve media files (game images, achievement icons, sounds)
		api.Get(fmt.Sprintf("/media/*"), http.HandlerFunc(r.handleServeMedia))
	})

	return router
}

func (r *Router) handleSetSound(w http.ResponseWriter, req *http.Request) error {
	var body struct {
		Sound string `json:"sound"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid request body"}
	}

	if err := r.Config.SetNotificationSound(body.Sound); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleGetConfig returns the current configuration
func (r *Router) handleGetConfig(w http.ResponseWriter, req *http.Request) error {
	return JSON(w, http.StatusOK, r.Config)
}

// handleGetAvailableSounds returns available notification sounds
func (r *Router) handleGetAvailableSounds(w http.ResponseWriter, req *http.Request) error {
	sounds := r.Config.GetAvailableSounds()
	return JSON(w, http.StatusOK, sounds)
}

// handleSetSteamAPIKey saves the Steam API key
func (r *Router) handleSetSteamAPIKey(w http.ResponseWriter, req *http.Request) error {
	var body struct {
		APIKey string `json:"apiKey"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid request body"}
	}

	if err := r.Config.SetSteamAPIKey(body.APIKey); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleSetSteamDataSource sets the Steam data source
func (r *Router) handleSetSteamDataSource(w http.ResponseWriter, req *http.Request) error {
	var body struct {
		Source config.SteamSource `json:"source"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid request body"}
	}

	if err := r.Config.SetSteamDataSource(body.Source); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleSetLogging toggles logging
func (r *Router) handleSetLogging(w http.ResponseWriter, req *http.Request) error {
	var body struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid request body"}
	}

	if err := r.Config.SetLoggingEnabled(body.Enabled); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleToggleEmulatorNotification toggles emulator notification
func (r *Router) handleToggleEmulatorNotification(w http.ResponseWriter, req *http.Request) error {
	indexStr := chi.URLParam(req, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid index"}
	}

	if err := r.Config.ToggleEmulatorNotification(index); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleAddPrefix adds a prefix path
func (r *Router) handleAddPrefix(w http.ResponseWriter, req *http.Request) error {
	var body struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid request body"}
	}

	if err := r.Config.AddPrefix(body.Path); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleRemovePrefix removes a prefix path
func (r *Router) handleRemovePrefix(w http.ResponseWriter, req *http.Request) error {
	indexStr := chi.URLParam(req, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid index"}
	}

	if err := r.Config.RemovePrefix(index); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleGetAllGames returns all cached games
func (r *Router) handleGetAllGames(w http.ResponseWriter, req *http.Request) error {
	games, err := r.Steam.LoadAllCachedGameData()
	if err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, games)
}

// handleGetGlobalAchievementPercentages returns global achievement percentages
func (r *Router) handleGetGlobalAchievementPercentages(w http.ResponseWriter, req *http.Request) error {
	id := chi.URLParam(req, "id")

	percentages, err := r.Steam.GetGlobalAchievementPercentages(id)
	if err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, percentages)
}

// handlePlaySound plays a notification sound
func (r *Router) handlePlaySound(w http.ResponseWriter, req *http.Request) error {
	var body struct {
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return AppError{Status: http.StatusBadRequest, Message: "Invalid request body"}
	}

	if err := r.Notifier.PlaySound(body.Filename); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleTestNotification sends a test notification
func (r *Router) handleTestNotification(w http.ResponseWriter, req *http.Request) error {
	if err := r.Notifier.TestNotification(); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleTestNotificationProgress sends a test progress notification
func (r *Router) handleTestNotificationProgress(w http.ResponseWriter, req *http.Request) error {
	if err := r.Notifier.TestNotificationProgress(); err != nil {
		return AppError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	return JSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleNotifications serves as the SSE endpoint for real-time notifications
func (r *Router) handleNotifications(w http.ResponseWriter, req *http.Request) error {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a channel for this client
	//clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	clientID := "sentinel-decky-client"

	notifications := make(chan string, 100)

	// Register this client
	r.Notifier.RegisterClient(clientID, notifications)

	// Close connection when client disconnects
	ctx := req.Context()
	go func() {
		<-ctx.Done()
		r.Notifier.UnregisterClient(clientID)
		close(notifications)
	}()

	// Send notifications to client
	for notification := range notifications {
		fmt.Fprintf(w, "data: %s\n\n", notification)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	return nil
}

// handleServeMedia serves local media files (game images, achievement icons, sounds)
func (r *Router) handleServeMedia(w http.ResponseWriter, req *http.Request) {
	// Get the wildcard path from chi
	relPath := chi.URLParam(req, "*")

	// Sanitize: clean the path to remove any ./ or ../ components
	relPath = filepath.Clean(relPath)

	// Prevent directory traversal attacks
	if strings.Contains(relPath, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Prevent directory listing
	if strings.HasSuffix(relPath, "/") || relPath == "" || relPath == "." {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Construct full path
	fullPath := filepath.Join(backend.DataDir, relPath)

	// Double-check the resolved path is within DataDir (defense in depth)
	absDataDir, _ := filepath.Abs(backend.DataDir)
	absFullPath, _ := filepath.Abs(fullPath)

	if !strings.HasPrefix(absFullPath, absDataDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Serve the file
	http.ServeFile(w, req, fullPath)
}
