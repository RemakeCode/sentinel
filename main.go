package main

import (
	"embed"
	"log"
	"net/http"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"sentinel/backend/notifier"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[application.Void]("sentinel::ready")
	application.RegisterEvent[backend.FetchStatusEvt]("sentinel::fetch-status")
	application.RegisterEvent[application.Void]("sentinel::data-updated")
}

func main() {
	app := application.New(application.Options{
		Name:        "Sentinel",
		Description: "Achievement Watcher",
		Services: []application.Service{
			application.NewService(&config.File{}),
			application.NewService(&steam.Service{}),
			application.NewService(&watcher.Service{}),
			application.NewService(&notifier.Service{}),
		},

		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/media/") {
					filename := strings.TrimPrefix(r.URL.Path, "/media/")
					filePath := filepath.Join(backend.MediaDir, filename)
					http.ServeFile(w, r, filePath)
					return
				}
				application.AssetFileServerFS(assets).ServeHTTP(w, r)
			}),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			ProgramName: "Sentinel",
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      "Sentinel",
		MinWidth:                   1280,
		MinHeight:                  720,
		Width:                      1920,
		Height:                     1080,
		URL:                        "/",
		DefaultContextMenuDisabled: false,
	})

	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) {
		app.Logger.Info("Sentinel ready!")
		app.Event.Emit("sentinel::ready")
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

}
