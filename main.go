package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"sentinel/backend"
	"sentinel/backend/ach"
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

var startMinimized bool

func init() {
	flag.BoolVar(&startMinimized, "startminimized", false, "Start with window minimized (systray only)")
	application.RegisterEvent[application.Void]("sentinel::ready")
	application.RegisterEvent[backend.FetchStatusEvt](backend.EventFetchStatus)
	application.RegisterEvent[application.Void](backend.EventDataUpdated)
}

func main() {
	var window *application.WebviewWindow

	appLogger := logger.New()
	// Load config early to check logging preferences
	cfg, err := config.Get()
	logLevel := "info" // default
	if err == nil && cfg != nil {
		logLevel = cfg.LogLevel
		logger.SetLevel(logger.ParseLevel(logLevel))
	}
	slog.SetDefault(appLogger)

	// Initialize services manually to handle dependencies
	configService := &config.File{}
	steamService := &steam.Service{
		Config: configService,
		Ach:    &ach.Service{},
	}
	achService := steamService.Ach // Share the same instance
	notifierService := &notifier.Service{
		Config: configService,
	}
	watcherService := &watcher.Service{
		Steam:    steamService,
		Ach:      achService,
		Config:   configService,
		Notifier: notifierService,
	}

	options := application.Options{
		Name:        "sentinel",
		Description: "An Achievement Watcher",
		Logger:      appLogger,
		LogLevel:    logger.ParseLevel(logLevel),
		Services: []application.Service{
			application.NewService(configService),
			application.NewService(steamService),
			application.NewService(achService),
			application.NewService(watcherService),
			application.NewService(notifierService),
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

	flag.Parse()

	app := application.New(options)

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      "Sentinel",
		MinWidth:                   1280,
		MinHeight:                  720,
		Width:                      1920,
		Height:                     1080,
		URL:                        "/",
		Hidden:                     startMinimized,
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
