//go:build !decky

package config

import (
	"context"
	"errors"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type AutostartManager struct {
	app *application.App
}

func NewAutostartManager(app *application.App) AutostartManager {
	return AutostartManager{app: app}
}

func (m AutostartManager) SetEnabled(enabled bool) error {
	if m.app == nil {
		return nil
	}
	if enabled {
		if err := m.app.Autostart.EnableWithOptions(application.AutostartOptions{
			Arguments: []string{"--startminimized"},
		}); err != nil && !errors.Is(err, application.ErrAutostartNotSupported) {
			return err
		}
		return nil
	}
	if err := m.app.Autostart.Disable(); err != nil && !errors.Is(err, application.ErrAutostartNotSupported) {
		return err
	}
	return nil
}

func (c *File) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	if err := c.Start(ctx); err != nil {
		return err
	}
	slog.Debug("Autostart sync skipped: Wails application handle is unavailable during service startup")
	return nil
}
