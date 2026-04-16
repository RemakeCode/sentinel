package api

import (
	"encoding/json"
	"net/http"
	"sentinel/backend/config"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Router struct {
	Config  *config.File
	Steam   *steam.Service
	Watcher *watcher.Service
}

func NewRouter(c *config.File, s *steam.Service, w *watcher.Service) *Router {
	return &Router{Config: c, Steam: s, Watcher: w}
}

// Handler returns a fully configured chi router as an http.Handler
func (r *Router) Handler() http.Handler {
	router := chi.NewRouter()

	// 1. Setup CORS (Essential for Decky Loader plugins)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 2. Standard Middlewares
	router.Use(middleware.Logger)    // Logs requests to terminal
	router.Use(middleware.Recoverer) // Prevents app from crashing on API panics

	// 3. Define Routes
	router.Route("/api", func(api chi.Router) {
		api.Post("/config/notification-sound", r.handleSetSound)

		// You can group routes easily
		api.Get("/steam/status", func(w http.ResponseWriter, req *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"status": "online"})
		})
	})

	return router
}

func (r *Router) handleSetSound(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Sound string `json:"sound"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	if err := r.Config.SetNotificationSound(body.Sound); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
