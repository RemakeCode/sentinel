package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/logger"
	"sentinel/backend/notifier"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"

	"net/http"
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
	application.RegisterEvent[application.Void]("sentinel::ready")
	application.RegisterEvent[backend.FetchStatusEvt](backend.EventFetchStatus)
	application.RegisterEvent[application.Void](backend.EventDataUpdated)
}

func main() {
	if runtime.GOOS == "linux" {
		os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

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

	// Initialize services
	configService := &config.File{}
	achService := &ach.Service{}

	steamService := &steam.Service{
		Config: configService,
		Ach:    achService,
	}

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
		Name:        "dev.sentinel.app",
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
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.sentinel.app",
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
		BackgroundColour:           application.NewRGB(18, 18, 18),
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("Sentinel")

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
		app.Event.Emit("sentinel::ready")

		slog.Info(fmt.Sprintf("%s %s is running", backend.AppName, backend.Version))
	})

	if err := app.Run(); err != nil {
		slog.Error("Application failed", "error", err)
	}
}
