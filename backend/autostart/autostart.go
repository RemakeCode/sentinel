//go:build !decky

package autostart

import (
	"context"
	"log/slog"

	"sentinel/backend/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	Config *config.File
}

func NewService(cfg *config.File) *Service {
	return &Service{Config: cfg}
}

func (s *Service) SetEnabled(enabled bool) error {
	if err := s.Config.SetStartOnLogin(enabled); err != nil {
		return err
	}
	return setEnabled(enabled)
}

func (s *Service) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	if err := setEnabled(s.Config.GetStartOnLogin()); err != nil {
		slog.Error("Failed to sync autostart", "error", err)
	}
	return nil
}
