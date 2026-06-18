//go:build !decky

package watcher

import (
	"sentinel/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *Service) emitDataUpdated() {
	app := application.Get()
	if app != nil {
		app.Event.Emit(backend.EventDataUpdated)
	}
}
