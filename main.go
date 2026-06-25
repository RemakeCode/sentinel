package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"unicode"

	"sentinel/backend"
	"sentinel/backend/bootstrap"
	"sentinel/backend/config"
	"sentinel/backend/logger"

	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

var startMinimized bool

func init() {
	flag.BoolVar(&startMinimized, "startminimized", false, "Start with window minimized (systray only)")
	application.RegisterEvent[backend.FetchStatusEvt](backend.EventFetchStatus)
	application.RegisterEvent[application.Void](backend.EventDataUpdated)
	application.RegisterEvent[string](backend.EventRefreshGameRequested)
}

func main() {
	flag.Parse()
	var window *application.WebviewWindow

	appLogger, logLevel := bootstrap.ConfigureLogger()
	services := bootstrap.NewServices()

	options := application.Options{
		Name:        "Sentinel",
		Description: "An Achievement Watcher",
		Logger:      appLogger,
		LogLevel:    logger.ParseLevel(logLevel),
		Services: []application.Service{
			application.NewService(services.Config),
			application.NewService(services.Steam),
			application.NewService(services.Ach),
			application.NewService(services.Watcher),
			application.NewService(services.Notifier),
		},

		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/media/") {
					relPath := strings.TrimPrefix(r.URL.Path, "/api/media/")
					fullPath := filepath.Join(backend.DataDir, relPath)
					http.ServeFile(w, r, fullPath)
					return
				}
				application.AssetFileServerFS(assets).ServeHTTP(w, r)
			}),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Linux: application.LinuxOptions{},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.sentinel.app",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if window != nil {
					window.Show()
					window.Focus()
				}
			},
		},
	}

	logger.SetLevel(options.LogLevel)

	app := application.New(options)

	services.Config.SetAutostart(config.NewAutostartManager(app))
	if err := services.Config.SyncAutostart(); err != nil {
		slog.Error("Failed to sync autostart", "error", err)
	}

	gameMenu := application.NewContextMenu("game-card-menu")
	gameMenu.Add("Refresh Metadata").OnClick(func(ctx *application.Context) {
		appID := strings.TrimSpace(ctx.ContextMenuData())
		if !isValidSteamAppID(appID) {
			slog.Warn("Ignoring invalid game context menu data", "appID", appID)
			return
		}

		app.Event.Emit(backend.EventRefreshGameRequested, appID)
	})
	gameMenu.Update()

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      "Sentinel",
		MinWidth:                   1280,
		MinHeight:                  720,
		Width:                      1920,
		Height:                     1080,
		URL:                        "/",
		Hidden:                     startMinimized,
		UseApplicationMenu:         false,
		DefaultContextMenuDisabled: true,
		BackgroundColour:           application.NewRGB(18, 18, 18),
		Linux: application.LinuxWindow{
			WebviewGpuPolicy: application.WebviewGpuPolicyOnDemand,
		},
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("Sentinel")
	tray.SetLabel("Sentinel")

	menu := application.NewMenu()
	showItem := menu.Add("Show")
	showItem.OnClick(func(_ *application.Context) {
		window.Show()
		window.Focus()
	})

	menu.AddSeparator()
	exitItem := menu.Add("Exit")
	exitItem.OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) {
		slog.Info(fmt.Sprintf("%s %s is running", backend.AppName, backend.Version))
	})

	if err := app.Run(); err != nil {
		slog.Error("Application failed", "error", err)
	}
}

func isValidSteamAppID(appID string) bool {
	if appID == "" {
		return false
	}

	for _, r := range appID {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}
