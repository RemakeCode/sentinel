package main

import (
	"embed"
	"fmt"
	"log/slog"
	"sentinel/backend"
	"sentinel/backend/config"
	"sentinel/backend/logger"
	"sentinel/backend/notifier"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func init() {
	application.RegisterEvent[application.Void]("sentinel::ready")
	application.RegisterEvent[backend.FetchStatusEvt]("sentinel::fetch-status")
	application.RegisterEvent[application.Void]("sentinel::data-updated")
}

func main() {
	var window *application.WebviewWindow

	appLogger := logger.New()
	slog.SetDefault(appLogger)

	cfg, err := config.Get()
	if err == nil && cfg.DisableLogging {
		logger.SetLevel(slog.Level(100))
	}

	options := application.Options{
		Name:        "sentinel",
		Description: "An Achievement Watcher",
		Logger:      appLogger,
		LogLevel:    slog.LevelInfo,
		Services: []application.Service{
			application.NewService(&config.File{}),
			application.NewService(&steam.Service{}),
			application.NewService(&watcher.Service{}),
			application.NewService(&notifier.Service{}),
		},

		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Linux: application.LinuxOptions{
			ProgramName: "sentinel",
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.sentinel",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				// Bring the existing instance to front when second instance is launched
				if window != nil {
					window.Show()
					window.Focus()
				}
			},
		},
	}

	// Sync slog level with Wails LogLevel option
	logger.SetLevel(options.LogLevel)

	app := application.New(options)

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      "Sentinel",
		MinWidth:                   1280,
		MinHeight:                  720,
		Width:                      1920,
		Height:                     1080,
		URL:                        "/",
		UseApplicationMenu:         false,
		DefaultContextMenuDisabled: false,
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	startFn, endFn := setupSystray(app, window, trayIcon)
	defer endFn()

	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) {
		startFn()
		app.Event.Emit("sentinel::ready")

		slog.Info(fmt.Sprintf("%s %s is running", backend.AppName, backend.Version))
	})

	if err := app.Run(); err != nil {
		slog.Error("Application failed", "error", err)
	}
}
