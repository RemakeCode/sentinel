//go:build !decky

package updater

import (
	"context"
	"log/slog"
	"runtime"

	"sentinel/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
	wup "github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	return s.Start(ctx)
}

// Start configures the application-owned desktop updater.
//
//wails:internal
func (s *Service) Start(ctx context.Context) error {
	app := application.Get()
	s.engine = app.Updater
	s.ctx = ctx

	manifest := "update.json"
	if backend.IsFlatpak() {
		manifest = "update-flatpak.json"
	}

	provider, err := endpoint.New(endpoint.Config{
		URL:     "https://github.com/RemakeCode/sentinel/releases/latest/download/" + manifest,
		Channel: "stable",
	})
	if err != nil {
		slog.Error("Failed to configure update endpoint", "error", err)
		return nil
	}

	err = app.Updater.Init(wup.Config{
		CurrentVersion: backend.Version,
		Providers:      []wup.Provider{provider},
		Platform:       runtime.GOOS,
		Arch:           runtime.GOARCH,
		Window:         wup.WindowNone,
	})
	if err != nil {
		slog.Error("Failed to initialize desktop updater", "error", err)
		return nil
	}

	return nil
}
