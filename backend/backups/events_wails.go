//go:build !decky

package backups

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const EventStatus = "sentinel::backup-status"

func (s *Service) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	s.Start()
	return nil
}

func (s *Service) emit(status Status) {
	if app := application.Get(); app != nil {
		app.Event.Emit(EventStatus, status)
	}
}
