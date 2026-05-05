package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sentinel/backend"
	"sentinel/backend/ach"
	"sentinel/backend/api"
	"sentinel/backend/config"
	"sentinel/backend/decky"
	"sentinel/backend/logger"
	"sentinel/backend/notifier"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"

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
var deckyMode bool

func init() {
	flag.BoolVar(&startMinimized, "startminimized", false, "Start with window minimized (systray only)")
	flag.BoolVar(&deckyMode, "decky", false, "Run in Decky plugin mode (headless with API server)")
	application.RegisterEvent[application.Void]("sentinel::ready")
	application.RegisterEvent[backend.FetchStatusEvt](backend.EventFetchStatus)
	application.RegisterEvent[application.Void](backend.EventDataUpdated)
}

func main() {
	if runtime.GOOS == "linux" {
		os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

	flag.Parse()
	var window *application.WebviewWindow

	appLogger := logger.New()
	cfg, err := config.Get()
	logLevel := "info"
	if err == nil && cfg != nil {
		logLevel = cfg.LogLevel
		logger.SetLevel(logger.ParseLevel(logLevel))
	}

	slog.SetDefault(appLogger)

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

	if deckyMode || os.Getenv("DECKY_MODE") == "true" {
		// Initialize services in the correct order for decky mode
		ctx := context.Background()
		options := application.DefaultServiceOptions

		slog.Info("slogger")
		// Initialize config service first
		if err := configService.ServiceStartup(ctx, options); err != nil {
			slog.Error("Failed to initialize config service", "error", err)
		}

		// Initialize steam service (depends on config)
		if err := steamService.ServiceStartup(ctx, options); err != nil {
			slog.Error("Failed to initialize steam service", "error", err)

		}

		// Initialize ach service
		if err := achService.ServiceStartup(ctx, options); err != nil {
			slog.Error("Failed to initialize ach service", "error", err)

		}

		// Initialize watcher service (depends on config)
		if err := watcherService.ServiceStartup(ctx, options); err != nil {
			slog.Error("Failed to initialize watcher service", "error", err)
		}

		// Initialize notifier service (depends on config)
		if err := notifierService.ServiceStartup(ctx, options); err != nil {
			slog.Error("Failed to initialize notifier service", "error", err)
		}

		go func() {
			router := api.NewRouter(configService, steamService, watcherService, notifierService)
			port := decky.GetPort()
			slog.Info(fmt.Sprintf("Decky API Server on 127.0.0.1:%d", port))

			slog.Info(fmt.Sprintf("-----isDecky:%t", decky.IsDecky()))

			if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), router.Handler()); err != nil {
				slog.Error("Decky API Server failed", "error", err)
			}
		}()
		select {}
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

	if !deckyMode {
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
}
