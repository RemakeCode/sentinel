//go:build !decky

package autostart

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func setEnabled(enabled bool) error {
	app := application.Get()
	if app == nil {
		return nil
	}
	if enabled {
		err := app.Autostart.EnableWithOptions(application.AutostartOptions{
			Arguments: []string{"--startminimized"},
		})
		if errors.Is(err, application.ErrAutostartNotSupported) {
			return nil
		}
		return err
	}
	err := app.Autostart.Disable()
	if errors.Is(err, application.ErrAutostartNotSupported) {
		return nil
	}
	return err
}
